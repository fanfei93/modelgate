package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/modelgate/internal/middleware"
	"github.com/modelgate/internal/newapi"
	"github.com/modelgate/internal/service"
)

type TeamHandler struct {
	teamService *service.TeamService
}

func NewTeamHandler(teamService *service.TeamService) *TeamHandler {
	return &TeamHandler{teamService: teamService}
}

// CreateTeam 创建团队
// POST /api/teams
func (h *TeamHandler) CreateTeam(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required,max=128"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	team, err := h.teamService.CreateTeam(userID, req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": team})
}

// GetTeams 获取我的团队列表
// GET /api/teams
func (h *TeamHandler) GetTeams(c *gin.Context) {
	userID := middleware.GetUserID(c)
	teams, err := h.teamService.GetUserTeams(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": teams})
}

// GetTeam 获取团队详情
// GET /api/teams/:slug
func (h *TeamHandler) GetTeam(c *gin.Context) {
	slug := c.Param("slug")
	team, err := h.teamService.GetTeamBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": team})
}

// DeleteTeam 解散团队
// DELETE /api/teams/:slug
func (h *TeamHandler) DeleteTeam(c *gin.Context) {
	slug := c.Param("slug")
	userID := middleware.GetUserID(c)

	if err := h.teamService.DeleteTeam(userID, slug); err != nil {
		status := http.StatusBadRequest
		if err == service.ErrTeamNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrNotTeamOwner {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "团队已解散"})
}

// CreateMemberKey 为当前成员创建个人 API Key
// POST /api/teams/:slug/members/me/key
func (h *TeamHandler) CreateMemberKey(c *gin.Context) {
	slug := c.Param("slug")
	userID := middleware.GetUserID(c)

	token, err := h.teamService.CreateMemberAPIKey(userID, slug)
	if err != nil {
		status := http.StatusBadRequest
		if err == service.ErrTeamNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrNotMember {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"key": token}})
}

// GetMemberKey 获取当前成员已有的 API Key
// GET /api/teams/:slug/members/me/key
func (h *TeamHandler) GetMemberKey(c *gin.Context) {
	slug := c.Param("slug")
	userID := middleware.GetUserID(c)

	key, err := h.teamService.GetMemberKey(userID, slug)
	if err != nil {
		status := http.StatusBadRequest
		if err == service.ErrTeamNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrNotMember {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"key": key}})
}

// ToggleMemberKey 切换当前成员的 API Key 状态（启用/禁用）
// PUT /api/teams/:slug/members/me/key
func (h *TeamHandler) ToggleMemberKey(c *gin.Context) {
	slug := c.Param("slug")
	userID := middleware.GetUserID(c)

	newStatus, err := h.teamService.ToggleMemberKey(userID, slug)
	if err != nil {
		status := http.StatusBadRequest
		if err == service.ErrTeamNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrNotMember {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"key_status": newStatus}})
}

// GetMyAPIKeys 获取当前用户在所有团队的 API Key 信息
// GET /api/me/api-keys
func (h *TeamHandler) GetMyAPIKeys(c *gin.Context) {
	userID := middleware.GetUserID(c)
	keys, err := h.teamService.GetUserAPIKeys(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": keys})
}

// --- 用户级别 API Key（无需团队 slug，支持多 Key） ---

// userKeyResponse 用户 API Key 列表项返回
type userKeyResponse struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	Name      string    `json:"name"`
	KeyMask   string    `json:"key_mask"`
	Status    int       `json:"status"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

// CreateUserKey 为用户创建 API Key
// POST /api/me/keys
func (h *TeamHandler) CreateUserKey(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"max=128"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	userID := middleware.GetUserID(c)
	key, err := h.teamService.CreateUserAPIKey(userID, req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"id":         key.ID,
		"user_id":    key.UserID,
		"name":       key.Name,
		"key":        key.Key,
		"key_mask":   maskKey(key.Key),
		"status":     key.Status,
		"created_at": key.CreatedAt,
	}})
}

// ListUserKeys 获取用户的 API Key 列表
// GET /api/me/keys
func (h *TeamHandler) ListUserKeys(c *gin.Context) {
	userID := middleware.GetUserID(c)
	keys, err := h.teamService.ListUserAPIKeys(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]userKeyResponse, len(keys))
	for i, k := range keys {
		result[i] = userKeyResponse{
			ID:        k.ID,
			UserID:    k.UserID,
			Name:      k.Name,
			KeyMask:   maskKey(k.Key),
			Status:    k.Status,
			CreatedAt: k.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: k.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

// GetUserKey 获取单个 API Key 完整信息（含完整密钥）
// GET /api/me/keys/:id
func (h *TeamHandler) GetUserKey(c *gin.Context) {
	keyID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Key ID 无效"})
		return
	}

	userID := middleware.GetUserID(c)
	k, err := h.teamService.GetUserAPIKey(userID, uint(keyID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"id":         k.ID,
		"user_id":    k.UserID,
		"name":       k.Name,
		"key":        k.Key,
		"key_mask":   maskKey(k.Key),
		"status":     k.Status,
		"created_at": k.CreatedAt,
	}})
}

// ToggleUserKey 切换用户 API Key 状态
// PUT /api/me/keys/:id/toggle
func (h *TeamHandler) ToggleUserKey(c *gin.Context) {
	keyID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Key ID 无效"})
		return
	}

	userID := middleware.GetUserID(c)
	newStatus, err := h.teamService.ToggleUserAPIKey(userID, uint(keyID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"key_status": newStatus}})
}

// DeleteUserKey 删除用户 API Key
// DELETE /api/me/keys/:id
func (h *TeamHandler) DeleteUserKey(c *gin.Context) {
	keyID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Key ID 无效"})
		return
	}

	userID := middleware.GetUserID(c)
	if err := h.teamService.DeleteUserAPIKey(userID, uint(keyID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "API Key 已删除"})
}

// AddMembers 批量添加团队成员（仅 owner）
// POST /api/teams/:slug/members
func (h *TeamHandler) AddMembers(c *gin.Context) {
	slug := c.Param("slug")
	var req struct {
		Entries []service.MemberEntry `json:"entries" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	userID := middleware.GetUserID(c)
	added, failed, err := h.teamService.AddMembers(userID, slug, req.Entries)
	if err != nil {
		status := http.StatusBadRequest
		if err == service.ErrTeamNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrNotTeamOwner {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"added":  added,
		"failed": failed,
	})
}

// RemoveMember 移除团队成员
// DELETE /api/teams/:slug/members/:memberId
func (h *TeamHandler) RemoveMember(c *gin.Context) {
	slug := c.Param("slug")
	memberID, err := strconv.ParseUint(c.Param("memberId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "成员 ID 无效"})
		return
	}

	userID := middleware.GetUserID(c)
	if err := h.teamService.RemoveMember(userID, slug, uint(memberID)); err != nil {
		status := http.StatusBadRequest
		if err == service.ErrTeamNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrNotTeamOwner {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "成员已移除"})
}

// CancelInvitation 取消待处理邀请
// DELETE /api/teams/:slug/invitations/:id
func (h *TeamHandler) CancelInvitation(c *gin.Context) {
	slug := c.Param("slug")
	invID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邀请 ID 无效"})
		return
	}

	userID := middleware.GetUserID(c)
	if err := h.teamService.CancelInvitation(userID, slug, uint(invID)); err != nil {
		status := http.StatusBadRequest
		if err == service.ErrTeamNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrNotTeamOwner {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "邀请已取消"})
}

// GetMemberQuota 获取成员额度信息
// GET /api/teams/:slug/members/:memberId/quota
func (h *TeamHandler) GetMemberQuota(c *gin.Context) {
	slug := c.Param("slug")
	memberID, err := strconv.ParseUint(c.Param("memberId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "成员 ID 无效"})
		return
	}

	info, err := h.teamService.GetMemberQuotaInfo(slug, uint(memberID))
	if err != nil {
		status := http.StatusBadRequest
		if err == service.ErrTeamNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrNotMember {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": info})
}

// SetMemberQuota 设置成员额度（仅 owner）
// PUT /api/teams/:slug/members/:memberId/quota
func (h *TeamHandler) SetMemberQuota(c *gin.Context) {
	slug := c.Param("slug")
	memberID, err := strconv.ParseUint(c.Param("memberId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "成员 ID 无效"})
		return
	}

	var req struct {
		Amount int64 `json:"amount"` // 分配额度（分）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "额度必须大于0"})
		return
	}

	userID := middleware.GetUserID(c)
	if err := h.teamService.SetMemberQuota(userID, slug, uint(memberID), req.Amount); err != nil {
		status := http.StatusBadRequest
		if err == service.ErrTeamNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrNotTeamOwner {
			status = http.StatusForbidden
		} else if errors.Is(err, service.ErrInsufficientBalance) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "额度设置成功"})
}

// RevokeMemberQuota 回收成员额度（仅 owner）
// DELETE /api/teams/:slug/members/:memberId/quota
func (h *TeamHandler) RevokeMemberQuota(c *gin.Context) {
	slug := c.Param("slug")
	memberID, err := strconv.ParseUint(c.Param("memberId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "成员 ID 无效"})
		return
	}

	userID := middleware.GetUserID(c)
	if err := h.teamService.RevokeMemberQuota(userID, slug, uint(memberID)); err != nil {
		status := http.StatusBadRequest
		if err == service.ErrTeamNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrNotTeamOwner {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "额度已回收"})
}

// GetMemberLogs 获取当前成员的调用日志
// GET /api/teams/:slug/members/me/logs?page=1&page_size=20&token_id=&token_name=&model_name=&start_timestamp=&end_timestamp=
func (h *TeamHandler) GetMemberLogs(c *gin.Context) {
	slug := c.Param("slug")
	userID := middleware.GetUserID(c)

	var q newapi.LogsQuery
	q.Page, _ = strconv.Atoi(c.Query("page"))
	q.PageSize, _ = strconv.Atoi(c.Query("page_size"))
	q.TokenID, _ = strconv.Atoi(c.Query("token_id"))
	q.TokenName = c.Query("token_name")
	q.ModelName = c.Query("model_name")
	q.StartTimestamp, _ = strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	q.EndTimestamp, _ = strconv.ParseInt(c.Query("end_timestamp"), 10, 64)

	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}

	result, err := h.teamService.GetMemberLogs(userID, slug, q)
	if err != nil {
		status := http.StatusBadRequest
		if err == service.ErrTeamNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrNotMember {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// GetMemberLogKeys 获取当前成员可用于日志筛选的 Key 列表
// GET /api/teams/:slug/members/me/log-keys
func (h *TeamHandler) GetMemberLogKeys(c *gin.Context) {
	slug := c.Param("slug")
	userID := middleware.GetUserID(c)

	keys, err := h.teamService.GetMemberLogKeys(userID, slug)
	if err != nil {
		status := http.StatusBadRequest
		if err == service.ErrTeamNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrNotMember {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": keys})
}

func maskKey(key string) string {
	if len(key) <= 10 {
		return key
	}
	return key[:5] + "****" + key[len(key)-4:]
}
