package notifications

import (
	"context"

	"github.com/CaffeinatedTech/domain-minder/internal/config"
	"github.com/CaffeinatedTech/domain-minder/internal/services/mailer"
)

type EmailNotifier struct {
	cfg    *config.Config
	mailer *mailer.Service
}

func NewEmailNotifier(cfg *config.Config, m *mailer.Service) *EmailNotifier {
	return &EmailNotifier{cfg: cfg, mailer: m}
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
	return n.mailer.EnqueueNotificationEmail(
		ctx,
		toEmail,
		notification.DomainName,
		notification.Registrar,
		notification.ExpiryDate,
		notification.DaysRemaining,
	)
}
