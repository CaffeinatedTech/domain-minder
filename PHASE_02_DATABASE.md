# Phase 2: Database Schema & Models

## What Should Already Exist

- `/home/adam/projects/domain-minder/go.mod` with dependencies
- `/home/adam/projects/domain-minder/internal/config/config.go` with Config struct
- `/home/adam/projects/domain-minder/cmd/server/main.go` basic Echo server
- `/home/adam/projects/domain-minder/data/` directory exists
- `/home/adam/projects/domain-minder/.env.example` configuration template

## Context

This phase establishes the data layer. You will create the database initialization, schema with users, domains, and notification_logs tables, Go models for each table, and CRUD helper functions. The users table includes email verification fields and a JSON string for notification thresholds.

## Agent Rules

1. Create files ONLY in `/home/adam/projects/domain-minder/internal/database/` and `/home/adam/projects/domain-minder/internal/models/`
2. Follow the exact schema provided - do not add or modify columns
3. Models must match SQL schema exactly with matching field names
4. CRUD functions must return proper Go types and errors
5. Use context for all database operations
6. Database file path comes from config.DBPath
7. Do NOT create any HTTP handlers yet - only database code
8. Do NOT create any services yet - only models and database layer

## Tasks

### 2.1 Create Database Initialization

Create `/home/adam/projects/domain-minder/internal/database/database.go`:

```go
package database

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    _ "github.com/mattn/go-sqlite3"

    "github.com/yourusername/domain-minder/internal/config"
)

var DB *sql.DB

func Init(cfg *config.Config) error {
    var err error
    DB, err = sql.Open("sqlite3", cfg.DBPath)
    if err != nil {
        return fmt.Errorf("failed to open database: %w", err)
    }

    DB.SetMaxOpenConns(25)
    DB.SetMaxIdleConns(5)
    DB.SetConnMaxLifetime(5 * time.Minute)

    if err := DB.Ping(); err != nil {
        return fmt.Errorf("failed to ping database: %w", err)
    }

    if err := migrate(); err != nil {
        return fmt.Errorf("failed to migrate database: %w", err)
    }

    return nil
}

func Close() error {
    if DB != nil {
        return DB.Close()
    }
    return nil
}
```

### 2.2 Create Migration Function

Add to `/home/adam/projects/domain-minder/internal/database/database.go`:

```go
func migrate() error {
    queries := []string{
        `CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            email TEXT UNIQUE NOT NULL,
            password_hash TEXT NOT NULL,
            email_verified BOOLEAN DEFAULT 0,
            email_verification_token TEXT,
            telegram_chat_id TEXT,
            notification_email BOOLEAN DEFAULT 1,
            notification_telegram BOOLEAN DEFAULT 0,
            notification_thresholds TEXT DEFAULT '[90, 60, 30, 14, 7, 3, 1]',
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );`,
        `CREATE TABLE IF NOT EXISTS domains (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            name TEXT NOT NULL,
            registrar TEXT,
            expiry_date DATETIME NOT NULL,
            whois_raw TEXT,
            last_checked DATETIME,
            status TEXT DEFAULT 'active',
            notes TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
        );`,
        `CREATE TABLE IF NOT EXISTS notification_logs (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            domain_id INTEGER NOT NULL,
            user_id INTEGER NOT NULL,
            notification_type TEXT NOT NULL,
            days_before_expiry INTEGER NOT NULL,
            sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            success BOOLEAN DEFAULT 0,
            error_message TEXT,
            FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE,
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
        );`,
        `CREATE INDEX IF NOT EXISTS idx_domains_user_id ON domains(user_id);`,
        `CREATE INDEX IF NOT EXISTS idx_notification_logs_domain_id ON notification_logs(domain_id);`,
        `CREATE INDEX IF NOT EXISTS idx_notification_logs_user_id ON notification_logs(user_id);`,
    }

    for _, query := range queries {
        if _, err := DB.Exec(query); err != nil {
            return err
        }
    }

    return nil
}
```

### 2.3 Create Models

Create `/home/adam/projects/domain-minder/internal/models/models.go`:

```go
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
    ID          int       `json:"id"`
    UserID      int       `json:"user_id"`
    Name        string    `json:"name"`
    Registrar   *string   `json:"registrar,omitempty"`
    ExpiryDate  time.Time `json:"expiry_date"`
    WHOISRaw    *string   `json:"whois_raw,omitempty"`
    LastChecked *time.Time `json:"last_checked,omitempty"`
    Status      string    `json:"status"`
    Notes       *string   `json:"notes,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type NotificationLog struct {
    ID                 int       `json:"id"`
    DomainID           int       `json:"domain_id"`
    UserID             int       `json:"user_id"`
    NotificationType   string    `json:"notification_type"`
    DaysBeforeExpiry   int       `json:"days_before_expiry"`
    SentAt             time.Time `json:"sent_at"`
    Success            bool      `json:"success"`
    ErrorMessage       *string   `json:"error_message,omitempty"`
}
```

### 2.4 Create User Database Functions

Create `/home/adam/projects/domain-minder/internal/database/users.go`:

```go
package database

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/yourusername/domain-minder/internal/models"
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
```

### 2.5 Create Domain Database Functions

Create `/home/adam/projects/domain-minder/internal/database/domains.go`:

```go
package database

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/yourusername/domain-minder/internal/models"
)

func CreateDomain(ctx context.Context, domain *models.Domain) (int64, error) {
    result, err := DB.ExecContext(ctx, `
        INSERT INTO domains (user_id, name, registrar, expiry_date, whois_raw, status, notes)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `, domain.UserID, domain.Name, domain.Registrar, domain.ExpiryDate, domain.WHOISRaw,
        domain.Status, domain.Notes)
    if err != nil {
        return 0, fmt.Errorf("failed to create domain: %w", err)
    }
    return result.LastInsertId()
}

func GetDomainByID(ctx context.Context, id int) (*models.Domain, error) {
    domain := &models.Domain{}
    err := DB.QueryRowContext(ctx, `
        SELECT id, user_id, name, registrar, expiry_date, whois_raw, last_checked,
            status, notes, created_at, updated_at
        FROM domains WHERE id = ?
    `, id).Scan(&domain.ID, &domain.UserID, &domain.Name, &domain.Registrar,
        &domain.ExpiryDate, &domain.WHOISRaw, &domain.LastChecked, &domain.Status,
        &domain.Notes, &domain.CreatedAt, &domain.UpdatedAt)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get domain: %w", err)
    }
    return domain, nil
}

func GetDomainsByUserID(ctx context.Context, userID int) ([]*models.Domain, error) {
    rows, err := DB.QueryContext(ctx, `
        SELECT id, user_id, name, registrar, expiry_date, whois_raw, last_checked,
            status, notes, created_at, updated_at
        FROM domains WHERE user_id = ? ORDER BY expiry_date ASC
    `, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get domains: %w", err)
    }
    defer rows.Close()

    var domains []*models.Domain
    for rows.Next() {
        domain := &models.Domain{}
        if err := rows.Scan(&domain.ID, &domain.UserID, &domain.Name, &domain.Registrar,
            &domain.ExpiryDate, &domain.WHOISRaw, &domain.LastChecked, &domain.Status,
            &domain.Notes, &domain.CreatedAt, &domain.UpdatedAt); err != nil {
            return nil, err
        }
        domains = append(domains, domain)
    }
    return domains, rows.Err()
}

func GetAllActiveDomains(ctx context.Context) ([]*models.Domain, error) {
    rows, err := DB.QueryContext(ctx, `
        SELECT id, user_id, name, registrar, expiry_date, whois_raw, last_checked,
            status, notes, created_at, updated_at
        FROM domains WHERE status = 'active'
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to get all domains: %w", err)
    }
    defer rows.Close()

    var domains []*models.Domain
    for rows.Next() {
        domain := &models.Domain{}
        if err := rows.Scan(&domain.ID, &domain.UserID, &domain.Name, &domain.Registrar,
            &domain.ExpiryDate, &domain.WHOISRaw, &domain.LastChecked, &domain.Status,
            &domain.Notes, &domain.CreatedAt, &domain.UpdatedAt); err != nil {
            return nil, err
        }
        domains = append(domains, domain)
    }
    return domains, rows.Err()
}

func UpdateDomain(ctx context.Context, domain *models.Domain) error {
    _, err := DB.ExecContext(ctx, `
        UPDATE domains SET name = ?, registrar = ?, expiry_date = ?, whois_raw = ?,
            last_checked = ?, status = ?, notes = ?, updated_at = ?
        WHERE id = ?
    `, domain.Name, domain.Registrar, domain.ExpiryDate, domain.WHOISRaw,
        domain.LastChecked, domain.Status, domain.Notes, time.Now(), domain.ID)
    return err
}

func UpdateDomainWHOIS(ctx context.Context, id int, expiryDate time.Time, registrar, whoisRaw string) error {
    _, err := DB.ExecContext(ctx, `
        UPDATE domains SET expiry_date = ?, registrar = ?, whois_raw = ?,
            last_checked = ?, updated_at = ?
        WHERE id = ?
    `, expiryDate, registrar, whoisRaw, time.Now(), time.Now(), id)
    return err
}

func DeleteDomain(ctx context.Context, id int) error {
    _, err := DB.ExecContext(ctx, "DELETE FROM domains WHERE id = ?", id)
    return err
}
```

### 2.6 Create Notification Log Database Functions

Create `/home/adam/projects/domain-minder/internal/database/notifications.go`:

```go
package database

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/yourusername/domain-minder/internal/models"
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
```

## Deliverables

- [ ] `/home/adam/projects/domain-minder/internal/database/database.go` with Init, Close, and migrate functions
- [ ] `/home/adam/projects/domain-minder/internal/models/models.go` with User, Domain, NotificationLog structs
- [ ] `/home/adam/projects/domain-minder/internal/database/users.go` with all user CRUD functions
- [ ] `/home/adam/projects/domain-minder/internal/database/domains.go` with all domain CRUD functions
- [ ] `/home/adam/projects/domain-minder/internal/database/notifications.go` with notification log functions
- [ ] All tables created with proper indexes
- [ ] Database file created at path specified in config

## Verification

```bash
cd /home/adam/projects/domain-minder

# Create .env file for testing
cat > .env <<EOF
DB_PATH=/tmp/domain_minder_test.db
PORT=9000
SESSION_SECRET=test-secret
EOF

# Run tests
go test ./internal/database/... -v
go test ./internal/models/... -v

# Run all tests (should pass)
go test ./...

# Verify database file created
ls -la /tmp/domain_minder_test.db
```

## Integration with Main

Update `/home/adam/projects/domain-minder/cmd/server/main.go` to initialize the database:

```go
func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Ensure data directory exists
    if err := os.MkdirAll("data", 0755); err != nil {
        log.Fatalf("Failed to create data directory: %v", err)
    }

    // Initialize database
    if err := database.Init(cfg); err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }
    defer database.Close()

    e := echo.New()
    // ... rest of setup
}
```

## Next Phase

After completing verification, proceed to **Phase 3: Authentication**.
