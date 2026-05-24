package service

import (
	"errors"

	"github.com/modelgate/internal/model"
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
	userRepo *repository.UserRepo
}

func NewAuthService(userRepo *repository.UserRepo) *AuthService {
	return &AuthService{userRepo: userRepo}
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

	user := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		DisplayName:  username,
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
