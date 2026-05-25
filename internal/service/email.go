package service

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/smtp"
	"strings"
	"time"

	"github.com/modelgate/internal/config"
	"github.com/modelgate/internal/model"
	"github.com/modelgate/internal/repository"
)

type EmailService struct {
	cfg     config.SMTPConfig
	vcRepo  *repository.VerificationCodeRepo
}

func NewEmailService(cfg config.SMTPConfig, vcRepo *repository.VerificationCodeRepo) *EmailService {
	return &EmailService{cfg: cfg, vcRepo: vcRepo}
}

// SendVerificationCode 生成验证码并发送到指定邮箱
func (s *EmailService) SendVerificationCode(toEmail string) error {
	code := fmt.Sprintf("%06d", rand.Intn(1000000))

	vc := &model.VerificationCode{
		Email:     toEmail,
		Code:      code,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := s.vcRepo.Upsert(vc); err != nil {
		return fmt.Errorf("保存验证码失败: %w", err)
	}

	fromName := s.cfg.FromName
	if fromName == "" {
		fromName = "ModelGate"
	}
	subject := fmt.Sprintf("%s 邮箱验证码", fromName)
	body := fmt.Sprintf("您的验证码是: %s，有效期10分钟。", code)

	return s.sendMail(toEmail, subject, body)
}

// VerifyCode 校验验证码
func (s *EmailService) VerifyCode(email, code string) error {
	vc, err := s.vcRepo.FindByEmail(email)
	if err != nil {
		return fmt.Errorf("验证码不存在或已过期")
	}
	if vc.Code != code {
		return fmt.Errorf("验证码错误")
	}
	if time.Now().After(vc.ExpiresAt) {
		s.vcRepo.DeleteByEmail(email)
		return fmt.Errorf("验证码已过期")
	}
	// 验证通过后删除
	s.vcRepo.DeleteByEmail(email)
	return nil
}

func (s *EmailService) sendMail(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	from := s.cfg.Username

	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body))

	// 尝试 TLS 连接
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)

	// 先用 TLS 连接
	tlsConfig := &tls.Config{
		ServerName:         s.cfg.Host,
		InsecureSkipVerify: false,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		// TLS 失败，尝试 STARTTLS
		client, err2 := smtp.Dial(addr)
		if err2 != nil {
			return fmt.Errorf("连接 SMTP 服务器失败: %w (tls: %v)", err2, err)
		}
		defer client.Close()

		if err := client.Hello("localhost"); err != nil {
			return fmt.Errorf("SMTP HELO 失败: %w", err)
		}
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("SMTP STARTTLS 失败: %w", err)
			}
		}
		if err := client.Auth(auth); err != nil {
			// 忽略认证错误，尝试不认证发送
			if !strings.Contains(err.Error(), "not supported") {
				return fmt.Errorf("SMTP 认证失败: %w", err)
			}
		}
		if err := client.Mail(from); err != nil {
			return fmt.Errorf("SMTP MAIL FROM 失败: %w", err)
		}
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("SMTP RCPT TO 失败: %w", err)
		}
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("SMTP DATA 失败: %w", err)
		}
		_, err = w.Write(msg)
		if err != nil {
			return fmt.Errorf("发送邮件内容失败: %w", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("关闭邮件数据失败: %w", err)
		}
		return client.Quit()
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		if !strings.Contains(err.Error(), "not supported") {
			return fmt.Errorf("SMTP 认证失败: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM 失败: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT TO 失败: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA 失败: %w", err)
	}
	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("发送邮件内容失败: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("关闭邮件数据失败: %w", err)
	}
	return client.Quit()
}

// IsConfigured 检查 SMTP 是否已配置
func (s *EmailService) IsConfigured() bool {
	return s.cfg.Host != "" && s.cfg.Port > 0
}

// SendInvitationEmail 发送团队邀请邮件
func (s *EmailService) SendInvitationEmail(toEmail, teamName, registerURL string) error {
	fromName := s.cfg.FromName
	if fromName == "" {
		fromName = "ModelGate"
	}
	subject := fmt.Sprintf("您被邀请加入团队「%s」", teamName)
	body := fmt.Sprintf(`您好！

您被邀请加入 ModelGate 团队「%s」。

请点击以下链接完成注册并加入团队：
%s

如果链接无法点击，请将上述地址复制到浏览器中打开。

此致
%s`, teamName, registerURL, fromName)

	return s.sendMail(toEmail, subject, body)
}
