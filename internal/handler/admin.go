package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/modelgate/internal/middleware"
	"github.com/modelgate/internal/service"
)

type AdminHandler struct {
	adminService *service.AdminService
}

func NewAdminHandler(adminService *service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

// ListTeams 超管 - 列出所有团队
// GET /api/admin/teams
func (h *AdminHandler) ListTeams(c *gin.Context) {
	teams, err := h.adminService.ListTeams()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": teams})
}

// RechargeTeam 超管 - 为团队充值
// POST /api/admin/teams/:slug/recharge
func (h *AdminHandler) RechargeTeam(c *gin.Context) {
	slug := c.Param("slug")
	var req struct {
		Amount int64  `json:"amount"` // 充值金额（分）
		Remark string `json:"remark"` // 备注（可选）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "充值金额必须大于0"})
		return
	}

	operatorID := middleware.GetUserID(c)
	operatorName := middleware.GetUsername(c)
	ip := c.ClientIP()

	if err := h.adminService.RechargeTeam(slug, req.Amount, operatorID, operatorName, ip, req.Remark); err != nil {
		status := http.StatusBadRequest
		if err == service.ErrTeamNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "充值成功"})
}

// ListSettings 超管 - 获取所有站点配置
// GET /api/admin/settings
func (h *AdminHandler) ListSettings(c *gin.Context) {
	settings, err := h.adminService.ListSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settings})
}

// UpdateSetting 超管 - 更新站点配置
// PUT /api/admin/settings/:key
func (h *AdminHandler) UpdateSetting(c *gin.Context) {
	key := c.Param("key")
	var req struct {
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.adminService.UpdateSetting(key, req.Value); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// ListLoginLogs 超管 - 查询登录日志
// GET /api/admin/login-logs?page=1&page_size=20
func (h *AdminHandler) ListLoginLogs(c *gin.Context) {
	page := 1
	pageSize := 20
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil && ps > 0 && ps <= 100 {
		pageSize = ps
	}

	logs, total, err := h.adminService.ListLoginLogs(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取登录日志失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":     logs,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// ListRechargeLogs 超管 - 查询充值审计日志
// GET /api/admin/recharge-logs?page=1&page_size=20
func (h *AdminHandler) ListRechargeLogs(c *gin.Context) {
	page := 1
	pageSize := 20
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil && ps > 0 && ps <= 100 {
		pageSize = ps
	}

	logs, total, err := h.adminService.ListRechargeLogs(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取充值日志失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":     logs,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}
