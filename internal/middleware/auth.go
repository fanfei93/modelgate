package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func AuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未提供认证信息"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "认证格式错误"})
			return
		}

		token, err := jwt.ParseWithClaims(parts[1], &Claims{},
			func(t *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "认证信息无效或已过期"})
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "认证信息解析失败"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// GetUserID 从 context 中获取当前用户 ID
func GetUserID(c *gin.Context) uint {
	id, _ := c.Get("user_id")
	return id.(uint)
}

// GetUsername 从 context 中获取当前用户名
func GetUsername(c *gin.Context) string {
	name, _ := c.Get("username")
	return name.(string)
}
