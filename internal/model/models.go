package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Username       string         `gorm:"size:64;uniqueIndex;not null" json:"username"`
	Email          string         `gorm:"size:128;uniqueIndex;not null" json:"email"`
	PasswordHash   string         `gorm:"size:256;not null" json:"-"`
	DisplayName    string         `gorm:"size:128" json:"display_name"`
	NewAPIUserID   int            `gorm:"default:0" json:"-"`          // 绑定的 new-api 用户 ID
	NewAPIPassword string         `gorm:"size:256;default:''" json:"-"` // new-api 账户密码，用于创建 token
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	Teams          []TeamMember   `gorm:"foreignKey:UserID" json:"-"`
}

type Team struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Name          string         `gorm:"size:128;not null" json:"name"`
	Slug          string         `gorm:"size:64;uniqueIndex;not null" json:"slug"`
	NewAPIUserID  int            `gorm:"default:0" json:"-"`      // [废弃] 团队不再绑定 new-api 用户
	NewAPIPassword string        `gorm:"size:256;default:''" json:"-"` // [废弃] 
	OwnerID       uint           `gorm:"not null;index" json:"owner_id"`
	Balance       int64          `gorm:"not null;default:0" json:"balance"` // 分为单位，由超管充值
	Status        string         `gorm:"size:16;not null;default:active" json:"status"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Members       []TeamMember     `gorm:"foreignKey:TeamID" json:"members,omitempty"`
	Invitations   []TeamInvitation `gorm:"foreignKey:TeamID" json:"invitations,omitempty"`
}

type TeamMember struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TeamID          uint      `gorm:"uniqueIndex:uk_team_user;not null" json:"team_id"`
	UserID          uint      `gorm:"uniqueIndex:uk_team_user;uniqueIndex:idx_member_user_id;not null" json:"user_id"`
	Role            string    `gorm:"size:16;not null;default:member" json:"role"` // owner / member
	JoinedAt        time.Time `json:"joined_at"`
	NewAPITokenID   int       `gorm:"index" json:"-"`                             // new-api 中该成员的 token ID
	NewAPIKey       string    `gorm:"size:256" json:"-"`                          // 成员的 API Key（存储在团队 new-api 用户下的 token）
	NewAPIKeyStatus int       `gorm:"not null;default:0" json:"-"`                // 0: 未创建, 1: 启用, 2: 禁用
	NewAPIKeyMask   string    `gorm:"-" json:"api_key_mask,omitempty"`            // 脱敏展示
	QuotaAllocated  int64     `gorm:"not null;default:0" json:"quota_allocated"`  // 已分配额度（new-api 配额单位，本地缓存）
	QuotaUsed       int64     `gorm:"not null;default:0" json:"quota_used"`       // 已使用额度（从 new-api token 同步，本地缓存）
	User            *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type TeamInvitation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TeamID    uint      `gorm:"uniqueIndex:uk_team_email;not null" json:"team_id"`
	Email     string    `gorm:"size:128;uniqueIndex:uk_team_email;not null" json:"email"`
	Name      string    `gorm:"size:64" json:"name"`                    // 邀请时填写的昵称
	InviterID uint      `gorm:"not null" json:"inviter_id"`
	Status    string    `gorm:"size:16;not null;default:pending" json:"status"` // pending / accepted
	CreatedAt time.Time `json:"created_at"`
}

// VerificationCode 邮箱验证码
type VerificationCode struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"size:128;uniqueIndex;not null" json:"email"`
	Code      string    `gorm:"size:16;not null" json:"-"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// QuotaAllocation 额度分配记录
type QuotaAllocation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TeamID    uint      `gorm:"not null;index" json:"team_id"`
	MemberID  uint      `gorm:"not null;index" json:"member_id"`
	OperatorID uint     `gorm:"not null" json:"operator_id"`                      // 操作者（owner）
	Type      string    `gorm:"size:16;not null" json:"type"`                     // allocate / revoke
	Amount    int64     `gorm:"not null" json:"amount"`                           // 分配/回收的额度（new-api 配额单位）
	CreatedAt time.Time `json:"created_at"`
}

// UserAPIKey 用户个人 API Key（一个用户可拥有多个）
type UserAPIKey struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index:idx_userapikey_user_id" json:"user_id"`
	Name      string    `gorm:"size:128;not null" json:"name"`     // 用户自定义名称，如 "Production" / "Dev"
	TokenID   int       `gorm:"index" json:"-"`                    // new-api 中的 token ID
	Key       string    `gorm:"size:256;not null" json:"-"`        // 完整密钥（不直接序列化）
	Status    int       `gorm:"not null;default:1" json:"status"`  // 1: 启用, 2: 禁用
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
