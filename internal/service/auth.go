package service

import (
	"errors"
	"fmt"
	"log"

	"github.com/modelgate/internal/model"
	"github.com/modelgate/internal/newapi"
	"github.com/modelgate/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrEmailExists    = errors.New("邮箱已注册")
	ErrUserNotFound   = errors.New("用户不存在")
	ErrWrongPassword  = errors.New("密码错误")
	ErrUserDisabled   = errors.New("账户已被禁用，请联系管理员")
	ErrBindingTampered = errors.New("账户数据异常，已被锁定，请联系管理员")
)

type AuthService struct {
	userRepo     *repository.UserRepo
	newAPIClient *newapi.Client
	loginLogRepo *repository.LoginLogRepo
}

func NewAuthService(userRepo *repository.UserRepo, newAPIClient *newapi.Client, loginLogRepo *repository.LoginLogRepo) *AuthService {
	return &AuthService{userRepo: userRepo, newAPIClient: newAPIClient, loginLogRepo: loginLogRepo}
}

func (s *AuthService) Register(username, email, password string) (*model.User, error) {
	// 检查邮箱是否已存在
	if _, err := s.userRepo.FindByEmail(email); err == nil {
		return nil, ErrEmailExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	// 检查用户名是否已存在
	if _, err := s.userRepo.FindByUsername(username); err == nil {
		return nil, errors.New("用户名已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 创建内部用户
	newAPIPassword := generateRandomPassword(20)
	newAPIName := fmt.Sprintf("mg_%s", username)
	if len(newAPIName) > 20 {
		newAPIName = newAPIName[:20]
	}
	newAPIEmail := fmt.Sprintf("%s@modelgate.local", username)
	newAPIUserID, err := s.newAPIClient.RegisterUserWithUsername(newAPIName, newAPIEmail, newAPIPassword)
	if err != nil {
		log.Printf("[ERROR] 创建用户账户失败: %v", err)
		return nil, fmt.Errorf("创建用户账户失败，请稍后重试")
	}
	log.Printf("[INFO] 用户 %s 的账户创建成功 (ID=%d)", username, newAPIUserID)

	user := &model.User{
		Username:       username,
		Email:          email,
		PasswordHash:   string(hash),
		DisplayName:    username,
		NewAPIUserID:   newAPIUserID,
		NewAPIPassword: newAPIPassword,
	}
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) GetUserByID(id uint) (*model.User, error) {
	return s.userRepo.FindByID(id)
}

func (s *AuthService) Login(username, password, ip, userAgent string) (*model.User, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.recordLog(0, username, ip, userAgent, false, "用户不存在")
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// 已被禁用的用户禁止登录
	if user.Status == "disabled" {
		s.recordLog(user.ID, username, ip, userAgent, false, "账户已被禁用")
		return nil, ErrUserDisabled
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.recordLog(user.ID, username, ip, userAgent, false, "密码错误")
		return nil, ErrWrongPassword
	}

	// 密码正确后，同步校验内部用户绑定完整性
	if err := s.verifyBinding(user); err != nil {
		// 绑定异常：禁用该用户并拒绝登录
		log.Printf("[SECURITY] 用户 %s (ID=%d) 绑定校验失败，已禁用账户: %v", user.Username, user.ID, err)
		user.Status = "disabled"
		_ = s.userRepo.Update(user)
		s.recordLog(user.ID, username, ip, userAgent, false, "绑定校验失败，账户已锁定")
		return nil, ErrBindingTampered
	}

	s.recordLog(user.ID, username, ip, userAgent, true, "")
	return user, nil
}

func (s *AuthService) recordLog(userID uint, username, ip, userAgent string, success bool, reason string) {
	entry := &model.LoginLog{
		UserID:    userID,
		Username:  username,
		IP:        ip,
		UserAgent: userAgent,
		Success:   success,
		Reason:    reason,
	}
	if err := s.loginLogRepo.Create(entry); err != nil {
		log.Printf("[ERROR] 记录登录日志失败: %v", err)
	}
}

// verifyBinding 校验内部用户绑定完整性，返回 nil 表示正常
// 校验项：1. NewAPIUserID 非零  2. 用户仍存在  3. 用户名匹配  4. 密码仍有效
func (s *AuthService) verifyBinding(user *model.User) error {
	if user.NewAPIUserID == 0 {
		return fmt.Errorf("未绑定内部账户")
	}

	// 校验内部用户是否仍存在
	info, err := s.newAPIClient.GetUserInfo(user.NewAPIUserID)
	if err != nil {
		return fmt.Errorf("内部账户 (ID=%d) 无法访问: %w", user.NewAPIUserID, err)
	}

	// 校验用户名是否被篡改（防止 ID 复用攻击）
	expectedName := fmt.Sprintf("mg_%s", user.Username)
	if len(expectedName) > 20 {
		expectedName = expectedName[:20]
	}
	if info.Username != expectedName {
		return fmt.Errorf("内部账户用户名不匹配: 期望=%s 实际=%s", expectedName, info.Username)
	}

	// 校验密码是否仍有效（防止密码被重置）
	if user.NewAPIPassword != "" {
		if err := s.newAPIClient.UserLogin(info.Username, user.NewAPIPassword); err != nil {
			return fmt.Errorf("内部账户密码已失效")
		}
	}

	return nil
}

// IsEmailRegistered 检查邮箱是否已注册
func (s *AuthService) IsEmailRegistered(email string) (bool, error) {
	_, err := s.userRepo.FindByEmail(email)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}
