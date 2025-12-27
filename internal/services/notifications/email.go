package notifications

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/CaffeinatedTech/domain-minder/internal/config"
)

type EmailNotifier struct {
	cfg *config.Config
}

func NewEmailNotifier(cfg *config.Config) *EmailNotifier {
	return &EmailNotifier{cfg: cfg}
}

func (n *EmailNotifier) Name() string {
	return "email"
}

func (n *EmailNotifier) CanSend() bool {
	return n.cfg.SMTPConfig.Host != "" && n.cfg.SMTPConfig.Username != ""
}

func (n *EmailNotifier) Send(ctx context.Context, notification *Notification) error {
	return nil
}

func (n *EmailNotifier) SendTo(ctx context.Context, toEmail string, notification *Notification) error {
	if !n.CanSend() {
		return fmt.Errorf("SMTP not configured")
	}

	auth := smtp.PlainAuth(
		"",
		n.cfg.SMTPConfig.Username,
		n.cfg.SMTPConfig.Password,
		n.cfg.SMTPConfig.Host,
	)

	subject := fmt.Sprintf("Domain %s expires in %d days", notification.DomainName, notification.DaysRemaining)

	body := fmt.Sprintf(`<html>
<body>
<h2>Domain Expiry Warning: %s</h2>
<p>Your domain <strong>%s</strong> registered with <strong>%s</strong></p>
<p>will expire in <strong>%d days</strong> on %s.</p>
<p>Log in to your Domain Minder dashboard for more details.</p>
</body>
</html>`,
		notification.DomainName,
		notification.DomainName,
		notification.Registrar,
		notification.DaysRemaining,
		notification.ExpiryDate,
	)

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		n.cfg.SMTPConfig.From,
		toEmail,
		subject,
		body,
	)

	addr := fmt.Sprintf("%s:%d", n.cfg.SMTPConfig.Host, n.cfg.SMTPConfig.Port)

	return smtp.SendMail(addr, auth, n.cfg.SMTPConfig.From, []string{toEmail}, []byte(msg))
}
