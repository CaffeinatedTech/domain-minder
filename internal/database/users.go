package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/CaffeinatedTech/domain-minder/internal/models"
)

func CreateUser(ctx context.Context, user *models.User) (int64, error) {
	result, err := DB.ExecContext(ctx, `
        INSERT INTO users (email, password_hash, email_verified, email_verification_token,
            telegram_chat_id, notification_email, notification_telegram, notification_thresholds)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `, user.Email, user.PasswordHash, user.EmailVerified, user.EmailVerificationToken,
		user.TelegramChatID, user.NotificationEmail, user.NotificationTelegram,
		user.NotificationThresholds)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}
	return result.LastInsertId()
}

func GetUserByID(ctx context.Context, id int) (*models.User, error) {
	user := &models.User{}
	err := DB.QueryRowContext(ctx, `
        SELECT id, email, password_hash, email_verified, email_verification_token,
            telegram_chat_id, notification_email, notification_telegram, notification_thresholds,
            created_at, updated_at
        FROM users WHERE id = ?
    `, id).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.EmailVerified,
		&user.EmailVerificationToken, &user.TelegramChatID, &user.NotificationEmail,
		&user.NotificationTelegram, &user.NotificationThresholds, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	err := DB.QueryRowContext(ctx, `
        SELECT id, email, password_hash, email_verified, email_verification_token,
            telegram_chat_id, notification_email, notification_telegram, notification_thresholds,
            created_at, updated_at
        FROM users WHERE email = ?
    `, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.EmailVerified,
		&user.EmailVerificationToken, &user.TelegramChatID, &user.NotificationEmail,
		&user.NotificationTelegram, &user.NotificationThresholds, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, nil
}

func GetUserByVerificationToken(ctx context.Context, token string) (*models.User, error) {
	user := &models.User{}
	err := DB.QueryRowContext(ctx, `
        SELECT id, email, password_hash, email_verified, email_verification_token,
            telegram_chat_id, notification_email, notification_telegram, notification_thresholds,
            created_at, updated_at
        FROM users WHERE email_verification_token = ?
    `, token).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.EmailVerified,
		&user.EmailVerificationToken, &user.TelegramChatID, &user.NotificationEmail,
		&user.NotificationTelegram, &user.NotificationThresholds, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by token: %w", err)
	}
	return user, nil
}

func UpdateUserVerification(ctx context.Context, id int, verified bool) error {
	_, err := DB.ExecContext(ctx, `
        UPDATE users SET email_verified = ?, email_verification_token = NULL, updated_at = ?
        WHERE id = ?
    `, verified, time.Now(), id)
	return err
}

func UpdateUser(ctx context.Context, user *models.User) error {
	_, err := DB.ExecContext(ctx, `
        UPDATE users SET email = ?, password_hash = ?, email_verified = ?,
            telegram_chat_id = ?, notification_email = ?, notification_telegram = ?,
            notification_thresholds = ?, updated_at = ?
        WHERE id = ?
    `, user.Email, user.PasswordHash, user.EmailVerified, user.TelegramChatID,
		user.NotificationEmail, user.NotificationTelegram, user.NotificationThresholds,
		time.Now(), user.ID)
	return err
}
