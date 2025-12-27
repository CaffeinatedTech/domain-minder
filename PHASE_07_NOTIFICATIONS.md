# Phase 7: Notifications

## What Should Already Exist

- `/home/adam/projects/domain-minder/internal/auth/email.go` with email sending functions
- `/home/adam/projects/domain-minder/internal/database/notifications.go` with notification log functions
- `/home/adam/projects/domain-minder/internal/models/models.go` with NotificationLog struct

## Context

This phase implements the modular notification system with email and Telegram channels. The system loads notification thresholds from user's JSON configuration, checks domains against thresholds, and sends notifications while respecting user preferences and email verification status.

## Agent Rules

1. Create files ONLY in `/home/adam/projects/domain-minder/internal/services/notifications/`
2. Create a Notifier interface with Send method
3. Implement EmailNotifier and TelegramNotifier
4. NotificationManager loads thresholds from user's notification_thresholds JSON column
5. Skip email notifications if user.EmailVerified is false
6. Log all notifications (success/failure) to notification_logs table
7. Prevent duplicate notifications for same domain/threshold
8. Use context for all operations with timeouts
9. Do NOT create background jobs yet - only the notification system
10. Do NOT add new dependencies beyond standard library and already-added packages

## Tasks

### 7.1 Create Notifier Interface

Create `/home/adam/projects/domain-minder/internal/services/notifications/notifier.go`:

```go
package notifications

import (
    "context"
)

type Notification struct {
    UserID           int
    DomainID         int
    DomainName       string
    Registrar        string
    ExpiryDate       string
    DaysRemaining    int
    Channel          string
}

type Notifier interface {
    Send(ctx context.Context, notification *Notification) error
    Name() string
    CanSend() bool
}
```

### 7.2 Implement Email Notifier

Create `/home/adam/projects/domain-minder/internal/services/notifications/email.go`:

```go
package notifications

import (
    "context"
    "fmt"
    "net/smtp"
    "strings"
    "github.com/yourusername/domain-minder/internal/config"
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
        "", // Will be replaced
        subject,
        body,
    )

    // Note: In a real implementation, you'd fetch the user's email from the database
    // For now, we assume the notification includes the email or we fetch it

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
```

### 7.3 Implement Telegram Notifier

Create `/home/adam/projects/domain-minder/internal/services/notifications/telegram.go`:

```go
package notifications

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    "github.com/yourusername/domain-minder/internal/config"
)

type TelegramNotifier struct {
    cfg    *config.Config
    client *http.Client
}

type TelegramMessage struct {
    ChatID              string            `json:"chat_id"`
    Text                string            `json:"text"`
    ParseMode           string            `json:"parse_mode,omitempty"`
    ReplyMarkup         *TelegramKeyboard `json:"reply_markup,omitempty"`
}

type TelegramKeyboard struct {
    InlineKeyboard [][]TelegramButton `json:"inline_keyboard"`
}

type TelegramButton struct {
    Text string `json:"text"`
    URL  string `json:"url,omitempty"`
}

func NewTelegramNotifier(cfg *config.Config) *TelegramNotifier {
    return &TelegramNotifier{
        cfg: cfg,
        client: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

func (n *TelegramNotifier) Name() string {
    return "telegram"
}

func (n *TelegramNotifier) CanSend() bool {
    return n.cfg.TelegramConfig.BotToken != ""
}

func (n *TelegramNotifier) Send(ctx context.Context, chatID string, notification *Notification) error {
    if !n.CanSend() {
        return fmt.Errorf("Telegram bot token not configured")
    }

    if chatID == "" {
        return fmt.Errorf("Telegram chat ID not set")
    }

    text := fmt.Sprintf(`*Domain Expiry Warning*

Domain: *%s*
Registrar: %s
Expires in: *%d days* (%s)

🔔 Don't forget to renew!`,
        notification.DomainName,
        notification.Registrar,
        notification.DaysRemaining,
        notification.ExpiryDate,
    )

    msg := TelegramMessage{
        ChatID:    chatID,
        Text:      text,
        ParseMode: "Markdown",
        ReplyMarkup: &TelegramKeyboard{
            InlineKeyboard: [][]TelegramButton{
                {
                    {Text: "View Dashboard", URL: n.buildDashboardURL()},
                },
            },
        },
    }

    body, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("failed to marshal message: %w", err)
    }

    url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.cfg.TelegramConfig.BotToken)
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := n.client.Do(req)
    if err != nil {
        return fmt.Errorf("failed to send message: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
    }

    return nil
}

func (n *TelegramNotifier) buildDashboardURL() string {
    return fmt.Sprintf("http://localhost:%d/dashboard", n.cfg.Port)
}
```

### 7.4 Create Notification Manager

Create `/home/adam/projects/domain-minder/internal/services/notifications/manager.go`:

```go
package notifications

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"
    "github.com/yourusername/domain-minder/internal/config"
    "github.com/yourusername/domain-minder/internal/database"
    "github.com/yourusername/domain-minder/internal/models"
)

type NotificationManager struct {
    cfg           *config.Config
    emailNotifier *EmailNotifier
    tgNotifier    *TelegramNotifier
}

func NewNotificationManager(cfg *config.Config) *NotificationManager {
    return &NotificationManager{
        cfg:           cfg,
        emailNotifier: NewEmailNotifier(cfg),
        tgNotifier:    NewTelegramNotifier(cfg),
    }
}

func (m *NotificationManager) CheckDomain(ctx context.Context, user *models.User, domain *models.Domain) ([]*models.NotificationLog, error) {
    if domain.Status != "active" {
        return nil, nil
    }

    daysRemaining := int(time.Until(domain.ExpiryDate).Hours() / 24)

    // If domain already expired, skip notifications
    if daysRemaining < 0 {
        return nil, nil
    }

    // Parse user's notification thresholds from JSON
    thresholds, err := m.parseThresholds(user.NotificationThresholds)
    if err != nil {
        log.Printf("Failed to parse thresholds for user %d: %v", user.ID, err)
        thresholds = []int{90, 60, 30, 14, 7, 3, 1}
    }

    var logs []*models.NotificationLog

    for _, days := range thresholds {
        if daysRemaining == days {
            // Check if notification already sent
            exists, err := database.NotificationExists(ctx, domain.ID, "email", days)
            if err != nil {
                log.Printf("Failed to check notification exists: %v", err)
                continue
            }
            if exists {
                continue
            }

            // Send notification
            logEntry := m.sendNotification(ctx, user, domain, days)
            logs = append(logs, logEntry)
        }
    }

    // Daily notifications in final week
    if daysRemaining < 7 {
        // Check if notification sent today
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
        DomainID:           domain.ID,
        UserID:             user.ID,
        NotificationType:   "email",
        DaysBeforeExpiry:   daysRemaining,
        SentAt:             time.Now(),
        Success:            false,
    }

    var emailErr error

    // Send email if user has email notifications enabled and email is verified
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

    // Log the notification
    database.CreateNotificationLog(ctx, logEntry)

    // Send Telegram if enabled
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
```

### 7.5 Add Notification Log Field

Update `/home/adam/projects/domain-minder/internal/database/notifications.go` to include user_id in CreateNotificationLog:

```go
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
```

## Deliverables

- [ ] `/home/adam/projects/domain-minder/internal/services/notifications/notifier.go` with Notifier interface
- [ ] `/home/adam/projects/domain-minder/internal/services/notifications/email.go` with EmailNotifier implementation
- [ ] `/home/adam/projects/domain-minder/internal/services/notifications/telegram.go` with TelegramNotifier implementation
- [ ] `/home/adam/projects/domain-minder/internal/services/notifications/manager.go` with NotificationManager
- [ ] Manager loads thresholds from user's JSON column
- [ ] Email notifications blocked if email not verified
- [ ] All notifications logged to database
- [ ] Duplicate notifications prevented
- [ ] Daily notifications in final week

## Verification

```bash
cd /home/adam/projects/domain-minder

# Build
go build -o domain-minder ./cmd/server/

# Start server
./domain-minder &
sleep 2

# Register and login
curl -X POST http://localhost:9000/register \
  -d "email=test@example.com&password=test1234&confirm_password=test1234" \
  -c cookies.txt -b cookies.txt -L

# Add a domain expiring soon (manually set date for testing)
# First add domain
curl -X POST http://localhost:9000/domains \
  -d "name=test-domain.com" \
  -b cookies.txt -L

# Update domain to expire in 30 days
# (In real scenario, WHOIS would set this)

# Test notification check endpoint
curl -X POST http://localhost:9000/admin/check \
  -b cookies.txt

# Check notification logs
# (Would need a debug endpoint or direct database check)

# Cleanup
pkill -f domain-minder
rm -f domain_minder.db cookies.txt
```

## Next Phase

After completing verification, proceed to **Phase 8: Background Jobs**.
