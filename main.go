package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/modelgate/internal/config"
	"github.com/modelgate/internal/handler"
	"github.com/modelgate/internal/middleware"
	"github.com/modelgate/internal/model"
	"github.com/modelgate/internal/newapi"
	"github.com/modelgate/internal/repository"
	"github.com/modelgate/internal/service"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	configPath := flag.String("config", "config.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 配置 Gin 模式
	if !cfg.Server.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化数据库
	db, err := initDB(cfg)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(
		&model.User{},
		&model.Team{},
		&model.TeamMember{},
		&model.TeamInvitation{},
		&model.QuotaAllocation{},
		&model.VerificationCode{},
		&model.UserAPIKey{},
		&model.SiteSetting{},
		&model.LoginLog{},
		&model.RechargeLog{},
	); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
	log.Println("数据库迁移完成")

	// 初始化默认站点配置
	initDefaultSettings(db)

	// 初始化各层
	userRepo := repository.NewUserRepo(db)
	teamRepo := repository.NewTeamRepo(db)
	memberRepo := repository.NewMemberRepo(db)
	userAPIKeyRepo := repository.NewUserAPIKeyRepo(db)
	invitationRepo := repository.NewInvitationRepo(db)
	quotaAllocRepo := repository.NewQuotaAllocationRepo(db)
	vcRepo := repository.NewVerificationCodeRepo(db)
	settingRepo := repository.NewSiteSettingRepo(db)
	loginLogRepo := repository.NewLoginLogRepo(db)
	rechargeLogRepo := repository.NewRechargeLogRepo(db)

	newAPIClient := newapi.NewClient(cfg.NewAPI.BaseURL, cfg.NewAPI.AdminKey, cfg.NewAPI.AdminUserID)
	logDB := newapi.NewLogDB(cfg.NewAPI.DatabaseDSN)
	emailService := service.NewEmailService(cfg.SMTP, vcRepo)

	authService := service.NewAuthService(userRepo, newAPIClient, loginLogRepo)
	teamService := service.NewTeamService(db, teamRepo, memberRepo, userRepo, userAPIKeyRepo, invitationRepo, quotaAllocRepo, newAPIClient, logDB, emailService, cfg.Server.BaseURL)
	teamService.InitQuotaPerUnit() // 从 new-api 动态获取 QuotaPerUnit 换算因子

	adminService := service.NewAdminService(db, teamRepo, memberRepo, settingRepo, loginLogRepo, rechargeLogRepo)

	authHandler := handler.NewAuthHandler(authService, teamService, emailService, cfg.JWT.Secret, cfg.JWT.ExpireHours, cfg.AdminEmails)
	teamHandler := handler.NewTeamHandler(teamService)
	adminHandler := handler.NewAdminHandler(adminService)

	// 创建路由
	r := gin.Default()

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// 健康检查
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 模型定价（公开接口，转发自 new-api）
	r.GET("/api/pricing", func(c *gin.Context) {
		pricing, err := newAPIClient.GetPricing()
		if err != nil {
			c.JSON(500, gin.H{"error": "获取模型定价失败"})
			return
		}
		c.JSON(200, pricing)
	})

	// 站点配置（公开接口，前端渲染导航栏等使用）
	r.GET("/api/site-settings", func(c *gin.Context) {
		settings, err := adminService.ListSettings()
		if err != nil {
			c.JSON(500, gin.H{"error": "获取站点配置失败"})
			return
		}
		c.JSON(200, gin.H{"data": settings})
	})

	// 认证路由（无需 JWT）
	authGroup := r.Group("/api/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/send-verification-code", authHandler.SendVerificationCode)
	}

	// 需要认证的路由
	protected := r.Group("/api")
	protected.Use(middleware.AuthRequired(cfg.JWT.Secret))
	{
		protected.GET("/auth/me", authHandler.Me)

		// API Keys 管理（用户级别，无需关联团队，支持多 Key）
		protected.GET("/me/keys", teamHandler.ListUserKeys)
		protected.POST("/me/keys", teamHandler.CreateUserKey)
		protected.GET("/me/keys/:id", teamHandler.GetUserKey)
		protected.PUT("/me/keys/:id/toggle", teamHandler.ToggleUserKey)
		protected.DELETE("/me/keys/:id", teamHandler.DeleteUserKey)

		// API Keys 管理（旧版，按团队展示）
		protected.GET("/me/api-keys", teamHandler.GetMyAPIKeys)

		// 团队管理
		teams := protected.Group("/teams")
		{
			teams.POST("", teamHandler.CreateTeam)
			teams.GET("", teamHandler.GetTeams)
			teams.GET("/:slug", teamHandler.GetTeam)
			teams.DELETE("/:slug", teamHandler.DeleteTeam)

			// 成员管理
			teams.POST("/:slug/members", teamHandler.AddMembers)
			teams.DELETE("/:slug/members/:memberId", teamHandler.RemoveMember)
			teams.DELETE("/:slug/invitations/:id", teamHandler.CancelInvitation)
			teams.GET("/:slug/members/me/key", teamHandler.GetMemberKey)
			teams.POST("/:slug/members/me/key", teamHandler.CreateMemberKey)
			teams.PUT("/:slug/members/me/key", teamHandler.ToggleMemberKey)
			teams.GET("/:slug/members/me/logs", teamHandler.GetMemberLogs)
			teams.GET("/:slug/members/me/log-keys", teamHandler.GetMemberLogKeys)

			// 成员配额管理
			teams.GET("/:slug/members/:memberId/quota", teamHandler.GetMemberQuota)
			teams.PUT("/:slug/members/:memberId/quota", teamHandler.SetMemberQuota)
			teams.DELETE("/:slug/members/:memberId/quota", teamHandler.RevokeMemberQuota)
		}

		// 管理员路由
		admin := protected.Group("/admin")
		admin.Use(middleware.AdminRequired(cfg.AdminEmails))
		{
			admin.GET("/teams", adminHandler.ListTeams)
			admin.POST("/teams/:slug/recharge", adminHandler.RechargeTeam)
			admin.GET("/settings", adminHandler.ListSettings)
			admin.PUT("/settings/:key", adminHandler.UpdateSetting)
			admin.GET("/login-logs", adminHandler.ListLoginLogs)
			admin.GET("/recharge-logs", adminHandler.ListRechargeLogs)
		}
	}

	// 前端静态文件服务（SPA fallback）
	staticDir := cfg.Server.StaticDir
	if staticDir == "" {
		staticDir = "web/dist"
	}
	serveStaticFile(r, staticDir)

	// 启动服务
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("ModelGate 服务启动于 http://localhost%s", addr)

	go func() {
		if err := r.Run(addr); err != nil {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务...")
}

func initDB(cfg *config.Config) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	switch cfg.Database.Driver {
	case "mysql":
		return gorm.Open(mysql.Open(cfg.Database.DSN), gormCfg)
	case "postgres":
		return gorm.Open(postgres.Open(cfg.Database.DSN), gormCfg)
	default:
		return gorm.Open(sqlite.Open(cfg.Database.DSN), gormCfg)
	}
}

// initDefaultSettings 初始化默认站点配置项（仅在首次运行时插入）
func initDefaultSettings(db *gorm.DB) {
	defaults := []model.SiteSetting{
		{Key: "site_name", Value: "ModelGate", Comment: "站点名称"},
		{Key: "menu_arena_visible", Value: "true", Comment: "是否显示操练场菜单"},
		{Key: "menu_docs_visible", Value: "true", Comment: "是否显示文档菜单"},
		{Key: "menu_docs_url", Value: "/docs", Comment: "文档菜单跳转链接"},
	}
	for _, d := range defaults {
		var count int64
		db.Model(&model.SiteSetting{}).Where("`key` = ?", d.Key).Count(&count)
		if count == 0 {
			db.Create(&d)
		}
	}
}

// serveStaticFile 提供前端静态文件，并处理 SPA fallback
func serveStaticFile(r *gin.Engine, staticDir string) {
	absDir, err := filepath.Abs(staticDir)
	if err != nil {
		log.Printf("[WARN] 静态文件目录解析失败: %v", err)
		return
	}

	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		log.Printf("[INFO] 静态文件目录 %s 不存在，跳过前端服务（仅提供 API）", absDir)
		return
	}

	log.Printf("[INFO] 前端静态文件: %s", absDir)

	fileServer := http.FileServer(http.Dir(absDir))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// API 路径不 fallback
		if strings.HasPrefix(path, "/api/") {
			c.JSON(404, gin.H{"error": "接口不存在"})
			return
		}

		// 尝试直接提供静态文件
		fullPath := filepath.Join(absDir, path)
		if _, err := os.Stat(fullPath); err == nil {
			// 检查是否不是目录
			if info, statErr := os.Stat(fullPath); statErr == nil && !info.IsDir() {
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}

		// 检查 index.html 是否存在（SPA fallback）
		indexPath := filepath.Join(absDir, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			c.File(indexPath)
			return
		}

		c.JSON(404, gin.H{"error": "页面不存在"})
	})
}

// embedFrontend 是一个备用方案：编译时将前端文件嵌入二进制
// 使用方式: go build -tags embed
// 需要在 web/ 下编译好前端产物
var _ = fs.ReadDir // 占位，如果将来要使用 embed 功能
