package repository

import (
	"github.com/modelgate/internal/model"
	"gorm.io/gorm"
)

type UserRepo struct{ db *gorm.DB }
type TeamRepo struct{ db *gorm.DB }
type MemberRepo struct{ db *gorm.DB }
type InvitationRepo struct{ db *gorm.DB }
type QuotaAllocationRepo struct{ db *gorm.DB }
type LoginLogRepo struct{ db *gorm.DB }

func NewUserRepo(db *gorm.DB) *UserRepo     { return &UserRepo{db} }
func NewTeamRepo(db *gorm.DB) *TeamRepo     { return &TeamRepo{db} }
func NewMemberRepo(db *gorm.DB) *MemberRepo { return &MemberRepo{db} }
func NewInvitationRepo(db *gorm.DB) *InvitationRepo { return &InvitationRepo{db} }
func NewQuotaAllocationRepo(db *gorm.DB) *QuotaAllocationRepo {
	return &QuotaAllocationRepo{db}
}
func NewLoginLogRepo(db *gorm.DB) *LoginLogRepo { return &LoginLogRepo{db} }

// --- UserRepo ---

func (r *UserRepo) Create(u *model.User) error {
	return r.db.Create(u).Error
}

func (r *UserRepo) FindByID(id uint) (*model.User, error) {
	var u model.User
	err := r.db.First(&u, id).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) FindByUsername(username string) (*model.User, error) {
	var u model.User
	err := r.db.Where("username = ?", username).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) FindByEmail(email string) (*model.User, error) {
	var u model.User
	err := r.db.Where("email = ?", email).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) Update(u *model.User) error {
	return r.db.Save(u).Error
}

func (r *UserRepo) List(page, pageSize int) ([]model.User, int64, error) {
	var users []model.User
	var total int64
	r.db.Model(&model.User{}).Count(&total)
	err := r.db.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users).Error
	return users, total, err
}

func (r *UserRepo) UpdateStatus(id uint, status string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("status", status).Error
}

// --- TeamRepo ---

func (r *TeamRepo) Create(tx *gorm.DB, team *model.Team) error {
	if tx == nil {
		tx = r.db
	}
	return tx.Create(team).Error
}

func (r *TeamRepo) FindByID(tx *gorm.DB, id uint) (*model.Team, error) {
	if tx == nil {
		tx = r.db
	}
	var team model.Team
	err := tx.Preload("Members").Preload("Members.User").Preload("Invitations").First(&team, id).Error
	if err != nil {
		return nil, err
	}
	return &team, nil
}

func (r *TeamRepo) FindBySlug(tx *gorm.DB, slug string) (*model.Team, error) {
	if tx == nil {
		tx = r.db
	}
	var team model.Team
	err := tx.Where("slug = ?", slug).Preload("Members").Preload("Members.User").Preload("Invitations").First(&team).Error
	if err != nil {
		return nil, err
	}
	return &team, nil
}

func (r *TeamRepo) FindBySlugLight(tx *gorm.DB, slug string) (*model.Team, error) {
	if tx == nil {
		tx = r.db
	}
	var team model.Team
	err := tx.Where("slug = ?", slug).First(&team).Error
	if err != nil {
		return nil, err
	}
	return &team, nil
}

func (r *TeamRepo) Update(tx *gorm.DB, team *model.Team) error {
	if tx == nil {
		tx = r.db
	}
	return tx.Save(team).Error
}

func (r *TeamRepo) Delete(tx *gorm.DB, id uint) error {
	if tx == nil {
		tx = r.db
	}
	return tx.Delete(&model.Team{}, id).Error
}

func (r *TeamRepo) FindByUserID(tx *gorm.DB, userID uint) ([]model.Team, error) {
	if tx == nil {
		tx = r.db
	}
	var teams []model.Team
	err := tx.
		Joins("JOIN team_members ON team_members.team_id = teams.id").
		Where("team_members.user_id = ?", userID).
		Where("teams.deleted_at IS NULL").
		Preload("Members.User").
		Find(&teams).Error
	return teams, err
}

// --- MemberRepo ---

func (r *MemberRepo) Create(tx *gorm.DB, m *model.TeamMember) error {
	if tx == nil {
		tx = r.db
	}
	return tx.Create(m).Error
}

func (r *MemberRepo) DeleteByID(tx *gorm.DB, id uint) error {
	if tx == nil {
		tx = r.db
	}
	return tx.Delete(&model.TeamMember{}, id).Error
}

// DeleteByTeamAndUser 删除指定团队中的成员记录
func (r *MemberRepo) DeleteByTeamAndUser(tx *gorm.DB, teamID, userID uint) error {
	if tx == nil {
		tx = r.db
	}
	return tx.Where("team_id = ? AND user_id = ?", teamID, userID).Delete(&model.TeamMember{}).Error
}

func (r *MemberRepo) Update(tx *gorm.DB, m *model.TeamMember) error {
	if tx == nil {
		tx = r.db
	}
	return tx.Save(m).Error
}

func (r *MemberRepo) FindByTeamAndUser(tx *gorm.DB, teamID, userID uint) (*model.TeamMember, error) {
	if tx == nil {
		tx = r.db
	}
	var m model.TeamMember
	err := tx.Where("team_id = ? AND user_id = ?", teamID, userID).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MemberRepo) FindByID(tx *gorm.DB, id uint) (*model.TeamMember, error) {
	if tx == nil {
		tx = r.db
	}
	var m model.TeamMember
	err := tx.First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MemberRepo) FindByUserID(tx *gorm.DB, userID uint) ([]model.TeamMember, error) {
	if tx == nil {
		tx = r.db
	}
	var members []model.TeamMember
	err := tx.Where("user_id = ?", userID).Find(&members).Error
	return members, err
}

// CountByUserID 统计用户所属的团队数量
func (r *MemberRepo) CountByUserID(tx *gorm.DB, userID uint) (int64, error) {
	if tx == nil {
		tx = r.db
	}
	var count int64
	err := tx.Model(&model.TeamMember{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// FindByTeamID 获取团队的所有成员
func (r *MemberRepo) FindByTeamID(tx *gorm.DB, teamID uint) ([]model.TeamMember, error) {
	if tx == nil {
		tx = r.db
	}
	var members []model.TeamMember
	err := tx.Where("team_id = ?", teamID).Find(&members).Error
	return members, err
}

// SumQuotaAllocatedExcept 计算团队中除指定成员外所有人的已分配额度总和
func (r *MemberRepo) SumQuotaAllocatedExcept(tx *gorm.DB, teamID, excludeMemberID uint) (int64, error) {
	if tx == nil {
		tx = r.db
	}
	var sum int64
	err := tx.Model(&model.TeamMember{}).
		Where("team_id = ? AND id != ?", teamID, excludeMemberID).
		Select("COALESCE(SUM(quota_allocated), 0)").
		Scan(&sum).Error
	return sum, err
}

// SumQuotaAllocatedByUserID 计算用户在所有团队中的已分配额度总和
func (r *MemberRepo) SumQuotaAllocatedByUserID(tx *gorm.DB, userID uint) (int64, error) {
	if tx == nil {
		tx = r.db
	}
	var sum int64
	err := tx.Model(&model.TeamMember{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(quota_allocated), 0)").
		Scan(&sum).Error
	return sum, err
}

// SumQuotaUsedByUserID 计算用户在所有团队中的已使用额度总和
func (r *MemberRepo) SumQuotaUsedByUserID(tx *gorm.DB, userID uint) (int64, error) {
	if tx == nil {
		tx = r.db
	}
	var sum int64
	err := tx.Model(&model.TeamMember{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(quota_used), 0)").
		Scan(&sum).Error
	return sum, err
}

// --- InvitationRepo ---

func (r *InvitationRepo) Create(inv *model.TeamInvitation) error {
	return r.db.Create(inv).Error
}

func (r *InvitationRepo) FindByTeamAndEmail(teamID uint, email string) (*model.TeamInvitation, error) {
	var inv model.TeamInvitation
	err := r.db.Where("team_id = ? AND email = ?", teamID, email).First(&inv).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *InvitationRepo) FindByEmail(email string) ([]model.TeamInvitation, error) {
	var invs []model.TeamInvitation
	err := r.db.Where("email = ? AND status = ?", email, "pending").Find(&invs).Error
	return invs, err
}

func (r *InvitationRepo) UpdateStatus(id uint, status string) error {
	return r.db.Model(&model.TeamInvitation{}).Where("id = ?", id).Update("status", status).Error
}

func (r *InvitationRepo) DeleteByID(id uint) error {
	return r.db.Delete(&model.TeamInvitation{}, id).Error
}

func (r *InvitationRepo) FindByTeamID(teamID uint) ([]model.TeamInvitation, error) {
	var invs []model.TeamInvitation
	err := r.db.Where("team_id = ?", teamID).Order("created_at DESC").Find(&invs).Error
	return invs, err
}

// --- UserAPIKeyRepo ---

type UserAPIKeyRepo struct{ db *gorm.DB }

func NewUserAPIKeyRepo(db *gorm.DB) *UserAPIKeyRepo {
	return &UserAPIKeyRepo{db}
}

func (r *UserAPIKeyRepo) Create(key *model.UserAPIKey) error {
	return r.db.Create(key).Error
}

func (r *UserAPIKeyRepo) FindByID(id uint) (*model.UserAPIKey, error) {
	var k model.UserAPIKey
	err := r.db.First(&k, id).Error
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *UserAPIKeyRepo) FindByUserID(userID uint) ([]model.UserAPIKey, error) {
	var keys []model.UserAPIKey
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&keys).Error
	return keys, err
}

func (r *UserAPIKeyRepo) Update(key *model.UserAPIKey) error {
	return r.db.Save(key).Error
}

func (r *UserAPIKeyRepo) Delete(id uint) error {
	return r.db.Delete(&model.UserAPIKey{}, id).Error
}

// --- QuotaAllocationRepo ---

func (r *QuotaAllocationRepo) Create(alloc *model.QuotaAllocation) error {
	return r.db.Create(alloc).Error
}

func (r *QuotaAllocationRepo) FindByMemberID(memberID uint) ([]model.QuotaAllocation, error) {
	var allocs []model.QuotaAllocation
	err := r.db.Where("member_id = ?", memberID).Order("created_at DESC").Find(&allocs).Error
	return allocs, err
}

func (r *QuotaAllocationRepo) FindByTeamID(teamID uint) ([]model.QuotaAllocation, error) {
	var allocs []model.QuotaAllocation
	err := r.db.Where("team_id = ?", teamID).Order("created_at DESC").Find(&allocs).Error
	return allocs, err
}

// --- VerificationCodeRepo ---

type VerificationCodeRepo struct{ db *gorm.DB }

func NewVerificationCodeRepo(db *gorm.DB) *VerificationCodeRepo {
	return &VerificationCodeRepo{db}
}

func (r *VerificationCodeRepo) Upsert(vc *model.VerificationCode) error {
	return r.db.Where("email = ?", vc.Email).Assign(vc).FirstOrCreate(vc).Error
}

func (r *VerificationCodeRepo) FindByEmail(email string) (*model.VerificationCode, error) {
	var vc model.VerificationCode
	err := r.db.Where("email = ?", email).First(&vc).Error
	if err != nil {
		return nil, err
	}
	return &vc, nil
}

func (r *VerificationCodeRepo) DeleteByEmail(email string) error {
	return r.db.Where("email = ?", email).Delete(&model.VerificationCode{}).Error
}

// --- SiteSettingRepo ---

type SiteSettingRepo struct{ db *gorm.DB }

func NewSiteSettingRepo(db *gorm.DB) *SiteSettingRepo {
	return &SiteSettingRepo{db}
}

func (r *SiteSettingRepo) FindByKey(key string) (*model.SiteSetting, error) {
	var s model.SiteSetting
	err := r.db.Where("`key` = ?", key).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SiteSettingRepo) FindAll() ([]model.SiteSetting, error) {
	var settings []model.SiteSetting
	err := r.db.Order("`key` ASC").Find(&settings).Error
	return settings, err
}

func (r *SiteSettingRepo) Upsert(setting *model.SiteSetting) error {
	return r.db.Save(setting).Error
}

// UpdateValue 按 key 更新 value
func (r *SiteSettingRepo) UpdateValue(key, value string) error {
	return r.db.Model(&model.SiteSetting{}).Where("`key` = ?", key).Update("value", value).Error
}

// GetValue 获取配置值，不存在返回默认值
func (r *SiteSettingRepo) GetValue(key string, defaultVal string) string {
	s, err := r.FindByKey(key)
	if err != nil {
		return defaultVal
	}
	return s.Value
}

// --- LoginLogRepo ---

func (r *LoginLogRepo) Create(log *model.LoginLog) error {
	return r.db.Create(log).Error
}

func (r *LoginLogRepo) List(page, pageSize int) ([]model.LoginLog, int64, error) {
	var logs []model.LoginLog
	var total int64
	r.db.Model(&model.LoginLog{}).Count(&total)
	err := r.db.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&logs).Error
	return logs, total, err
}

// --- RechargeLogRepo ---

type RechargeLogRepo struct{ db *gorm.DB }

func NewRechargeLogRepo(db *gorm.DB) *RechargeLogRepo {
	return &RechargeLogRepo{db}
}

func (r *RechargeLogRepo) Create(log *model.RechargeLog) error {
	return r.db.Create(log).Error
}

func (r *RechargeLogRepo) List(page, pageSize int) ([]model.RechargeLog, int64, error) {
	var logs []model.RechargeLog
	var total int64
	r.db.Model(&model.RechargeLog{}).Count(&total)
	err := r.db.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&logs).Error
	return logs, total, err
}

func (r *RechargeLogRepo) FindByTeamID(teamID uint, page, pageSize int) ([]model.RechargeLog, int64, error) {
	var logs []model.RechargeLog
	var total int64
	r.db.Model(&model.RechargeLog{}).Where("team_id = ?", teamID).Count(&total)
	err := r.db.Where("team_id = ?", teamID).Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&logs).Error
	return logs, total, err
}
