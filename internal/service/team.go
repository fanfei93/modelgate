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
	ErrQuotaBelowUsed        = errors.New("设置的额度不能低于已使用的额度")
	ErrMaxTeamLimit          = errors.New("每个用户只能加入或创建一个团队")
)

type TeamService struct {
	teamRepo        *repository.TeamRepo
	memberRepo      *repository.MemberRepo
	userRepo        *repository.UserRepo
	userAPIKeyRepo  *repository.UserAPIKeyRepo
	invitationRepo  *repository.InvitationRepo
	quotaAllocRepo  *repository.QuotaAllocationRepo
	newAPIClient    *newapi.Client
	emailService    *EmailService
	baseURL         string // 前端地址，用于生成邀请链接
	db              *gorm.DB

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
	userAPIKeyRepo *repository.UserAPIKeyRepo,
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
		userAPIKeyRepo:  userAPIKeyRepo,
		invitationRepo:  invitationRepo,
		quotaAllocRepo:  quotaAllocRepo,
		newAPIClient:    newAPIClient,
		emailService:    emailService,
		baseURL:         baseURL,
	}
}

// CreateTeam 创建团队
// 用户只能创建一个团队（或加入一个团队）
func (s *TeamService) CreateTeam(ownerID uint, name string) (*model.Team, error) {
	// 检查用户是否已在某个团队中（只能加入/创建一个团队）
	count, err := s.memberRepo.CountByUserID(nil, ownerID)
	if err != nil {
		return nil, fmt.Errorf("检查团队数量失败: %w", err)
	}
	if count > 0 {
		return nil, ErrMaxTeamLimit
	}

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

	team := &model.Team{
		Name:    name,
		Slug:    slug,
		OwnerID: ownerID,
		Balance: 0,
		Status:  "active",
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
	// 脱敏 API Key
	for i := range team.Members {
		team.Members[i].NewAPIKeyMask = maskAPIKey(team.Members[i].NewAPIKey)
	}

	// 从 new-api 实时同步各成员的 quota_used
	s.syncMembersQuotaUsed(team)

	return team, nil
}

// syncMembersQuotaUsed 并发从 new-api 获取各成员的真实已用额度，更新到 team.Members
func (s *TeamService) syncMembersQuotaUsed(team *model.Team) {
	// 收集需要同步的成员：有分配额度的
	type memberSync struct {
		index          int
		newAPIUserID  int
		userID        uint
	}
	var toSync []memberSync
	for i, m := range team.Members {
		if m.QuotaAllocated <= 0 || m.User == nil || m.User.NewAPIUserID <= 0 {
			continue
		}
		toSync = append(toSync, memberSync{
			index:         i,
			newAPIUserID:  m.User.NewAPIUserID,
			userID:        m.UserID,
		})
	}

	if len(toSync) == 0 {
		return
	}

	// 并发获取
	type syncResult struct {
		index   int
		used    int64
		success bool
	}
	ch := make(chan syncResult, len(toSync))
	for _, ms := range toSync {
		go func(ms memberSync) {
			userInfo, err := s.newAPIClient.GetUserInfo(ms.newAPIUserID)
			if err != nil {
				ch <- syncResult{index: ms.index}
				return
			}
			remainCents := int64(userInfo.Quota) / s.quotaTokensPerCent()
			// used = 该用户所有团队分配总额 - 剩余额度
			totalAllocated, _ := s.memberRepo.SumQuotaAllocatedByUserID(nil, ms.userID)
			usedCents := totalAllocated - remainCents
			if usedCents < 0 {
				usedCents = 0
			}
			ch <- syncResult{index: ms.index, used: usedCents, success: true}
		}(ms)
	}

	for range toSync {
		r := <-ch
		if r.success {
			team.Members[r.index].QuotaUsed = r.used
		}
	}
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

	// 清理所有成员在 new-api 中的 token
	members, err := s.memberRepo.FindByTeamID(nil, team.ID)
	if err == nil {
		for _, m := range members {
			if m.NewAPITokenID > 0 {
				if delErr := s.newAPIClient.AdminDeleteToken(m.NewAPITokenID); delErr != nil {
					log.Printf("[WARN] 删除成员 token (id=%d) 失败: %v", m.NewAPITokenID, delErr)
				}
			}
		}
	}

	tx := s.db.Begin()
	if err := s.teamRepo.Delete(tx, team.ID); err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

// SyncTeamQuota [已废弃] 团队余额现由 modelgate 本地管理，不再从 new-api 同步
func (s *TeamService) SyncTeamQuota(team *model.Team) error {
	// no-op: 团队余额由超管充值，本地管理
	return nil
}

// CreateMemberAPIKey 为当前成员创建个人 API Key（遗留接口）
// 在成员自己的 new-api 用户下创建 token，设置 unlimited_quota=true
// 计费由 user.Quota 控制（通过 syncUserQuota 同步）
func (s *TeamService) CreateMemberAPIKey(userID uint, slug string) (string, error) {
	team, err := s.teamRepo.FindBySlugLight(nil, slug)
	if err != nil {
		return "", ErrTeamNotFound
	}
	member, err := s.memberRepo.FindByTeamAndUser(nil, team.ID, userID)
	if err != nil {
		return "", ErrNotMember
	}

	// 获取成员信息
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return "", fmt.Errorf("获取用户信息失败: %w", err)
	}
	if user.NewAPIUserID <= 0 {
		return "", fmt.Errorf("用户的 API 账户尚未创建，请联系管理员")
	}

	// 成员必须已被分配额度
	if member.QuotaAllocated <= 0 {
		return "", fmt.Errorf("尚未分配额度，请联系团队 Owner 先分配额度后再创建 API Key")
	}

	// 如果已有旧 token，删除
	if member.NewAPITokenID > 0 {
		if err := s.newAPIClient.AdminDeleteToken(member.NewAPITokenID); err != nil {
			log.Printf("[WARN] 删除旧 token (id=%d) 失败: %v", member.NewAPITokenID, err)
		}
	}

	tokenName := fmt.Sprintf("mg_%s_%s", team.Slug, user.Username)
	if len(tokenName) > 30 {
		tokenName = tokenName[:30]
	}

	// 创建 unlimited_quota=true 的 token（计费由 user.Quota 控制）
	unlimited := true
	tokenID, key, err := s.newAPIClient.AdminCreateTokenWithQuota(
		user.NewAPIUserID, tokenName, nil, &unlimited,
	)
	if err != nil {
		return "", fmt.Errorf("创建 API Key 失败: %w", err)
	}

	member.NewAPITokenID = tokenID
	member.NewAPIKey = key
	member.NewAPIKeyStatus = 1 // 启用
	if err := s.memberRepo.Update(nil, member); err != nil {
		log.Printf("[WARN] 更新成员 API key 失败: %v", err)
	}

	// CreateMemberAPIKey 不改变分配额度，无需同步 user.Quota
	// user.Quota 由 SetMemberQuota 通过 syncUserQuotaDelta 管理

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

// --- 用户级别 API Key（不关联团队，支持多 Key） ---

// CreateUserAPIKey 为当前用户创建 API Key（无需团队 slug）
func (s *TeamService) CreateUserAPIKey(userID uint, name string) (*model.UserAPIKey, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}
	if user.NewAPIUserID <= 0 {
		return nil, fmt.Errorf("用户的 API 账户尚未创建，请联系管理员")
	}
	if name == "" {
		name = "Default"
	}

	// 查找用户的团队成员记录（用于 token 命名）
	members, _ := s.memberRepo.FindByUserID(nil, userID)
	teamSlug := ""
	if len(members) > 0 {
		if team, err := s.teamRepo.FindByID(nil, members[0].TeamID); err == nil && team != nil {
			teamSlug = team.Slug
		}
	}

	tokenName := fmt.Sprintf("mg_%s_%s", name, user.Username)
	if teamSlug != "" {
		tokenName = fmt.Sprintf("mg_%s_%s_%s", teamSlug, name, user.Username)
	}
	if len(tokenName) > 30 {
		tokenName = tokenName[:30]
	}

	// 用户级别 API Key：设置 unlimited_quota=true
	// 额度控制通过 new-api 的 user.Quota 实现（AdminUpdateUserQuota），
	// token 自身不需要额度限制，否则 remain_quota=0 + unlimited=false 会导致
	// ValidateUserToken 直接返回 "Invalid token"
	unlimited := true
	tokenID, key, err := s.newAPIClient.AdminCreateTokenWithQuota(user.NewAPIUserID, tokenName, nil, &unlimited)
	if err != nil {
		return nil, fmt.Errorf("创建 API Key 失败: %w", err)
	}

	apiKey := &model.UserAPIKey{
		UserID:  userID,
		Name:    name,
		TokenID: tokenID,
		Key:     key,
		Status:  1,
	}
	if err := s.userAPIKeyRepo.Create(apiKey); err != nil {
		// 回滚：删除已创建的 new-api token
		_ = s.newAPIClient.AdminDeleteToken(tokenID)
		return nil, fmt.Errorf("存储 API Key 失败: %w", err)
	}

	return apiKey, nil
}

// ListUserAPIKeys 获取当前用户的 API Key 列表
func (s *TeamService) ListUserAPIKeys(userID uint) ([]model.UserAPIKey, error) {
	return s.userAPIKeyRepo.FindByUserID(userID)
}

// GetUserAPIKey 获取单个 API Key
func (s *TeamService) GetUserAPIKey(userID, keyID uint) (*model.UserAPIKey, error) {
	k, err := s.userAPIKeyRepo.FindByID(keyID)
	if err != nil {
		return nil, fmt.Errorf("API Key 不存在")
	}
	if k.UserID != userID {
		return nil, fmt.Errorf("无权访问该 API Key")
	}
	return k, nil
}

// ToggleUserAPIKey 切换 API Key 状态
func (s *TeamService) ToggleUserAPIKey(userID, keyID uint) (int, error) {
	k, err := s.userAPIKeyRepo.FindByID(keyID)
	if err != nil {
		return 0, fmt.Errorf("API Key 不存在")
	}
	if k.UserID != userID {
		return 0, fmt.Errorf("无权操作该 API Key")
	}

	newStatus := 2
	if k.Status == 2 {
		newStatus = 1
	}

	if k.TokenID > 0 {
		if err := s.newAPIClient.AdminUpdateTokenStatus(k.TokenID, newStatus); err != nil {
			log.Printf("[WARN] 更新 new-api token (id=%d) 状态失败: %v", k.TokenID, err)
			return 0, fmt.Errorf("更新 Key 状态失败")
		}
	}

	k.Status = newStatus
	if err := s.userAPIKeyRepo.Update(k); err != nil {
		return 0, err
	}

	return newStatus, nil
}

// DeleteUserAPIKey 删除用户的 API Key
func (s *TeamService) DeleteUserAPIKey(userID, keyID uint) error {
	k, err := s.userAPIKeyRepo.FindByID(keyID)
	if err != nil {
		return fmt.Errorf("API Key 不存在")
	}
	if k.UserID != userID {
		return fmt.Errorf("无权操作该 API Key")
	}

	// 删除 new-api 中的 token
	if k.TokenID > 0 {
		if err := s.newAPIClient.AdminDeleteToken(k.TokenID); err != nil {
			log.Printf("[WARN] 删除 new-api token (id=%d) 失败: %v", k.TokenID, err)
		}
	}

	return s.userAPIKeyRepo.Delete(keyID)
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

		// 检查用户是否已在其他团队中（每个用户只能加入一个团队）
		if userCount, cntErr := s.memberRepo.CountByUserID(nil, user.ID); cntErr == nil && userCount > 0 {
			failed = append(failed, fmt.Sprintf("%s: 该用户已属于其他团队", displayLabel))
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

	// 检查用户是否已属于其他团队
	count, cntErr := s.memberRepo.CountByUserID(nil, userID)
	if cntErr != nil {
		log.Printf("[WARN] 检查用户团队数量失败: %v", cntErr)
		return
	}
	if count > 0 {
		log.Printf("[INFO] 用户 %d 已属于团队，跳过邀请处理", userID)
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

	// 清理成员在 new-api 中的 token
	if member.NewAPITokenID > 0 {
		if delErr := s.newAPIClient.AdminDeleteToken(member.NewAPITokenID); delErr != nil {
			log.Printf("[WARN] 删除成员 token (id=%d) 失败: %v", member.NewAPITokenID, delErr)
		}
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

// GetMemberQuotaInfo 获取成员额度信息（从 new-api 同步 user.Quota 真实钱包余额）
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

	// 从 new-api 获取用户的真实钱包余额（user.Quota），这是计费的真正来源
	user, err := s.userRepo.FindByID(member.UserID)
	if err != nil {
		return info, nil
	}
	if user.NewAPIUserID > 0 {
		userInfo, err := s.newAPIClient.GetUserInfo(user.NewAPIUserID)
		if err != nil {
			log.Printf("[WARN] 获取用户 new-api 信息失败 userID=%d newAPIUserID=%d: %v", member.UserID, user.NewAPIUserID, err)
			return info, nil
		}

		// 计算用户在所有团队的总分配额度
		totalAllocated, _ := s.memberRepo.SumQuotaAllocatedByUserID(nil, member.UserID)

		// userInfo.Quota 是 new-api 中用户的剩余额度（quota tokens 单位）
		remainCents := int64(userInfo.Quota) / s.quotaTokensPerCent()
		usedCents := totalAllocated - remainCents
		if usedCents < 0 {
			usedCents = 0
		}

		info.QuotaRemain = remainCents
		info.QuotaUsed = usedCents

		// 更新本地缓存
		member.QuotaUsed = usedCents
		_ = s.memberRepo.Update(nil, member)
	}

	return info, nil
}

// syncUserQuotaDelta 将额度增量同步到 new-api
// new-api 中 user.Quota 是剩余额度（随 API 调用被扣减），不能直接用本地计算覆盖
// 正确做法：先从 new-api 获取当前真实 user.Quota，再加上增量 delta
// deltaCents > 0 表示增加分配，< 0 表示减少分配
func (s *TeamService) syncUserQuotaDelta(userID uint, deltaCents int64) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return fmt.Errorf("获取用户信息失败: %w", err)
	}
	if user.NewAPIUserID <= 0 {
		return fmt.Errorf("用户 %s 的 API 账户尚未创建", user.Username)
	}

	// 从 new-api 获取用户当前真实剩余额度
	userInfo, err := s.newAPIClient.GetUserInfo(user.NewAPIUserID)
	if err != nil {
		log.Printf("[WARN] syncUserQuotaDelta: 获取用户 new-api 信息失败 userID=%d: %v", user.NewAPIUserID, err)
		// 获取失败时不覆盖，避免用错误数据覆盖真实额度
		return nil
	}

	// 在当前真实剩余额度基础上加增量
	newQuota := int64(userInfo.Quota) + deltaCents*int64(s.quotaTokensPerCent())
	if newQuota < 0 {
		newQuota = 0
	}

	log.Printf("[INFO] syncUserQuotaDelta userID=%d newAPIUserID=%d currentQuota=%d delta=%d(cents) newQuota=%d",
		userID, user.NewAPIUserID, userInfo.Quota, deltaCents, newQuota)

	return s.newAPIClient.AdminUpdateUserQuota(user.NewAPIUserID, int(newQuota))
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

	// 校验：设置的额度不能为负数
	if amount < 0 {
		return fmt.Errorf("分配额度不能为负数")
	}

	// 从 new-api 实时获取已使用额度（本地 quota_used 可能陈旧）
	realUsedCents := member.QuotaUsed // 默认使用本地缓存
	if memberUser, err := s.userRepo.FindByID(member.UserID); err == nil && memberUser.NewAPIUserID > 0 {
		if userInfo, err := s.newAPIClient.GetUserInfo(memberUser.NewAPIUserID); err == nil {
			totalAllocated, _ := s.memberRepo.SumQuotaAllocatedByUserID(nil, member.UserID)
			remainCents := int64(userInfo.Quota) / s.quotaTokensPerCent()
			calculated := totalAllocated - remainCents
			if calculated < 0 {
				calculated = 0
			}
			realUsedCents = calculated // 获取成功则覆盖
		}
	}

	if amount < realUsedCents {
		return fmt.Errorf("%w: 已使用 ¥%.2f，设置额度 ¥%.2f 不能低于已使用额度",
			ErrQuotaBelowUsed,
			float64(realUsedCents)/100,
			float64(amount)/100,
		)
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

	// 获取成员的新-api 用户 ID
	user, err := s.userRepo.FindByID(member.UserID)
	if err != nil {
		return fmt.Errorf("获取用户信息失败: %w", err)
	}
	if user.NewAPIUserID <= 0 {
		return fmt.Errorf("成员 %s 的 API 账户尚未创建", user.Username)
	}

	log.Printf("[INFO] SetMemberQuota member=%d newAPIUserID=%d amount=%d(cents) oldAllocated=%d(cents) quotaTokensPerCent=%d",
		memberID, user.NewAPIUserID, amount, member.QuotaAllocated, s.quotaTokensPerCent())

	// 计算分配增量
	deltaCents := amount - member.QuotaAllocated

	// 更新本地成员分配额度
	member.QuotaAllocated = amount
	if err := s.memberRepo.Update(nil, member); err != nil {
		log.Printf("[WARN] 更新成员配额缓存失败: %v", err)
	}

	// 将增量同步到 new-api：在当前真实剩余额度基础上加 delta
	if deltaCents != 0 {
		if err := s.syncUserQuotaDelta(member.UserID, deltaCents); err != nil {
			log.Printf("[WARN] 同步用户 %d 额度到 new-api 失败: %v", user.NewAPIUserID, err)
		}
	}

	// 如果成员有旧的团队 token，也更新其配额（兼容旧数据，但计费已不依赖它）
	if member.NewAPITokenID > 0 {
		q := int(amount * s.quotaTokensPerCent())
		if amount > 0 {
			// 设置 token 为无限额度（计费由 user.Quota 控制），同时更新 remain_quota 保持数据一致
			unlimited := true
			if err := s.newAPIClient.AdminUpdateTokenQuota(member.NewAPITokenID, &q, &unlimited); err != nil {
				log.Printf("[WARN] 更新 token 配额失败 tokenID=%d: %v", member.NewAPITokenID, err)
			}
			// 恢复 token 状态
			if member.NewAPIKeyStatus == 1 {
				if err := s.newAPIClient.AdminUpdateTokenStatus(member.NewAPITokenID, 1); err != nil {
					log.Printf("[WARN] 恢复 token 状态失败 tokenID=%d: %v", member.NewAPITokenID, err)
				}
			}
		}
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

// GetMemberLogs 获取当前成员的调用日志（查询该用户在 new-api 中的所有 token 日志）
// 返回的 LogItem.Quota 已从 new-api 内部配额点换算为 modelgate "分"（cents）
func (s *TeamService) GetMemberLogs(userID uint, slug string, q newapi.LogsQuery) (*newapi.PaginatedLogs, error) {
	team, err := s.teamRepo.FindBySlugLight(nil, slug)
	if err != nil {
		return nil, ErrTeamNotFound
	}
	_, err = s.memberRepo.FindByTeamAndUser(nil, team.ID, userID)
	if err != nil {
		return nil, ErrNotMember
	}

	// 获取用户的 new-api 信息
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}
	if user.NewAPIUserID <= 0 {
		return &newapi.PaginatedLogs{Items: []newapi.LogItem{}, Total: 0, Page: q.Page, PageSize: q.PageSize}, nil
	}

	// 使用 admin API 按 username 查询该用户所有 token 的日志
	// new-api 用户名格式为 mg_<username>
	newAPIUsername := fmt.Sprintf("mg_%s", user.Username)
	if len(newAPIUsername) > 20 {
		newAPIUsername = newAPIUsername[:20]
	}
	q.Username = newAPIUsername

	log.Printf("[INFO] GetMemberLogs userID=%d slug=%s newAPIUserID=%d newAPIUsername=%s page=%d pageSize=%d tokenID=%d tokenName=%q start=%d end=%d",
		userID, slug, user.NewAPIUserID, newAPIUsername, q.Page, q.PageSize, q.TokenID, q.TokenName, q.StartTimestamp, q.EndTimestamp)

	result, err := s.newAPIClient.GetLogsByUserID(q)
	if err != nil {
		log.Printf("[ERROR] GetMemberLogs GetLogsByUserID failed: %v", err)
		return nil, fmt.Errorf("获取日志失败: %w", err)
	}
	if result.Items == nil {
		result.Items = []newapi.LogItem{}
	}

	// 将 new-api 的 quota（内部配额点/tokens）换算为 modelgate "分"（cents）
	// 1 分 = quotaTokensPerCent() tokens
	qpc := s.quotaTokensPerCent()

	// 构建 tokenID -> modelgate key 名称 的映射，将 new-api 的 token_name 替换为用户友好的名称
	tokenNameMap := s.buildTokenNameMap(userID)

	for i := range result.Items {
		result.Items[i].Quota = result.Items[i].Quota / int(qpc)
		// 替换 token_name 为 modelgate 中的用户自定义名称
		if name, ok := tokenNameMap[result.Items[i].TokenID]; ok {
			result.Items[i].TokenName = name
		}
	}

	return result, nil
}

// LogKey 日志筛选用的 Key 条目
type LogKey struct {
	TokenID int    `json:"token_id"`
	Name    string `json:"name"`
}

// GetMemberLogKeys 获取当前成员可用于日志筛选的 Key 列表
// 只包含用户级 API Key（所有成员现在都使用用户级 API Key）
func (s *TeamService) GetMemberLogKeys(userID uint, slug string) ([]LogKey, error) {
	team, err := s.teamRepo.FindBySlugLight(nil, slug)
	if err != nil {
		return nil, ErrTeamNotFound
	}
	_, err = s.memberRepo.FindByTeamAndUser(nil, team.ID, userID)
	if err != nil {
		return nil, ErrNotMember
	}

	keys := make([]LogKey, 0)

	// 用户级 API Key
	userKeys, err := s.userAPIKeyRepo.FindByUserID(userID)
	if err == nil {
		for _, k := range userKeys {
			if k.TokenID > 0 {
				keys = append(keys, LogKey{TokenID: k.TokenID, Name: k.Name})
			}
		}
	}

	return keys, nil
}

// buildTokenNameMap 构建 new-api tokenID -> modelgate 名称 的映射
// 包括用户级 API Key（UserAPIKey）和团队成员 Key（TeamMember）
func (s *TeamService) buildTokenNameMap(userID uint) map[int]string {
	nameMap := make(map[int]string)

	// 用户级 API Key
	userKeys, err := s.userAPIKeyRepo.FindByUserID(userID)
	if err == nil {
		for _, k := range userKeys {
			if k.TokenID > 0 {
				nameMap[k.TokenID] = k.Name
			}
		}
	}

	// 团队成员 Key
	members, err := s.memberRepo.FindByUserID(nil, userID)
	if err == nil {
		for _, m := range members {
			if m.NewAPITokenID > 0 {
				team, err := s.teamRepo.FindByID(nil, m.TeamID)
				if err == nil && team != nil {
					nameMap[m.NewAPITokenID] = team.Name
				}
			}
		}
	}

	return nameMap
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
