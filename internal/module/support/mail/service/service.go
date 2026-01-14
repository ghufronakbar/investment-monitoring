// internal/module/support/mail/service/service.go
package mail_service

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/smtp"

	"investment_monitoring/config"
)

type MailService struct {
	cfg *config.Config
}

func NewService(cfg *config.Config) *MailService {
	return &MailService{
		cfg: cfg,
	}
}

// SendInvitationInput contains data needed for invitation email
type SendInvitationInput struct {
	ToEmail          string
	ToName           string
	OrganizationName string
	OrganizationRole string
	MemberRole       string // ADMIN or MEMBER
}

// SendInvitationEmail sends notification that user has been invited to an organization
func (s *MailService) SendInvitationEmail(ctx context.Context, input SendInvitationInput) error {
	subject := fmt.Sprintf("Undangan Bergabung ke %s - %s", input.OrganizationName, s.cfg.APP_NAME)

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Undangan Bergabung</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #2563eb;">Selamat Datang di %s</h2>
        
        <p>Halo <strong>%s</strong>,</p>
        
        <p>Anda telah diundang untuk bergabung ke organisasi <strong>%s</strong> sebagai <strong>%s</strong>.</p>
        
        <p>Silakan kunjungi aplikasi kami untuk melengkapi proses setup akun Anda:</p>
        
        <p style="text-align: center; margin: 30px 0;">
            <a href="%s" style="background-color: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px;">
                Setup Akun Saya
            </a>
        </p>
        
        <p>Gunakan email <strong>%s</strong> untuk login.</p>
        
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        
        <p style="color: #666; font-size: 12px;">
            Email ini dikirim oleh %s.<br>
            Jika Anda tidak merasa diundang, abaikan email ini.
        </p>
    </div>
</body>
</html>
`, s.cfg.APP_NAME, input.ToName, input.OrganizationName, input.MemberRole, s.cfg.WEB_URL, input.ToEmail, s.cfg.APP_NAME)

	return s.sendEmail(input.ToEmail, subject, body)
}

// SendPasswordSetupInput contains data for password setup email
type SendPasswordSetupInput struct {
	ToEmail string
	ToName  string
}

// SendPasswordSetupEmail sends reminder to set up password
func (s *MailService) SendPasswordSetupEmail(ctx context.Context, input SendPasswordSetupInput) error {
	subject := fmt.Sprintf("Setup Password Anda - %s", s.cfg.APP_NAME)

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Setup Password</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #2563eb;">Setup Password Anda</h2>
        
        <p>Halo <strong>%s</strong>,</p>
        
        <p>Silakan setup password untuk akun Anda dengan mengunjungi link berikut:</p>
        
        <p style="text-align: center; margin: 30px 0;">
            <a href="%s" style="background-color: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px;">
                Setup Password
            </a>
        </p>
        
        <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
        
        <p style="color: #666; font-size: 12px;">
            Email ini dikirim oleh %s.
        </p>
    </div>
</body>
</html>
`, input.ToName, s.cfg.WEB_URL, s.cfg.APP_NAME)

	return s.sendEmail(input.ToEmail, subject, body)
}

// sendEmail is the internal function to send emails via SMTP
func (s *MailService) sendEmail(to, subject, htmlBody string) error {
	from := s.cfg.SMTP_SENDER

	// Construct the email message with proper headers
	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("From: %s <%s>\r\n", s.cfg.SMTP_NAME, from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	// Create SMTP auth
	auth := smtp.PlainAuth("", s.cfg.SMTP_USER, s.cfg.SMTP_PASS, s.cfg.SMTP_HOST)

	// Send the email
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTP_HOST, s.cfg.SMTP_PORT)
	err := smtp.SendMail(addr, auth, from, []string{to}, msg.Bytes())
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// Helper function to render template (for future use)
func renderTemplate(tmpl string, data interface{}) (string, error) {
	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
