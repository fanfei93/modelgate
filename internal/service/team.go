package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/modelgate/internal/model"
	"github.com/modelgate/internal/newapi"
	"github.com/modelgate/internal/repository"
	"gorm.io/gorm"
)

// defaultQuotaPerUnit new-api 默认的 QuotaPerUnit 值（$1 = 500,000 tokens）
// 实际值会通过 /api/status 接口动态获取
const defaultQuotaPerUnit = 500_000.0

var (
	ErrTeamNotFound          = errors.New("团队不存在")
	ErrNotTeamOwner          = errors.New("仅团队拥有者可操作")
	ErrAlreadyMember         = errors.New("已是团队成员")
	ErrNotMember             = errors.New("不是团队成员")
	ErrTeamSlugExists        = errors.New("团队标识已被占用")
	ErrInsufficientBalance   = errors.New("团队额度不足，无法分配")
)

type TeamService struct {
	teamRepo       *repository.TeamRepo
	memberRepo     *repository.MemberRepo
	userRepo       *repository.UserRepo
	invitationRepo *repository.InvitationRepo
	quotaAllocRepo *repository.QuotaAllocationRepo
	newAPIClient   *newapi.Client
	emailService   *EmailService
	baseURL        string // 前端地址，用于生成邀请链接
	db             *gorm.DB

	// quotaPerUnit new-api 的 token/美元 换算因子，从 /api/status 动态获取
	// quotaTokensPerCent = quotaPerUnit / 100，即 modelgate "分" 对应的 token 数
	quotaPerUnit float64
}

// QuotaTokensPerCent 返回 modelgate "分" → new-api token 的换算因子
func (s *TeamService) quotaTokensPerCent() int64 {
	if s.quotaPerUnit <= 0 {
		return int64(defaultQuotaPerUnit / 100) // fallback: 5000
	}
	return int64(s.quotaPerUnit / 100)
}

// InitQuotaPerUnit 从 new-api /api/status 接口获取 QuotaPerUnit 值
func (s *TeamService) InitQuotaPerUnit() {
	status, err := s.newAPIClient.GetStatus()
	if err != nil {
		log.Printf("[WARN] 获取 new-api QuotaPerUnit 失败，使用默认值 %.0f: %v", defaultQuotaPerUnit, err)
		s.quotaPerUnit = defaultQuotaPerUnit
		return
	}
	if status.QuotaPerUnit <= 0 {
		log.Printf("[WARN] new-api 返回的 QuotaPerUnit 无效 (%.0f)，使用默认值 %.0f", status.QuotaPerUnit, defaultQuotaPerUnit)
		s.quotaPerUnit = defaultQuotaPerUnit
		return
	}
	s.quotaPerUnit = status.QuotaPerUnit
	log.Printf("[INFO] 从 new-api 获取 QuotaPerUnit = %.0f，换算因子 quotaTokensPerCent = %d", s.quotaPerUnit, s.quotaTokensPerCent())
}

func NewTeamService(
	db *gorm.DB,
	teamRepo *repository.TeamRepo,
	memberRepo *repository.MemberRepo,
	userRepo *repository.UserRepo,
	invitationRepo *repository.InvitationRepo,
	quotaAllocRepo *repository.QuotaAllocationRepo,
	newAPIClient *newapi.Client,
	emailService *EmailService,
	baseURL string,
) *TeamService {
	return &TeamService{
		db:              db,
		teamRepo:        teamRepo,
		memberRepo:      memberRepo,
		userRepo:        userRepo,
		invitationRepo:  invitationRepo,
		quotaAllocRepo:  quotaAllocRepo,
		newAPIClient:    newAPIClient,
		emailService:    emailService,
		baseURL:         baseURL,
	}
}

// CreateTeam 创建团队，会在 new-api 中创建对应的内部用户
// slug 由服务端根据 name 自动生成，无需客户端传入
func (s *TeamService) CreateTeam(ownerID uint, name string) (*model.Team, error) {
	slug := generateSlug(name)

	// 检查 slug 唯一性，冲突时追加随机后缀
	for i := 0; i < 10; i++ {
		if _, err := s.teamRepo.FindBySlugLight(nil, slug); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break // slug 可用
			}
			return nil, err
		}
		// 冲突重试
		slug = generateSlug(name)
		if i == 9 {
			return nil, errors.New("无法生成唯一标识，请重试")
		}
	}

	// 在 new-api 中创建内部用户（用于团队级别的计费，不作为 API Key 暴露）
	// new-api 约束: username max=20, password min=8 max=20, display_name max=20
	username := slug
	if len(username) > 20 {
		username = username[:20]
	}
	password := generateRandomPassword(16)
	newAPIEmail := fmt.Sprintf("%s@modelgate.local", slug)
	newAPIUserID, err := s.newAPIClient.RegisterUserWithUsername(username, newAPIEmail, password)
	if err != nil {
		return nil, fmt.Errorf("创建内部用户失败: %w", err)
	}

	team := &model.Team{
		Name:           name,
		Slug:           slug,
		NewAPIUserID:   newAPIUserID,
		NewAPIPassword: password,
		OwnerID:        ownerID,
		Balance:        0,
		Status:         "active",
	}

	tx := s.db.Begin()

	if err := s.teamRepo.Create(tx, team); err != nil {
		tx.Rollback()
		return nil, err
	}

	// 创建 owner 成员记录
	member := &model.TeamMember{
		TeamID:   team.ID,
		UserID:   ownerID,
		Role:     "owner",
		JoinedAt: time.Now(),
	}
	if err := s.memberRepo.Create(tx, member); err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	return team, nil
}

// GetUserTeams 获取用户所在的所有团队
func (s *TeamService) GetUserTeams(userID uint) ([]model.Team, error) {
	return s.teamRepo.FindByUserID(nil, userID)
}

// GetTeamBySlug 根据 slug 获取团队详情
func (s *TeamService) GetTeamBySlug(slug string) (*model.Team, error) {
	team, err := s.teamRepo.FindBySlug(nil, slug)
	if err != nil {
		return nil, ErrTeamNotFound
	}
	// 从 new-api 同步团队总额度
	if syncErr := s.SyncTeamQuota(team); syncErr != nil {
		log.Printf("[WARN] 同步团队 %s 额度失败: %v", slug, syncErr)
		// 使用本地缓存值，不阻塞
	}
	// 从 new-api 同步各成员额度并脱敏 API Key
	for i := range team.Members {
		member := &team.Members[i]
		if member.NewAPITokenID > 0 {
			if tokenInfo, tokErr := s.newAPIClient.AdminGetTokenInfo(member.NewAPITokenID); tokErr == nil {
			member.QuotaAllocated = int64(tokenInfo.RemainQuota+tokenInfo.UsedQuota) / s.quotaTokensPerCent()
			member.QuotaUsed = int64(tokenInfo.UsedQuota) / s.quotaTokensPerCent()
				_ = s.memberRepo.Update(nil, member)
			} else {
				log.Printf("[WARN] 同步成员 %d (token=%d) 额度失败: %v", member.ID, member.NewAPITokenID, tokErr)
			}
		}
		team.Members[i].NewAPIKeyMask = maskAPIKey(team.Members[i].NewAPIKey)
	}
	return team, nil
}

// DeleteTeam 解散团队（仅 owner）
func (s *TeamService) DeleteTeam(userID uint, slug string) error {
	team, err := s.teamRepo.FindBySlugLight(nil, slug)
	if err != nil {
		return ErrTeamNotFound
	}
	if team.OwnerID != userID {
		return ErrNotTeamOwner
	}

	tx := s.db.Begin()
	if err := s.teamRepo.Delete(tx, team.ID); err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

// SyncTeamQuota 从 new-api 同步团队总额度到本地 Balance
func (s *TeamService) SyncTeamQuota(team *model.Team) error {
	userInfo, err := s.newAPIClient.GetUserInfo(team.NewAPIUserID)
	if err != nil {
		return fmt.Errorf("获取团队额度失败: %w", err)
	}
	team.Balance = userInfo.Quota / s.quotaTokensPerCent()
	return nil
}

// CreateMemberAPIKey 为当前成员创建个人 API Key
// 在团队的 new-api 用户下创建一个独立 token，每个成员拥有独立的 Key，共享团队额度
func (s *TeamService) CreateMemberAPIKey(userID uint, slug string) (string, error) {
	team, err := s.teamRepo.FindBySlugLight(nil, slug)
	if err != nil {
		return "", ErrTeamNotFound
	}
	member, err := s.memberRepo.FindByTeamAndUser(nil, team.ID, userID)
	if err != nil {
		return "", ErrNotMember
	}

	// 获取成员用户名作为 token 名称
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return "", fmt.Errorf("获取用户信息失败: %w", err)
	}
	tokenName := fmt.Sprintf("Member: %s", user.Username)

	// 先删除旧 token（如果存在）
	if member.NewAPITokenID > 0 {
		if err := s.newAPIClient.AdminDeleteToken(member.NewAPITokenID); err != nil {
			log.Printf("[WARN] 删除旧 token (id=%d) 失败: %v", member.NewAPITokenID, err)
		}
	}

	// 使用 admin key 在团队 new-api 用户下创建新 token
	tokenID, key, err := s.newAPIClient.AdminCreateToken(team.NewAPIUserID, tokenName)
	if err != nil {
		return "", fmt.Errorf("创建 API Key 失败: %w", err)
	}

	member.NewAPITokenID = tokenID
	member.NewAPIKey = key
	member.NewAPIKeyStatus = 1 // 启用
	if err := s.memberRepo.Update(nil, member); err != nil {
		log.Printf("[WARN] 更新成员 API key 失败: %v", err)
	}

	return key, nil
}

// GetMemberKey 获取当前成员已有的 API Key（完整密钥，可重复查看）
func (s *TeamService) GetMemberKey(userID uint, slug string) (string, error) {
	team, err := s.teamRepo.FindBySlugLight(nil, slug)
	if err != nil {
		return "", ErrTeamNotFound
	}
	member, err := s.memberRepo.FindByTeamAndUser(nil, team.ID, userID)
	if err != nil {
		return "", ErrNotMember
	}
	if member.NewAPIKey == "" {
		return "", fmt.Errorf("尚未创建 API Key")
	}
	return member.NewAPIKey, nil
}

// ToggleMemberKey 切换当前成员的 API Key 状态（启用/禁用）
func (s *TeamService) ToggleMemberKey(userID uint, slug string) (int, error) {
	team, err := s.teamRepo.FindBySlugLight(nil, slug)
	if err != nil {
		return 0, ErrTeamNotFound
	}
	member, err := s.memberRepo.FindByTeamAndUser(nil, team.ID, userID)
	if err != nil {
		return 0, ErrNotMember
	}
	if member.NewAPIKey == "" {
		return 0, fmt.Errorf("尚未创建 API Key")
	}

	// 切换状态：1(启用) -> 2(禁用), 2(禁用) -> 1(启用)，0(未创建)不支持
	newStatus := 2
	if member.NewAPIKeyStatus == 2 {
		newStatus = 1
	}

	if member.NewAPITokenID > 0 {
		if err := s.newAPIClient.AdminUpdateTokenStatus(member.NewAPITokenID, newStatus); err != nil {
			log.Printf("[WARN] 更新 new-api token (id=%d) 状态失败: %v", member.NewAPITokenID, err)
			return 0, fmt.Errorf("更新 Key 状态失败")
		}
	}

	member.NewAPIKeyStatus = newStatus
	if err := s.memberRepo.Update(nil, member); err != nil {
		return 0, err
	}

	return newStatus, nil
}

// APIKeyInfo 用户 API Key 摘要信息
type APIKeyInfo struct {
	TeamID     uint   `json:"team_id"`
	TeamName   string `json:"team_name"`
	TeamSlug   string `json:"team_slug"`
	Role       string `json:"role"`
	HasKey     bool   `json:"has_key"`
	KeyStatus  int    `json:"key_status"`  // 0: 未创建, 1: 启用, 2: 禁用
	APIKeyMask string `json:"api_key_mask"`
}

// GetUserAPIKeys 获取用户在所有团队中的 API Key 信息
func (s *TeamService) GetUserAPIKeys(userID uint) ([]APIKeyInfo, error) {
	members, err := s.memberRepo.FindByUserID(nil, userID)
	if err != nil {
		return nil, err
	}

	result := make([]APIKeyInfo, 0, len(members))
	for _, m := range members {
		team, err := s.teamRepo.FindByID(nil, m.TeamID)
		if err != nil {
			continue
		}
		result = append(result, APIKeyInfo{
			TeamID:     team.ID,
			TeamName:   team.Name,
			TeamSlug:   team.Slug,
			Role:       m.Role,
			HasKey:     m.NewAPIKey != "",
			KeyStatus:  m.NewAPIKeyStatus,
			APIKeyMask: maskAPIKey(m.NewAPIKey),
		})
	}
	return result, nil
}

// --- 成员管理 ---

// MemberEntry 批量添加成员时的单条记录
type MemberEntry struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// AddMembers 批量添加团队成员（仅 owner）
// 已注册用户直接加入；未注册用户创建邀请并发送邮件，注册后自动成为成员
// 返回成功数量（直接加入+已发送邀请）、失败列表
func (s *TeamService) AddMembers(ownerID uint, slug string, entries []MemberEntry) (int, []string, error) {
	team, err := s.teamRepo.FindBySlugLight(nil, slug)
	if err != nil {
		return 0, nil, ErrTeamNotFound
	}
	if team.OwnerID != ownerID {
		return 0, nil, ErrNotTeamOwner
	}

	canSendEmail := s.emailService != nil && s.emailService.IsConfigured()
	added := 0
	var failed []string

	for _, entry := range entries {
		email := strings.TrimSpace(entry.Email)
		if email == "" {
			continue
		}
		name := strings.TrimSpace(entry.Name)
		displayLabel := email
		if name != "" {
			displayLabel = fmt.Sprintf("%s (%s)", name, email)
		}

		user, err := s.userRepo.FindByEmail(email)
		if err != nil {
			// 用户未注册 → 创建邀请
			if _, invErr := s.invitationRepo.FindByTeamAndEmail(team.ID, email); invErr == nil {
				failed = append(failed, fmt.Sprintf("%s: 已发送过邀请", displayLabel))
				continue
			}
			inv := &model.TeamInvitation{
				TeamID:    team.ID,
				Email:     email,
				Name:      name,
				InviterID: ownerID,
				Status:    "pending",
			}
			if err := s.invitationRepo.Create(inv); err != nil {
				failed = append(failed, fmt.Sprintf("%s: 发送邀请失败", displayLabel))
				continue
			}

			// 发送邀请邮件
			if canSendEmail {
				registerURL := fmt.Sprintf("%s/register?email=%s", s.baseURL, url.QueryEscape(email))
				if emailErr := s.emailService.SendInvitationEmail(email, team.Name, registerURL); emailErr != nil {
					log.Printf("[WARN] 发送邀请邮件失败 %s: %v", email, emailErr)
				}
			}

			added++ // 邀请也算"成功"
			continue
		}

		// 用户已注册 → 直接加入
		// 检查是否已是成员
		if _, err := s.memberRepo.FindByTeamAndUser(nil, team.ID, user.ID); err == nil {
			failed = append(failed, fmt.Sprintf("%s: 已是团队成员", displayLabel))
			continue
		}

		// 如果存在同名邀请，先标记为 accepted
		if inv, invErr := s.invitationRepo.FindByTeamAndEmail(team.ID, email); invErr == nil {
			_ = s.invitationRepo.UpdateStatus(inv.ID, "accepted")
		}

		member := &model.TeamMember{
			TeamID:   team.ID,
			UserID:   user.ID,
			Role:     "member",
			JoinedAt: time.Now(),
		}
		if err := s.memberRepo.Create(nil, member); err != nil {
			failed = append(failed, fmt.Sprintf("%s: 添加失败", displayLabel))
			continue
		}
		added++
	}

	return added, failed, nil
}

// ProcessInvitations 处理用户注册后的待处理邀请
// 将该邮箱的所有 pending 邀请转为正式成员
func (s *TeamService) ProcessInvitations(email string, userID uint) {
	invs, err := s.invitationRepo.FindByEmail(email)
	if err != nil {
		return
	}
	for _, inv := range invs {
		// 检查是否已是成员（防止重复）
		if _, err := s.memberRepo.FindByTeamAndUser(nil, inv.TeamID, userID); err == nil {
			_ = s.invitationRepo.UpdateStatus(inv.ID, "accepted")
			continue
		}
		member := &model.TeamMember{
			TeamID:   inv.TeamID,
			UserID:   userID,
			Role:     "member",
			JoinedAt: time.Now(),
		}
		if err := s.memberRepo.Create(nil, member); err != nil {
			log.Printf("[WARN] 处理邀请失败 invitation=%d team=%d user=%d: %v", inv.ID, inv.TeamID, userID, err)
			continue
		}
		_ = s.invitationRepo.UpdateStatus(inv.ID, "accepted")
		log.Printf("[INFO] 邀请已接受: %s 加入团队 %d", email, inv.TeamID)
	}
}

// CancelInvitation 取消待处理邀请（仅 owner）
func (s *TeamService) CancelInvitation(ownerID uint, slug string, invitationID uint) error {
	team, err := s.teamRepo.FindBySlugLight(nil, slug)
	if err != nil {
		return ErrTeamNotFound
	}
	if team.OwnerID != ownerID {
		return ErrNotTeamOwner
	}
	return s.invitationRepo.DeleteByID(invitationID)
}

// RemoveMember 移除团队成员（仅 owner）
func (s *TeamService) RemoveMember(ownerID uint, slug string, memberID uint) error {
	team, err := s.teamRepo.FindBySlugLight(nil, slug)
	if err != nil {
		return ErrTeamNotFound
	}
	if team.OwnerID != ownerID {
		return ErrNotTeamOwner
	}

	member, err := s.memberRepo.FindByID(nil, memberID)
	if err != nil || member.TeamID != team.ID {
		return ErrNotMember
	}
	if member.Role == "owner" {
		return errors.New("无法移除团队拥有者")
	}

	return s.memberRepo.DeleteByID(nil, memberID)
}

// --- 成员额度管理 ---

// QuotaInfo 成员额度信息
type QuotaInfo struct {
	QuotaAllocated int64 `json:"quota_allocated"` // 已分配额度
	QuotaUsed      int64 `json:"quota_used"`      // 已使用额度
	QuotaRemain    int64 `json:"quota_remain"`    // 剩余额度（new-api token 的 remain_quota）
	HasKey         bool  `json:"has_key"`         // 是否已创建 API Key
}

// GetMemberQuotaInfo 获取成员额度信息（从 new-api 同步 token 配额）
func (s *TeamService) GetMemberQuotaInfo(slug string, memberID uint) (*QuotaInfo, error) {
	team, err := s.teamRepo.FindBySlugLight(nil, slug)
	if err != nil {
		return nil, ErrTeamNotFound
	}

	member, err := s.memberRepo.FindByID(nil, memberID)
	if err != nil || member.TeamID != team.ID {
		return nil, ErrNotMember
	}

	info := &QuotaInfo{
		QuotaAllocated: member.QuotaAllocated,
		QuotaUsed:      member.QuotaUsed,
		QuotaRemain:    0,
		HasKey:         member.NewAPIKey != "",
	}

	// 如果有 key，从 new-api 同步最新配额数据
	if member.NewAPITokenID > 0 {
		tokenInfo, err := s.newAPIClient.AdminGetTokenInfo(member.NewAPITokenID)
		if err != nil {
			log.Printf("[WARN] 获取 token 信息失败 member=%d token=%d: %v", memberID, member.NewAPITokenID, err)
			// 返回本地缓存的金额
			return info, nil
		}
		info.QuotaRemain = int64(tokenInfo.RemainQuota) / s.quotaTokensPerCent()
		info.QuotaUsed = int64(tokenInfo.UsedQuota) / s.quotaTokensPerCent()

		// 更新本地缓存
		member.QuotaAllocated = (int64(tokenInfo.RemainQuota) + int64(tokenInfo.UsedQuota)) / s.quotaTokensPerCent()
		member.QuotaUsed = int64(tokenInfo.UsedQuota) / s.quotaTokensPerCent()
		_ = s.memberRepo.Update(nil, member)
	}

	return info, nil
}

// SetMemberQuota 设置成员额度（仅 owner）
// amount 为分单位，与充值单位一致
func (s *TeamService) SetMemberQuota(ownerID uint, slug string, memberID uint, amount int64) error {
	team, err := s.teamRepo.FindBySlugLight(nil, slug)
	if err != nil {
		return ErrTeamNotFound
	}
	if team.OwnerID != ownerID {
		return ErrNotTeamOwner
	}

	member, err := s.memberRepo.FindByID(nil, memberID)
	if err != nil || member.TeamID != team.ID {
		return ErrNotMember
	}

	// 同步团队最新额度
	if syncErr := s.SyncTeamQuota(team); syncErr != nil {
		log.Printf("[WARN] 同步团队 %s 额度失败: %v", slug, syncErr)
	}

	// 校验团队额度是否充足
	othersAllocated, err := s.memberRepo.SumQuotaAllocatedExcept(nil, team.ID, memberID)
	if err != nil {
		return fmt.Errorf("校验团队额度失败: %w", err)
	}
	newTotalAllocated := othersAllocated + amount
	if newTotalAllocated > team.Balance {
		return fmt.Errorf("%w: 团队总余额 ¥%.2f，其他成员已分配 ¥%.2f，本次分配 ¥%.2f 超出可用额度",
			ErrInsufficientBalance,
			float64(team.Balance)/100,
			float64(othersAllocated)/100,
			float64(amount)/100,
		)
	}

	// 如果成员还没有 API Key，先创建
	if member.NewAPIKey == "" || member.NewAPITokenID <= 0 {
		user, err := s.userRepo.FindByID(member.UserID)
		if err != nil {
			return fmt.Errorf("获取用户信息失败: %w", err)
		}
		tokenName := fmt.Sprintf("Member: %s", user.Username)

		q := int(amount * s.quotaTokensPerCent())
		tokenID, key, err := s.newAPIClient.AdminCreateTokenWithQuota(
			team.NewAPIUserID, tokenName, &q, nil,
		)
		if err != nil {
			return fmt.Errorf("创建 API Key 失败: %w", err)
		}

		member.NewAPITokenID = tokenID
		member.NewAPIKey = key
		member.NewAPIKeyStatus = 1
	} else {
		// 更新现有 token 的配额
		q := int(amount * s.quotaTokensPerCent())
		if err := s.newAPIClient.AdminUpdateTokenQuota(member.NewAPITokenID, &q, nil); err != nil {
			return fmt.Errorf("更新配额失败: %w", err)
		}
	}

	// 计算差额：新分配 - 旧分配
	member.QuotaAllocated = amount
	if err := s.memberRepo.Update(nil, member); err != nil {
		log.Printf("[WARN] 更新成员配额缓存失败: %v", err)
	}

	// 记录分配日志
	alloc := &model.QuotaAllocation{
		TeamID:     team.ID,
		MemberID:   member.ID,
		OperatorID: ownerID,
		Type:       "allocate",
		Amount:     amount,
	}
	_ = s.quotaAllocRepo.Create(alloc)

	return nil
}

// RevokeMemberQuota 回收成员的额度（仅 owner）
func (s *TeamService) RevokeMemberQuota(ownerID uint, slug string, memberID uint) error {
	return s.SetMemberQuota(ownerID, slug, memberID, 0)
}

// GetMemberLogs 获取当前成员的调用日志（使用成员自己的 API Key）
// 返回的 LogItem.Quota 已从 new-api 内部配额点换算为 modelgate "分"（cents）
func (s *TeamService) GetMemberLogs(userID uint, slug string) ([]newapi.LogItem, error) {
	team, err := s.teamRepo.FindBySlugLight(nil, slug)
	if err != nil {
		return nil, ErrTeamNotFound
	}
	member, err := s.memberRepo.FindByTeamAndUser(nil, team.ID, userID)
	if err != nil {
		return nil, ErrNotMember
	}
	if member.NewAPIKey == "" {
		return []newapi.LogItem{}, nil
	}

	logs, err := s.newAPIClient.GetLogsByToken(member.NewAPIKey)
	if err != nil {
		return nil, fmt.Errorf("获取日志失败: %w", err)
	}
	if logs == nil {
		logs = []newapi.LogItem{}
	}

	// 将 new-api 的 quota（内部配额点/tokens）换算为 modelgate "分"（cents）
	// 1 分 = quotaTokensPerCent() tokens
	qpc := s.quotaTokensPerCent()
	for i := range logs {
		logs[i].Quota = logs[i].Quota / int(qpc)
	}

	return logs, nil
}

func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

func maskAPIKey(key string) string {
	if len(key) <= 10 {
		return key
	}
	return key[:5] + "****" + key[len(key)-4:]
}

// nonAlphaRegex 匹配非字母数字的字符
var nonAlphaRegex = regexp.MustCompile(`[^a-z0-9]+`)

// generateSlug 根据团队名称自动生成 URL 友好标识
// 中文等非 ASCII 字符会使用随机短哈希替代
func generateSlug(name string) string {
	// 转小写，只保留 ASCII 字母和数字
	var buf strings.Builder
	hasASCII := false
	for _, r := range strings.ToLower(name) {
		if ('a' <= r && r <= 'z') || ('0' <= r && r <= '9') {
			buf.WriteRune(r)
			hasASCII = true
		} else if r == ' ' || r == '-' || r == '_' {
			buf.WriteRune('-')
		} else if r < 128 {
			// 其他 ASCII 可打印字符，跳过
			continue
		} else {
			// 非 ASCII 字符（如中文），跳过
			continue
		}
	}

	raw := strings.Trim(buf.String(), "-")

	// 如果没有 ASCII 内容（如纯中文名），用随机后缀
	if !hasASCII || len(raw) == 0 {
		raw = "team-" + randomSuffix(6)
	} else {
		// 归一化连字符
		raw = nonAlphaRegex.ReplaceAllString(raw, "-")
		raw = strings.Trim(raw, "-")
		if len(raw) > 40 {
			raw = raw[:40]
		}
		raw = strings.Trim(raw, "-")
	}

	return raw
}

// randomSuffix 生成指定长度的随机小写字母数字后缀
func randomSuffix(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}
