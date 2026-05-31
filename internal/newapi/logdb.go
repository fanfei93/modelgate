package newapi

import (
	"fmt"
	"log"
	"sync"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// LogDB 直连日志数据库查询
type LogDB struct {
	db *gorm.DB
	mu sync.RWMutex
}

// NewLogDB 创建日志数据库连接（dsn 为空则返回 nil）
func NewLogDB(dsn string) *LogDB {
	if dsn == "" {
		log.Println("[INFO] database_dsn 未配置，日志查询将使用 HTTP API")
		return nil
	}
	log.Printf("[INFO] 正在连接日志数据库")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Printf("[WARN] 连接日志数据库失败: %v，日志查询将使用 HTTP API", err)
		return nil
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)

	log.Printf("[INFO] 已连接日志数据库")
	return &LogDB{db: db}
}

// logs 表对应的 ORM 模型（只映射查询所需字段）
type LogRecord struct {
	ID               int    `gorm:"column:id;primaryKey"`
	UserID           int    `gorm:"column:user_id"`
	CreatedAt        int64  `gorm:"column:created_at"`
	Type             int    `gorm:"column:type"`
	Content          string `gorm:"column:content"`
	Username         string `gorm:"column:username"`
	TokenName        string `gorm:"column:token_name"`
	ModelName        string `gorm:"column:model_name"`
	Quota            int    `gorm:"column:quota"`
	PromptTokens     int    `gorm:"column:prompt_tokens"`
	CompletionTokens int    `gorm:"column:completion_tokens"`
	UseTime          int    `gorm:"column:use_time"`
	IsStream         bool   `gorm:"column:is_stream"`
	ChannelID        int    `gorm:"column:channel_id"`
	TokenID          int    `gorm:"column:token_id"`
	Group            string `gorm:"column:group"`
	IP               string `gorm:"column:ip"`
	RequestID        string `gorm:"column:request_id"`
}

func (LogRecord) TableName() string {
	return "logs"
}

// TokenRecord tokens 表（只映射查询所需字段）
type TokenRecord struct {
	ID     int    `gorm:"column:id;primaryKey"`
	Name   string `gorm:"column:name"`
	UserID int    `gorm:"column:user_id"`
}

func (TokenRecord) TableName() string {
	return "tokens"
}

// GetLogsByUserID 直接从数据库查询指定用户的调用日志
func (l *LogDB) GetLogsByUserID(q LogsQuery) (*PaginatedLogs, error) {
	if l == nil {
		return nil, fmt.Errorf("logdb 未初始化")
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}

	tx := l.db.Model(&LogRecord{})

	// 按用户名筛选
	if q.Username != "" {
		tx = tx.Where("username = ?", q.Username)
	}
	// 按 token_name 筛选
	if q.TokenName != "" {
		tx = tx.Where("token_name = ?", q.TokenName)
	}
	// 按 token_id 筛选
	if q.TokenID > 0 {
		tx = tx.Where("token_id = ?", q.TokenID)
	}
	// 按模型名称筛选
	if q.ModelName != "" {
		tx = tx.Where("model_name LIKE ?", "%"+q.ModelName+"%")
	}
	// 时间范围
	if q.StartTimestamp > 0 {
		tx = tx.Where("created_at >= ?", q.StartTimestamp)
	}
	if q.EndTimestamp > 0 {
		tx = tx.Where("created_at <= ?", q.EndTimestamp)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询日志总数失败: %w", err)
	}

	var records []LogRecord
	offset := (q.Page - 1) * q.PageSize
	if err := tx.Order("id DESC").Limit(q.PageSize).Offset(offset).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("查询日志失败: %w", err)
	}

	items := make([]LogItem, 0, len(records))
	for _, r := range records {
		items = append(items, LogItem{
			ID:               r.ID,
			UserID:           r.UserID,
			CreatedAt:        r.CreatedAt,
			Type:             r.Type,
			Content:          r.Content,
			Username:         r.Username,
			TokenName:        r.TokenName,
			ModelName:        r.ModelName,
			Quota:            r.Quota,
			PromptTokens:     r.PromptTokens,
			CompletionTokens: r.CompletionTokens,
			UseTime:          float64(r.UseTime),
			IsStream:         r.IsStream,
			Channel:          r.ChannelID,
			TokenID:          r.TokenID,
			Group:            r.Group,
			IP:               r.IP,
		})
	}

	return &PaginatedLogs{
		Items:    items,
		Total:    int(total),
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}

// GetTokenIDsByUserID 查询指定用户的所有 token ID
func (l *LogDB) GetTokenIDsByUserID(newAPIUserID int) ([]TokenRecord, error) {
	if l == nil {
		return nil, fmt.Errorf("logdb 未初始化")
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	var tokens []TokenRecord
	if err := l.db.Where("user_id = ?", newAPIUserID).Find(&tokens).Error; err != nil {
		return nil, fmt.Errorf("查询 token 列表失败: %w", err)
	}
	return tokens, nil
}
