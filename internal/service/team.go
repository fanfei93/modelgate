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

// CreateMemberAPIKey 为当前成员创建个人 API Key
// 在成员自己的 new-api 用户下创建 token
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

	// 如果已有 key，先获取旧 token 的配额信息，以便重建时保留
	var oldQuota *int
	if member.NewAPITokenID > 0 {
		if info, err := s.newAPIClient.AdminGetTokenInfo(member.NewAPITokenID); err == nil {
			q := info.RemainQuota + info.UsedQuota
			oldQuota = &q
		}
		// 删除旧 token
		if err := s.newAPIClient.AdminDeleteToken(member.NewAPITokenID); err != nil {
			log.Printf("[WARN] 删除旧 token (id=%d) 失败: %v", member.NewAPITokenID, err)
		}
	} else if member.QuotaAllocated > 0 {
		// 首次创建 token，但成员已有分配额度，使用分配额度作为初始配额
		q := int(member.QuotaAllocated * s.quotaTokensPerCent())
		oldQuota = &q
		log.Printf("[INFO] CreateAPIKey: 使用已分配额度 member=%d QuotaAllocated=%d(cents) q=%d(tokens)",
			member.ID, member.QuotaAllocated, q)
	} else {
		// 成员尚未被分配额度，不能创建 API Key
		return "", fmt.Errorf("尚未分配额度，请联系团队 Owner 先分配额度后再创建 API Key")
	}

	tokenName := fmt.Sprintf("mg_%s_%s", team.Slug, user.Username)
	if len(tokenName) > 30 {
		tokenName = tokenName[:30]
	}

	// 在成员自己的 new-api 用户下创建新 token，保留旧配额
	if oldQuota != nil {
		log.Printf("[INFO] CreateAPIKey user=%d member=%d newAPIUserID=%d oldQuota=%d",
			userID, member.ID, user.NewAPIUserID, *oldQuota)
	} else {
		log.Printf("[INFO] CreateAPIKey user=%d member=%d newAPIUserID=%d oldQuota=nil QuotaAllocated=%d",
			userID, member.ID, user.NewAPIUserID, member.QuotaAllocated)
	}
	tokenID, key, err := s.newAPIClient.AdminCreateTokenWithQuota(
		user.NewAPIUserID, tokenName, oldQuota, nil,
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

	q := int(amount * s.quotaTokensPerCent())
	log.Printf("[INFO] SetMemberQuota member=%d newAPIUserID=%d amount=%d(sents) quotaTokensPerCent=%d q=%d",
		memberID, user.NewAPIUserID, amount, s.quotaTokensPerCent(), q)

	// 同步用户级别的额度到 new-api（new-api 计费时先检查 user.Quota 再检查 token.RemainQuota）
	if err := s.newAPIClient.AdminUpdateUserQuota(user.NewAPIUserID, q); err != nil {
		log.Printf("[WARN] 同步用户 %d 额度到 new-api 失败: %v", user.NewAPIUserID, err)
		// 不阻断流程，token 额度已设置
	}

	// 根据成员 token 状态选择操作路径
	if member.NewAPIKey == "" || member.NewAPITokenID <= 0 {
		// 没有 API Key，或 NewAPITokenID 未记录（旧数据迁移场景，token 已存在于 new-api 但 modelgate 未记录其 ID）
		// 创建新 token 并附带配额，旧 orphan token 随后可通过后台手动清理
		if member.NewAPITokenID <= 0 && member.NewAPIKey != "" {
			log.Printf("[INFO] 成员 %d 有 API Key 但 NewAPITokenID 未记录，将创建新 token", member.ID)
		}
		tokenName := fmt.Sprintf("mg_%s_%s", slug, user.Username)
		if len(tokenName) > 30 {
			tokenName = tokenName[:30]
		}

		tokenID, key, err := s.newAPIClient.AdminCreateTokenWithQuota(
			user.NewAPIUserID, tokenName, &q, nil,
		)
		if err != nil {
			return fmt.Errorf("创建 API Key 失败: %w", err)
		}

		member.NewAPITokenID = tokenID
		member.NewAPIKey = key
		member.NewAPIKeyStatus = 1
	} else {
		// 更新现有 token 的配额
		if err := s.newAPIClient.AdminUpdateTokenQuota(member.NewAPITokenID, &q, nil); err != nil {
			return fmt.Errorf("更新配额失败: %w", err)
		}
		// 配额更新后，如果 token 因之前额度耗尽被标记为 Exhausted(4)，
		// 需要恢复为 Enabled(1)，否则后续 API 调用仍会返回 "Invalid token"
		if member.NewAPIKeyStatus == 1 {
			if err := s.newAPIClient.AdminUpdateTokenStatus(member.NewAPITokenID, 1); err != nil {
				log.Printf("[WARN] 恢复 token 状态失败 tokenID=%d: %v", member.NewAPITokenID, err)
			}
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
