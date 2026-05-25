package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// ServiceAuth 服务间调用认证中间件
// 仅验证 access_token（Bearer token）和管理员权限，不检查 New-Api-User header
// 供 modelgate 等后端服务调用，不走 session
func ServiceAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		accessToken := c.Request.Header.Get("Authorization")
		if accessToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": common.TranslateMessage(c, i18n.MsgAuthNotLoggedIn),
			})
			c.Abort()
			return
		}

		user, err := model.ValidateAccessToken(accessToken)
		if err != nil {
			if errors.Is(err, model.ErrDatabase) {
				common.SysLog("ServiceAuth ValidateAccessToken database error: " + err.Error())
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": common.TranslateMessage(c, i18n.MsgDatabaseError),
				})
			} else {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": common.TranslateMessage(c, i18n.MsgAuthAccessTokenInvalid),
				})
			}
			c.Abort()
			return
		}

		if user == nil || strings.TrimSpace(user.Username) == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": common.TranslateMessage(c, i18n.MsgAuthAccessTokenInvalid),
			})
			c.Abort()
			return
		}

		if !common.IsValidateRole(user.Role) {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": common.TranslateMessage(c, i18n.MsgAuthUserInfoInvalid),
			})
			c.Abort()
			return
		}

		if user.Role < common.RoleAdminUser {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": common.TranslateMessage(c, i18n.MsgAuthInsufficientPrivilege),
			})
			c.Abort()
			return
		}

		if user.Status == common.UserStatusDisabled {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": common.TranslateMessage(c, i18n.MsgAuthUserBanned),
			})
			c.Abort()
			return
		}

		c.Set("username", user.Username)
		c.Set("role", user.Role)
		c.Set("id", user.Id)
		c.Next()
	}
}
