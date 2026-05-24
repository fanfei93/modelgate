package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/modelgate/internal/middleware"
	"github.com/modelgate/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
	teamService *service.TeamService
	jwtSecret   string
	jwtExpire   int // hours
}

func NewAuthHandler(authService *service.AuthService, teamService *service.TeamService, jwtSecret string, jwtExpire int) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		teamService: teamService,
		jwtSecret:   jwtSecret,
		jwtExpire:   jwtExpire,
	}
}

// Register 用户注册
// POST /api/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3,max=64"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6,max=128"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	user, err := h.authService.Register(req.Username, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 处理该邮箱的待处理邀请（异步，不阻塞注册）
	go h.teamService.ProcessInvitations(req.Email, user.ID)

	token, err := h.generateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成 Token 失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":           user.ID,
			"username":     user.Username,
			"email":        user.Email,
			"display_name": user.DisplayName,
		},
	})
}

// Login 用户登录
// POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	user, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	token, err := h.generateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成 Token 失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":           user.ID,
			"username":     user.Username,
			"email":        user.Email,
			"display_name": user.DisplayName,
		},
	})
}

// Me 获取当前用户信息
// GET /api/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	user, err := h.authService.GetUserByID(middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":           user.ID,
			"username":     user.Username,
			"email":        user.Email,
			"display_name": user.DisplayName,
		},
	})
}

func (h *AuthHandler) generateToken(userID uint, username string) (string, error) {
	claims := &middleware.Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(h.jwtExpire) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}
