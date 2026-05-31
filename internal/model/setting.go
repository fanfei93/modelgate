package model

import "time"

// SiteSetting 站点配置项（key-value 结构）
type SiteSetting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"size:64;uniqueIndex;not null" json:"key"`
	Value     string    `gorm:"size:2048;not null;default:''" json:"value"`
	Comment   string    `gorm:"size:256" json:"comment"` // 配置项说明
	UpdatedAt time.Time `json:"updated_at"`
}
