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
	teamRepo   *repository.TeamRepo
	memberRepo *repository.MemberRepo
	db         *gorm.DB
}

func NewAdminService(db *gorm.DB, teamRepo *repository.TeamRepo, memberRepo *repository.MemberRepo) *AdminService {
	return &AdminService{
		db:         db,
		teamRepo:   teamRepo,
		memberRepo: memberRepo,
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
func (s *AdminService) RechargeTeam(slug string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("充值金额必须大于0")
	}

	team, err := s.teamRepo.FindBySlugLight(nil, slug)
	if err != nil {
		return ErrTeamNotFound
	}

	team.Balance += amount
	return s.teamRepo.Update(nil, team)
}

// GetTeamBalance 查看团队余额
func (s *AdminService) GetTeamBalance(slug string) (int64, error) {
	team, err := s.teamRepo.FindBySlugLight(nil, slug)
	if err != nil {
		return 0, ErrTeamNotFound
	}
	return team.Balance, nil
}
