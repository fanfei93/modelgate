package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterAdminTokenRoutes 注册管理员 Token 管理路由（供 modelgate 后端调用）
func RegisterAdminTokenRoutes(apiRouter *gin.RouterGroup) {
	adminTokenRoute := apiRouter.Group("/admin/token")
	adminTokenRoute.Use(middleware.ServiceAuth())
	{
		adminTokenRoute.POST("/create", controller.AdminCreateToken)
		adminTokenRoute.POST("/delete", controller.AdminDeleteToken)
		adminTokenRoute.POST("/status", controller.AdminUpdateTokenStatus)
		adminTokenRoute.POST("/quota", controller.AdminUpdateTokenQuota)
		adminTokenRoute.POST("/info", controller.AdminGetTokenInfo)
	}
}
