package database

import (
	"context"
	"fmt"
	"time"

	"github.com/CaffeinatedTech/domain-minder/internal/models"
)

func CreateNotificationLog(ctx context.Context, log *models.NotificationLog) (int64, error) {
	result, err := DB.ExecContext(ctx, `
        INSERT INTO notification_logs (domain_id, user_id, notification_type, days_before_expiry, success, error_message)
        VALUES (?, ?, ?, ?, ?, ?)
    `, log.DomainID, log.UserID, log.NotificationType, log.DaysBeforeExpiry, log.Success, log.ErrorMessage)
	if err != nil {
		return 0, fmt.Errorf("failed to create notification log: %w", err)
	}
	return result.LastInsertId()
}

func GetNotificationLogsByDomain(ctx context.Context, domainID int) ([]*models.NotificationLog, error) {
	rows, err := DB.QueryContext(ctx, `
        SELECT id, domain_id, user_id, notification_type, days_before_expiry, sent_at, success, error_message
        FROM notification_logs WHERE domain_id = ? ORDER BY sent_at DESC
    `, domainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.NotificationLog
	for rows.Next() {
		log := &models.NotificationLog{}
		if err := rows.Scan(&log.ID, &log.DomainID, &log.UserID, &log.NotificationType,
			&log.DaysBeforeExpiry, &log.SentAt, &log.Success, &log.ErrorMessage); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func NotificationExists(ctx context.Context, domainID int, notificationType string, daysBeforeExpiry int) (bool, error) {
	var count int
	err := DB.QueryRowContext(ctx, `
        SELECT COUNT(*) FROM notification_logs
        WHERE domain_id = ? AND notification_type = ? AND days_before_expiry = ?
    `, domainID, notificationType, daysBeforeExpiry).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func GetRecentNotifications(ctx context.Context, userID int, since time.Time) ([]*models.NotificationLog, error) {
	rows, err := DB.QueryContext(ctx, `
        SELECT id, domain_id, user_id, notification_type, days_before_expiry, sent_at, success, error_message
        FROM notification_logs WHERE user_id = ? AND sent_at >= ? ORDER BY sent_at DESC
    `, userID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent notifications: %w", err)
	}
	defer rows.Close()

	var logs []*models.NotificationLog
	for rows.Next() {
		log := &models.NotificationLog{}
		if err := rows.Scan(&log.ID, &log.DomainID, &log.UserID, &log.NotificationType,
			&log.DaysBeforeExpiry, &log.SentAt, &log.Success, &log.ErrorMessage); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}
