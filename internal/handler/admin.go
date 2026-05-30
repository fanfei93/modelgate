package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
		Amount int64 `json:"amount"` // 充值金额（分）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "充值金额必须大于0"})
		return
	}

	if err := h.adminService.RechargeTeam(slug, req.Amount); err != nil {
		status := http.StatusBadRequest
		if err == service.ErrTeamNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "充值成功"})
}
