package email

import (
	"testing"
)

func TestEmailService_GenerateVerificationEmail(t *testing.T) {
	svc := NewEmailService(EmailConfig{
		SMTPHost:   "smtp.example.com",
		SMTPPort:   587,
		SMTPUser:   "test@example.com",
		SMTPPass:   "password",
		FromEmail:  "noreply@example.com",
		AppBaseURL: "http://localhost:5173",
	})

	tests := []struct {
		name        string
		email       string
		verifyToken string
		wantErr     bool
	}{
		{
			name:        "valid email and token",
			email:       "user@example.com",
			verifyToken: "verify-token-123",
			wantErr:     false,
		},
		{
			name:        "empty email",
			email:       "",
			verifyToken: "verify-token-123",
			wantErr:     true,
		},
		{
			name:        "empty token",
			email:       "user@example.com",
			verifyToken: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject, body, err := svc.GenerateVerificationEmail(tt.email, tt.verifyToken)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if subject == "" {
				t.Error("expected non-empty subject")
			}
			if body == "" {
				t.Error("expected non-empty body")
			}
			// Body should contain the verify link
			expectedLink := "http://localhost:5173/verify?token=verify-token-123"
			if !containsString(body, expectedLink) {
				t.Errorf("body should contain verify link: %s", expectedLink)
			}
		})
	}
}

func TestEmailService_NewWithDefaults(t *testing.T) {
	svc := NewEmailService(EmailConfig{
		SMTPHost:   "smtp.example.com",
		SMTPPort:   587,
		SMTPUser:   "test@example.com",
		SMTPPass:   "password",
		FromEmail:  "noreply@example.com",
		AppBaseURL: "http://localhost:5173",
	})

	if svc == nil {
		t.Error("expected non-nil service")
	}
}

func TestEmailService_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  EmailConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: EmailConfig{
				SMTPHost:   "smtp.example.com",
				SMTPPort:   587,
				SMTPUser:   "test@example.com",
				SMTPPass:   "password",
				FromEmail:  "noreply@example.com",
				AppBaseURL: "http://localhost:5173",
			},
			wantErr: false,
		},
		{
			name: "missing SMTP host",
			config: EmailConfig{
				SMTPPort:   587,
				SMTPUser:   "test@example.com",
				SMTPPass:   "password",
				FromEmail:  "noreply@example.com",
				AppBaseURL: "http://localhost:5173",
			},
			wantErr: true,
		},
		{
			name: "missing from email",
			config: EmailConfig{
				SMTPHost:   "smtp.example.com",
				SMTPPort:   587,
				SMTPUser:   "test@example.com",
				SMTPPass:   "password",
				AppBaseURL: "http://localhost:5173",
			},
			wantErr: true,
		},
		{
			name: "missing app base URL",
			config: EmailConfig{
				SMTPHost:  "smtp.example.com",
				SMTPPort:  587,
				SMTPUser:  "test@example.com",
				SMTPPass:  "password",
				FromEmail: "noreply@example.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// MockEmailSender for testing SendVerificationEmail
type MockEmailSender struct {
	SentEmails []SentEmail
	ShouldFail bool
}

type SentEmail struct {
	To      string
	Subject string
	Body    string
}

func (m *MockEmailSender) Send(to, subject, body string) error {
	if m.ShouldFail {
		return ErrSendFailed
	}
	m.SentEmails = append(m.SentEmails, SentEmail{
		To:      to,
		Subject: subject,
		Body:    body,
	})
	return nil
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
