package service

import (
	"errors"
	"fmt"

	"github.com/modelgate/internal/model"
	"github.com/modelgate/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrNotAdmin  = errors.New("无管理员权限")
	ErrAdminOnly = errors.New("仅管理员可执行此操作")
)

type AdminService struct {
	teamRepo       *repository.TeamRepo
	memberRepo     *repository.MemberRepo
	settingRepo    *repository.SiteSettingRepo
	loginLogRepo   *repository.LoginLogRepo
	rechargeLogRepo *repository.RechargeLogRepo
	db             *gorm.DB
}

func NewAdminService(db *gorm.DB, teamRepo *repository.TeamRepo, memberRepo *repository.MemberRepo, settingRepo *repository.SiteSettingRepo, loginLogRepo *repository.LoginLogRepo, rechargeLogRepo *repository.RechargeLogRepo) *AdminService {
	return &AdminService{
		db:              db,
		teamRepo:        teamRepo,
		memberRepo:      memberRepo,
		settingRepo:     settingRepo,
		loginLogRepo:    loginLogRepo,
		rechargeLogRepo: rechargeLogRepo,
	}
}

// AdminTeamItem 超管团队列表项
type AdminTeamItem struct {
	ID              uint   `json:"id"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	OwnerID         uint   `json:"owner_id"`
	Balance         int64  `json:"balance"`          // 分
	Status          string `json:"status"`
	MemberCount     int    `json:"member_count"`
	CreatedAt       string `json:"created_at"`
}

// ListTeams 列出所有团队（管理员视角）
func (s *AdminService) ListTeams() ([]AdminTeamItem, error) {
	var teams []model.Team
	if err := s.db.Where("deleted_at IS NULL").Order("created_at DESC").Find(&teams).Error; err != nil {
		return nil, fmt.Errorf("获取团队列表失败: %w", err)
	}

	result := make([]AdminTeamItem, 0, len(teams))
	for _, t := range teams {
		memberCount := 0
		if members, err := s.memberRepo.FindByTeamID(nil, t.ID); err == nil {
			memberCount = len(members)
		}

		result = append(result, AdminTeamItem{
			ID:          t.ID,
			Name:        t.Name,
			Slug:        t.Slug,
			OwnerID:     t.OwnerID,
			Balance:     t.Balance,
			Status:      t.Status,
			MemberCount: memberCount,
			CreatedAt:   t.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return result, nil
}

// RechargeTeam 为团队充值（管理员操作）
// amount 为分单位
// operatorID / operatorName 为执行充值的超管信息
// ip 为操作者 IP
// remark 为可选备注
func (s *AdminService) RechargeTeam(slug string, amount int64, operatorID uint, operatorName, ip, remark string) error {
	if amount <= 0 {
		return fmt.Errorf("充值金额必须大于0")
	}

	team, err := s.teamRepo.FindBySlugLight(nil, slug)
	if err != nil {
		return ErrTeamNotFound
	}

	balanceBefore := team.Balance
	team.Balance += amount
	if err := s.teamRepo.Update(nil, team); err != nil {
		return err
	}

	// 记录充值审计日志
	rechargeLog := &model.RechargeLog{
		TeamID:        team.ID,
		TeamName:      team.Name,
		OperatorID:    operatorID,
		OperatorName:  operatorName,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  team.Balance,
		Remark:        remark,
		IP:            ip,
	}
	if err := s.rechargeLogRepo.Create(rechargeLog); err != nil {
		// 日志记录失败不影响充值结果，但需记录
		_ = err
	}

	return nil
}

// GetTeamBalance 查看团队余额
func (s *AdminService) GetTeamBalance(slug string) (int64, error) {
	team, err := s.teamRepo.FindBySlugLight(nil, slug)
	if err != nil {
		return 0, ErrTeamNotFound
	}
	return team.Balance, nil
}

// --- 站点配置管理 ---

// SiteSettingItem 配置项 DTO
type SiteSettingItem struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Comment string `json:"comment"`
}

// ListSettings 获取所有站点配置
func (s *AdminService) ListSettings() ([]SiteSettingItem, error) {
	settings, err := s.settingRepo.FindAll()
	if err != nil {
		return nil, fmt.Errorf("获取站点配置失败: %w", err)
	}
	result := make([]SiteSettingItem, 0, len(settings))
	for _, s := range settings {
		result = append(result, SiteSettingItem{
			Key:     s.Key,
			Value:   s.Value,
			Comment: s.Comment,
		})
	}
	return result, nil
}

// UpdateSetting 更新单个站点配置
func (s *AdminService) UpdateSetting(key, value string) error {
	// 先检查 key 是否存在
	if _, err := s.settingRepo.FindByKey(key); err != nil {
		return fmt.Errorf("配置项 %s 不存在", key)
	}
	return s.settingRepo.UpdateValue(key, value)
}

// --- 登录日志 ---

// LoginLogItem 登录日志 DTO
type LoginLogItem struct {
	ID        uint   `json:"id"`
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Success   bool   `json:"success"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"created_at"`
}

// ListLoginLogs 分页查询登录日志
func (s *AdminService) ListLoginLogs(page, pageSize int) ([]LoginLogItem, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	logs, total, err := s.loginLogRepo.List(page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("获取登录日志失败: %w", err)
	}
	items := make([]LoginLogItem, 0, len(logs))
	for _, l := range logs {
		items = append(items, LoginLogItem{
			ID:        l.ID,
			UserID:    l.UserID,
			Username:  l.Username,
			IP:        l.IP,
			UserAgent: l.UserAgent,
			Success:   l.Success,
			Reason:    l.Reason,
			CreatedAt: l.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return items, total, nil
}

// --- 充值审计日志 ---

// RechargeLogItem 充值审计日志 DTO
type RechargeLogItem struct {
	ID            uint   `json:"id"`
	TeamID        uint   `json:"team_id"`
	TeamName      string `json:"team_name"`
	OperatorID    uint   `json:"operator_id"`
	OperatorName  string `json:"operator_name"`
	Amount        int64  `json:"amount"`
	BalanceBefore int64  `json:"balance_before"`
	BalanceAfter  int64  `json:"balance_after"`
	Remark        string `json:"remark"`
	IP            string `json:"ip"`
	CreatedAt     string `json:"created_at"`
}

// ListRechargeLogs 分页查询充值审计日志
func (s *AdminService) ListRechargeLogs(page, pageSize int) ([]RechargeLogItem, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	logs, total, err := s.rechargeLogRepo.List(page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("获取充值日志失败: %w", err)
	}
	items := make([]RechargeLogItem, 0, len(logs))
	for _, l := range logs {
		items = append(items, RechargeLogItem{
			ID:            l.ID,
			TeamID:        l.TeamID,
			TeamName:      l.TeamName,
			OperatorID:    l.OperatorID,
			OperatorName:  l.OperatorName,
			Amount:        l.Amount,
			BalanceBefore: l.BalanceBefore,
			BalanceAfter:  l.BalanceAfter,
			Remark:        l.Remark,
			IP:            l.IP,
			CreatedAt:      l.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return items, total, nil
}
