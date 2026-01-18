package email

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/smtp"
)

var (
	ErrSendFailed = errors.New("failed to send email")
)

// EmailConfig holds SMTP configuration
type EmailConfig struct {
	SMTPHost   string
	SMTPPort   int
	SMTPUser   string
	SMTPPass   string
	FromEmail  string
	AppBaseURL string
}

// Sender interface for sending emails (useful for mocking in tests)
type Sender interface {
	Send(to, subject, body string) error
}

// EmailService handles email sending
type EmailService struct {
	config EmailConfig
}

// NewEmailService creates a new email service
func NewEmailService(config EmailConfig) *EmailService {
	return &EmailService{
		config: config,
	}
}

// GenerateVerificationEmail generates the verification email content
func (s *EmailService) GenerateVerificationEmail(email, verifyToken string) (subject, body string, err error) {
	if email == "" {
		return "", "", errors.New("email is required")
	}
	if verifyToken == "" {
		return "", "", errors.New("verify_token is required")
	}

	subject = "验证您的 KubeAgents 账户"
	verifyLink := fmt.Sprintf("%s/verify?token=%s", s.config.AppBaseURL, verifyToken)

	body = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>验证您的账户</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #2563eb;">欢迎加入 KubeAgents！</h1>
        <p>感谢您注册 KubeAgents 账户。请点击下面的链接验证您的邮箱地址：</p>
        <p style="margin: 30px 0;">
            <a href="%s" style="background-color: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px;">
                验证邮箱
            </a>
        </p>
        <p>或者复制以下链接到浏览器：</p>
        <p style="word-break: break-all; color: #666;">%s</p>
        <p style="margin-top: 30px; color: #666; font-size: 14px;">
            如果您没有注册 KubeAgents 账户，请忽略此邮件。
        </p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        <p style="color: #999; font-size: 12px;">
            此邮件由 KubeAgents 系统自动发送，请勿回复。
        </p>
    </div>
</body>
</html>`, verifyLink, verifyLink)

	return subject, body, nil
}

// SendVerificationEmail sends a verification email to the user
func (s *EmailService) SendVerificationEmail(toEmail, verifyToken string) error {
	subject, body, err := s.GenerateVerificationEmail(toEmail, verifyToken)
	if err != nil {
		return err
	}

	// Log user and verification link
	verifyLink := fmt.Sprintf("%s/verify?token=%s", s.config.AppBaseURL, verifyToken)
	log.Printf("[EMAIL] User: %s, Verification link: %s", toEmail, verifyLink)

	err = s.sendMail(toEmail, subject, body)
	if err != nil {
		return err
	}

	return nil
}

// sendMail sends an email using SMTP
func (s *EmailService) sendMail(to, subject, body string) error {
	from := s.config.FromEmail
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	// Build email message
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	msg := []byte(fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: %s\r\n%s\r\n%s",
		to, from, subject, mime, body))

	// Create auth if credentials provided
	var auth smtp.Auth
	if s.config.SMTPUser != "" && s.config.SMTPPass != "" {
		auth = smtp.PlainAuth("", s.config.SMTPUser, s.config.SMTPPass, s.config.SMTPHost)
	}

	// Check if using SSL (port 465) or STARTTLS (port 587)
	var err error
	if s.config.SMTPPort == 465 {
		err = s.sendMailSSL(addr, auth, from, to, msg)
	} else {
		err = smtp.SendMail(addr, auth, from, []string{to}, msg)
	}

	if err != nil {
		return fmt.Errorf("%w: %v", ErrSendFailed, err)
	}

	return nil
}

// sendMailSSL sends email using direct SSL/TLS connection (for port 465)
func (s *EmailService) sendMailSSL(addr string, auth smtp.Auth, from, to string, msg []byte) error {
	// Create TLS configuration
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         s.config.SMTPHost,
	}

	// Connect to SMTP server with TLS
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, s.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Authenticate if credentials provided
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("auth failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipient
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send email body
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = writer.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}
