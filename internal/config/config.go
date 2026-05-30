package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server      ServerConfig   `yaml:"server"`
	JWT         JWTConfig      `yaml:"jwt"`
	Database    DatabaseConfig `yaml:"database"`
	NewAPI      NewAPIConfig   `yaml:"new_api"`
	SMTP        SMTPConfig     `yaml:"smtp"`
	AdminEmails []string       `yaml:"admin_emails"` // 超管邮箱列表
}

type ServerConfig struct {
	Port      int    `yaml:"port"`
	DebugMode bool   `yaml:"debug_mode"`
	StaticDir string `yaml:"static_dir"` // 前端构建产物目录，为空则使用 web/dist
	BaseURL   string `yaml:"base_url"`   // 用于生成邮件中链接的前端地址，如 http://localhost:5173
}

type JWTConfig struct {
	Secret      string `yaml:"secret"`
	ExpireHours int    `yaml:"expire_hours"`
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"` // sqlite, mysql, postgres
	DSN    string `yaml:"dsn"`
}

type NewAPIConfig struct {
	BaseURL      string `yaml:"base_url"`
	AdminKey     string `yaml:"admin_key"`
	AdminUserID  int    `yaml:"admin_user_id"` // new-api 中 admin key 对应的用户 ID，用于 API 调用时的 New-Api-User 头
}

type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	FromName string `yaml:"from_name"` // 发件人名称，默认 "ModelGate"
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	setDefaults(cfg)
	return cfg, nil
}

func setDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.JWT.ExpireHours == 0 {
		cfg.JWT.ExpireHours = 72
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Database.DSN == "" {
		cfg.Database.DSN = "data.db"
	}
}
