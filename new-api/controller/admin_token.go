package controller

import (
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

type AdminCreateTokenRequest struct {
	UserId         int    `json:"user_id" binding:"required"`
	Name           string `json:"name" binding:"required,max=50"`
	RemainQuota    *int   `json:"remain_quota"`
	UnlimitedQuota *bool  `json:"unlimited_quota"`
}

// AdminCreateToken 管理员为指定用户创建 API Token（返回完整 key）
func AdminCreateToken(c *gin.Context) {
	var req AdminCreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	// 检查用户令牌数量是否已达上限
	maxTokens := operation_setting.GetMaxUserTokens()
	count, err := model.CountUserTokens(req.UserId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if int(count) >= maxTokens {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("已达到最大令牌数量限制 (%d)", maxTokens),
		})
		return
	}

	key, err := common.GenerateKey()
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgTokenGenerateFailed)
		common.SysLog("failed to generate token key: " + err.Error())
		return
	}

	remainQuota := 0
	if req.RemainQuota != nil {
		remainQuota = *req.RemainQuota
	}
	unlimited := false
	if req.UnlimitedQuota != nil {
		unlimited = *req.UnlimitedQuota
	}

	cleanToken := model.Token{
		UserId:         req.UserId,
		Name:           req.Name,
		Key:            key,
		CreatedTime:    common.GetTimestamp(),
		AccessedTime:   common.GetTimestamp(),
		ExpiredTime:    -1,
		RemainQuota:    remainQuota,
		UnlimitedQuota: unlimited,
	}
	if err := cleanToken.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"id":  cleanToken.Id,
			"key": key,
		},
	})
}

type AdminDeleteTokenRequest struct {
	TokenId int `json:"token_id" binding:"required"`
}

// AdminDeleteToken 管理员删除指定 token（供 modelgate 后端调用）
func AdminDeleteToken(c *gin.Context) {
	var req AdminDeleteTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	token := model.Token{Id: req.TokenId}
	if err := token.Delete(); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

type AdminUpdateTokenStatusRequest struct {
	TokenId int `json:"token_id" binding:"required"`
	Status  int `json:"status" binding:"required"` // 1: 启用, 2: 禁用
}

// AdminUpdateTokenStatus 管理员更新 token 状态（启用/禁用）
func AdminUpdateTokenStatus(c *gin.Context) {
	var req AdminUpdateTokenStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if req.Status != common.TokenStatusEnabled && req.Status != common.TokenStatusDisabled {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	token, err := model.GetTokenById(req.TokenId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	token.Status = req.Status
	if err := token.Update(); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

type AdminUpdateTokenQuotaRequest struct {
	TokenId        int  `json:"token_id" binding:"required"`
	RemainQuota    *int `json:"remain_quota"`
	UnlimitedQuota *bool `json:"unlimited_quota"`
}

// AdminUpdateTokenQuota 管理员更新 token 的配额（remain_quota / unlimited_quota）
func AdminUpdateTokenQuota(c *gin.Context) {
	var req AdminUpdateTokenQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	token, err := model.GetTokenById(req.TokenId)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 如果全是 nil，不操作
	if req.RemainQuota == nil && req.UnlimitedQuota == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "未提供需要更新的字段",
		})
		return
	}

	if req.RemainQuota != nil {
		token.RemainQuota = *req.RemainQuota
	}
	if req.UnlimitedQuota != nil {
		token.UnlimitedQuota = *req.UnlimitedQuota
	}

	if token.UnlimitedQuota {
		token.Status = common.TokenStatusEnabled
	}

	if err := token.Update(); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"id":              token.Id,
			"remain_quota":    token.RemainQuota,
			"unlimited_quota": token.UnlimitedQuota,
		},
	})
}

type AdminGetTokenInfoRequest struct {
	TokenId int `json:"token_id" binding:"required"`
}

// AdminGetTokenInfo 管理员查询指定 token 的信息（含配额）
func AdminGetTokenInfo(c *gin.Context) {
	var req AdminGetTokenInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	token, err := model.GetTokenById(req.TokenId)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"id":              token.Id,
			"user_id":         token.UserId,
			"name":            token.Name,
			"status":          token.Status,
			"remain_quota":    token.RemainQuota,
			"unlimited_quota": token.UnlimitedQuota,
			"used_quota":      token.UsedQuota,
		},
	})
}
