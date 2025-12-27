package models

import "time"

type User struct {
	ID                     int       `json:"id"`
	Email                  string    `json:"email"`
	PasswordHash           string    `json:"-"`
	EmailVerified          bool      `json:"email_verified"`
	EmailVerificationToken *string   `json:"-"`
	TelegramChatID         *string   `json:"telegram_chat_id,omitempty"`
	NotificationEmail      bool      `json:"notification_email"`
	NotificationTelegram   bool      `json:"notification_telegram"`
	NotificationThresholds string    `json:"notification_thresholds"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type Domain struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	Name        string     `json:"name"`
	Registrar   *string    `json:"registrar,omitempty"`
	ExpiryDate  time.Time  `json:"expiry_date"`
	WHOISRaw    *string    `json:"whois_raw,omitempty"`
	LastChecked *time.Time `json:"last_checked,omitempty"`
	Status      string     `json:"status"`
	Notes       *string    `json:"notes,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type NotificationLog struct {
	ID               int       `json:"id"`
	DomainID         int       `json:"domain_id"`
	UserID           int       `json:"user_id"`
	NotificationType string    `json:"notification_type"`
	DaysBeforeExpiry int       `json:"days_before_expiry"`
	SentAt           time.Time `json:"sent_at"`
	Success          bool      `json:"success"`
	ErrorMessage     *string   `json:"error_message,omitempty"`
}
