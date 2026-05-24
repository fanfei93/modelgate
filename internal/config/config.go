package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	JWT      JWTConfig      `yaml:"jwt"`
	Database DatabaseConfig `yaml:"database"`
	NewAPI   NewAPIConfig   `yaml:"new_api"`
}

type ServerConfig struct {
	Port      int    `yaml:"port"`
	DebugMode bool   `yaml:"debug_mode"`
	StaticDir string `yaml:"static_dir"` // 前端构建产物目录，为空则使用 web/dist
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
	BaseURL  string `yaml:"base_url"`
	AdminKey string `yaml:"admin_key"`
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
