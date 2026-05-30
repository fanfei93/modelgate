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
)

type AuthService struct {
	userRepo      *repository.UserRepo
	newAPIClient  *newapi.Client
}

func NewAuthService(userRepo *repository.UserRepo, newAPIClient *newapi.Client) *AuthService {
	return &AuthService{userRepo: userRepo, newAPIClient: newAPIClient}
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

	// 在 new-api 中创建对应的内部用户
	newAPIPassword := generateRandomPassword(20)
	newAPIName := fmt.Sprintf("mg_%s", username)
	if len(newAPIName) > 20 {
		newAPIName = newAPIName[:20]
	}
	newAPIEmail := fmt.Sprintf("%s@modelgate.local", username)
	newAPIUserID, err := s.newAPIClient.RegisterUserWithUsername(newAPIName, newAPIEmail, newAPIPassword)
	if err != nil {
		return nil, fmt.Errorf("创建API账户失败: %w", err)
	}
	log.Printf("[INFO] 用户 %s 的 new-api 账户创建成功 (ID=%d)", username, newAPIUserID)

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

func (s *AuthService) Login(username, password string) (*model.User, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrWrongPassword
	}
	return user, nil
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
