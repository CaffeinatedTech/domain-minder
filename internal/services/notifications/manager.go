package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/CaffeinatedTech/domain-minder/internal/config"
	"github.com/CaffeinatedTech/domain-minder/internal/database"
	"github.com/CaffeinatedTech/domain-minder/internal/models"
	"github.com/CaffeinatedTech/domain-minder/internal/services/mailer"
)

type NotificationManager struct {
	cfg           *config.Config
	emailNotifier *EmailNotifier
	tgNotifier    *TelegramNotifier
}

func NewNotificationManager(cfg *config.Config, m *mailer.Service) *NotificationManager {
	return &NotificationManager{
		cfg:           cfg,
		emailNotifier: NewEmailNotifier(cfg, m),
		tgNotifier:    NewTelegramNotifier(cfg),
	}
}

func (m *NotificationManager) CheckDomain(ctx context.Context, user *models.User, domain *models.Domain) ([]*models.NotificationLog, error) {
	if domain.Status != "active" {
		return nil, nil
	}

	daysRemaining := int(time.Until(domain.ExpiryDate).Hours() / 24)

	if daysRemaining < 0 {
		return nil, nil
	}

	thresholds, err := m.parseThresholds(user.NotificationThresholds)
	if err != nil {
		log.Printf("Failed to parse thresholds for user %d: %v", user.ID, err)
		thresholds = []int{90, 60, 30, 14, 7, 3, 1}
	}

	var logs []*models.NotificationLog

	for _, days := range thresholds {
		if daysRemaining == days {
			exists, err := database.NotificationExists(ctx, domain.ID, "email", days)
			if err != nil {
				log.Printf("Failed to check notification exists: %v", err)
				continue
			}
			if exists {
				continue
			}

			logEntry := m.sendNotification(ctx, user, domain, days)
			logs = append(logs, logEntry)
		}
	}

	if daysRemaining < 7 {
		since := time.Now().AddDate(0, 0, -1)
		recentLogs, err := database.GetRecentNotifications(ctx, user.ID, since)
		if err != nil {
			log.Printf("Failed to get recent notifications: %v", err)
		} else {
			sentToday := false
			for _, l := range recentLogs {
				if l.DomainID == domain.ID && l.NotificationType == "email" && daysRemaining == l.DaysBeforeExpiry {
					sentToday = true
					break
				}
			}
			if !sentToday {
				logEntry := m.sendNotification(ctx, user, domain, daysRemaining)
				logs = append(logs, logEntry)
			}
		}
	}

	return logs, nil
}

func (m *NotificationManager) sendNotification(ctx context.Context, user *models.User, domain *models.Domain, daysRemaining int) *models.NotificationLog {
	notification := &Notification{
		UserID:        user.ID,
		DomainID:      domain.ID,
		DomainName:    domain.Name,
		Registrar:     m.getRegistrar(domain),
		ExpiryDate:    domain.ExpiryDate.Format("January 2, 2006"),
		DaysRemaining: daysRemaining,
	}

	logEntry := &models.NotificationLog{
		DomainID:         domain.ID,
		UserID:           user.ID,
		NotificationType: "email",
		DaysBeforeExpiry: daysRemaining,
		SentAt:           time.Now(),
		Success:          false,
	}

	var emailErr error

	if user.NotificationEmail && user.EmailVerified {
		if m.emailNotifier.CanSend() {
			emailErr = m.emailNotifier.SendTo(ctx, user.Email, notification)
			if emailErr == nil {
				logEntry.Success = true
			}
		} else {
			emailErr = fmt.Errorf("SMTP not configured")
		}
	} else if user.NotificationEmail && !user.EmailVerified {
		emailErr = fmt.Errorf("email not verified")
		logEntry.ErrorMessage = stringPtr("email not verified")
	}

	if emailErr != nil {
		logEntry.Success = false
		msg := emailErr.Error()
		logEntry.ErrorMessage = &msg
	}

	database.CreateNotificationLog(ctx, logEntry)

	if user.NotificationTelegram && user.TelegramChatID != nil && *user.TelegramChatID != "" {
		tgErr := m.tgNotifier.Send(ctx, *user.TelegramChatID, notification)
		if tgErr != nil {
			log.Printf("Failed to send Telegram notification for %s: %v", domain.Name, tgErr)
		}
	}

	return logEntry
}

func (m *NotificationManager) parseThresholds(jsonStr string) ([]int, error) {
	var thresholds []int
	if err := json.Unmarshal([]byte(jsonStr), &thresholds); err != nil {
		return nil, err
	}
	return thresholds, nil
}

func (m *NotificationManager) getRegistrar(domain *models.Domain) string {
	if domain.Registrar != nil {
		return *domain.Registrar
	}
	return "Unknown"
}

func stringPtr(s string) *string {
	return &s
}
