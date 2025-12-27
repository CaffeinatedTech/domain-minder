# Domain Minder Implementation Plan

## Overview

This plan provides a staged implementation roadmap for Domain Minder, a domain expiry monitoring system. Each phase builds upon the previous, with clear deliverables and testing requirements.

---

## Phase 1: Project Setup

### 1.1 Initialize Go Module
- Create `go.mod` with module name `github.com/CaffeinatedTech/domain-minder`
- Set Go version to 1.21+
- Dependencies to add:
  - `github.com/labstack/echo/v4@v4.11.4`
  - `github.com/mattn/go-sqlite3@v1.14.22`
  - `github.com/labstack/echo/v4/middleware/session@v4.11.4`
  - `golang.org/x/crypto@v0.17.0` (for password hashing)

### 1.2 Directory Structure
```
domain-minder/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── database/
│   │   └── database.go
│   ├── models/
│   │   └── models.go
│   ├── handlers/
│   │   ├── auth.go
│   │   ├── domains.go
│   │   ├── dashboard.go
│   │   └── notifications.go
│   ├── services/
│   │   ├── whois.go
│   │   ├── checker.go
│   │   └── notifications/
│   │       ├── notifier.go
│   │       ├── email.go
│   │       └── telegram.go
│   ├── middleware/
│   │   └── auth.go
│   └── templates/
│       └── *.html
├── migrations/
├── data/
├── .env
├── .env.example
├── go.mod
├── go.sum
├── README.md
└── PLAN.md
```

### 1.3 Configuration System
- Create `internal/config/config.go`
- Load from environment variables with `.env` support
- Configuration struct with fields:
  - `DBPath`: string
  - `Port`: int
  - `SessionSecret`: string
  - `SMTPConfig`: struct (host, port, user, pass, from)
  - `TelegramConfig`: struct (bot token, chat ID)
  - `CheckInterval`: duration
  - `NotificationThresholds`: []int (days before expiry)

### 1.4 Basic Echo Server
- Create `cmd/server/main.go`
- Initialize Echo instance
- Add middleware: logger, recover, static files
- Create health check endpoint at `/health`
- Start server on configured port

**Deliverable**: Running Echo server responding to `/health`

---

## Phase 2: Database Schema & Models

### 2.1 Database Initialization
- Create `internal/database/database.go`
- Initialize SQLite connection
- Create database file if not exists
- Enable foreign keys

### 2.2 Schema Definition
Create tables:

**users table:**
```sql
CREATE TABLE users (
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
);
```

**domains table:**
```sql
CREATE TABLE domains (
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
);
```

### 2.3 Models
Create `internal/models/models.go`:
- `User` struct with email verification fields and notification_thresholds JSON string
- `Domain` struct
- `NotificationLog` struct

### 2.4 Database Helper Functions
- CRUD operations for each model
- Create `internal/database/users.go`, `domains.go`, `notifications.go`

**Deliverable**: All database tables created, models defined, CRUD functions implemented

---

## Phase 3: User Authentication

### 3.1 Session Management
- Configure Echo session middleware in `internal/middleware/auth.go`
- Use `github.com/labstack/echo/v4/middleware/session`
- Store sessions with secure cookies
- Session config:
  - `SessionName`: "domain-minder-session"
  - `SessionStore`: SQLite-backed store or cookie-based
  - `CookieHTTPOnly`: true
  - `CookieSecure`: true (in production)

### 3.2 Password Handling
- Use `golang.org/x/crypto/bcrypt` for password hashing
- Functions: `HashPassword`, `CheckPassword`

### 3.3 Auth Handlers
Create `internal/handlers/auth.go`:

**Endpoints:**
- `GET /register` - Registration form
- `POST /register` - Process registration
- `GET /login` - Login form
- `POST /login` - Process login
- `POST /logout` - Logout user
- `GET /settings` - User settings (notifications preferences)

**Registration flow:**
1. Validate email format
2. Check email not already registered
3. Hash password
4. Insert user into database
5. Create session
6. Redirect to dashboard

**Login flow:**
1. Find user by email
2. Verify password
3. Create session
4. Redirect to dashboard

### 3.4 Auth Middleware
- Create `requireAuth` middleware
- Protect dashboard, domain management, settings routes
- Redirect to login if not authenticated

### 3.5 User Settings
- Update notification preferences (email/telegram)
- Add/remove telegram chat ID
- Update password
- Configure custom notification thresholds (JSON array editor)

### 3.6 Email Verification Flow

**Database additions:**
- `email_verified BOOLEAN DEFAULT 0` (in users table)
- `email_verification_token TEXT` (in users table, unique, nullable)
- `email_verification_sent_at DATETIME` (for rate limiting)

**Verification workflow:**
1. **Generate token**: On registration, create cryptographically secure 32-byte token
2. **Send email**: Send verification email with link `https://domain.com/verify?token={token}`
3. **Verify endpoint**: `GET /verify?token={token}`
   - Find user by token
   - Mark `email_verified = 1`
   - Clear verification token
   - Show success message
4. **Resend option**: `POST /verify/resend` - sends new verification email
   - Rate limit: max 3 emails per hour
   - Generate new token each time

**Dashboard reminder:**
- When `email_verified = false` and `notification_email = true`:
  - Show prominent banner: "Email not verified - click to resend verification"
  - Show warning icon next to email in settings
  - Disable email notifications until verified
  - Banner dismissible (stored in session, not database)

**Email template:**
```html
<h2>Verify Your Email Address</h2>
<p>Click the link below to verify your email for Domain Minder:</p>
<p><a href="{{ .VerificationURL }}">Verify Email</a></p>
<p>This link expires in 24 hours.</p>
```

**Deliverable**: Working user registration, login, logout, and session management

---

## Phase 4: Domain Management (CRUD)

### 4.1 Domain Handlers
Create `internal/handlers/domains.go`:

**Endpoints:**
- `GET /domains` - List all user's domains
- `GET /domains/new` - Add domain form
- `POST /domains` - Add new domain
- `GET /domains/:id` - View domain details
- `GET /domains/:id/edit` - Edit domain form
- `POST /domains/:id` - Update domain
- `POST /domains/:id/delete` - Delete domain
- `POST /domains/:id/check` - Trigger immediate WHOIS check

### 4.2 Add Domain Flow
1. User enters domain name
2. Validate domain format (basic regex)
3. Perform initial WHOIS lookup
4. Extract expiry date and registrar
5. Save domain with extracted data
6. Show success with domain details

### 4.3 Edit Domain Flow
- Allow manual override of:
  - Registrar name
  - Expiry date
  - Notes
- Status options: active, expired, pending

### 4.4 Delete Domain
- Soft delete or hard delete (implement hard delete for simplicity)
- Cascade delete notification logs

### 4.5 Domain List View
- Table showing: domain name, registrar, expiry date, days remaining, status
- Sortable by any column
- Search/filter functionality

**Deliverable**: Full domain CRUD with WHOIS lookup on add

---

## Phase 5: WHOIS Service

### 5.1 WHOIS Client
Create `internal/services/whois.go`:

```go
type WHOISResult struct {
    DomainName    string
    Registrar     string
    ExpiryDate    time.Time
    WHOISRaw      string
    Error         error
}
```

- Use standard net package for WHOIS queries
- Query default WHOIS server or use whois.iana.org to find appropriate server
- Handle various WHOIS response formats
- Parse expiry date from response (multiple format support)

### 5.2 Expiry Date Parsing
Common formats to handle:
- `Expiry Date: 2025-12-31`
- `Expiration Date: 31-Dec-2025`
- `expires: 2025-12-31T23:59:59Z`
- `Domain Expiration Date: 12/31/2025`

### 5.3 WHOIS Cache
- Cache WHOIS results to avoid rate limiting
- Cache duration: 24 hours
- Allow manual refresh

### 5.4 Error Handling
- Domain not found
- WHOIS server timeout
- Invalid response format
- Rate limiting

**Deliverable**: Reliable WHOIS lookup with parsing and error handling

---

## Phase 6: Dashboard & UI

### 6.1 Base Template
Create `internal/templates/base.html`:
- HTML5 boilerplate
- HTMX CDN import
- CSS framework (use simple CSS or Tailwind via CDN)
- Header with user info and navigation
- Flash message support

### 6.2 Dashboard View
Create `internal/templates/dashboard.html`:

**Features:**
- Summary stats: total domains, expiring soon, expired
- Progress bars for each domain showing days remaining
- Visual styling:
  - Green: > 60 days
  - Yellow: 30-60 days
  - Orange: 14-30 days
  - Red: < 14 days
  - Critical: < 7 days (pulsing animation)

**Progress bar implementation:**
```html
<div class="progress-bar" data-days="{{ .DaysRemaining }}">
  <div class="fill" style="width: {{ .Percentage }}%; background: {{ .Color }}"></div>
  <span class="label">{{ .DaysRemaining }} days</span>
</div>
```

### 6.3 HTMX Integration
- Use HTMX for dynamic interactions:
  - `hx-delete` for domain deletion
  - `hx-post` for adding domains
  - `hx-trigger` for auto-refresh of dashboard
  - `hx-swap` for smooth updates

### 6.4 Responsive Design
- Mobile-friendly layout
- Collapsible sidebar for mobile
- Touch-friendly targets

### 6.5 Visual Polish
- Clean color scheme
- Smooth animations
- Loading states (HTMX `hx-indicator`)
- Toast notifications for actions

**Deliverable**: Beautiful, responsive dashboard with progress bars and HTMX interactions

---

## Phase 7: Notification System (Modular)

### 7.1 Notification Interface
Create `internal/services/notifications/notifier.go`:

```go
type Notifier interface {
    Send(ctx context.Context, notification *Notification) error
    Name() string
}

type Notification struct {
    UserID       int
    DomainName   string
    Registrar    string
    ExpiryDate   time.Time
    DaysRemaining int
    Channel      string
}
```

### 7.2 Email Notifier
Create `internal/services/notifications/email.go`:

- Use `net/smtp` package
- HTML email template with:
  - Domain name
  - Registrar
  - Days remaining
  - Warning level
  - Link to dashboard
- Plain text fallback

**Email template:**
```html
<h2>Domain Expiry Warning: {{ .DomainName }}</h2>
<p>Your domain <strong>{{ .DomainName }}</strong> registered with <strong>{{ .Registrar }}</strong> 
will expire in <strong>{{ .DaysRemaining }} days</strong> on {{ .ExpiryDate.Format "January 2, 2006" }}.</p>
```

### 7.3 Telegram Notifier
Create `internal/services/notifications/telegram.go`:

- Use Telegram Bot API
- Message format similar to email
- Markdown formatting for bold/italics
- Inline keyboard to open dashboard

### 7.4 Notification Manager
Create `internal/services/notifications/manager.go`:

- Load thresholds from user's `notification_thresholds` JSON column
- Parse JSON array of integers
- Check domains against user's thresholds
- Skip if `email_verified = false` for email notifications
- Log all notifications (success/failure)
- Prevent duplicate notifications
- Support user preferences (email on/off, telegram on/off)

### 7.5 Default Thresholds
- Default JSON stored in users table: `[90, 60, 30, 14, 7, 3, 1]`
- Users can customize thresholds in settings (JSON array input)
- In final week (days < 7): notify daily regardless of custom thresholds
- Parse JSON on startup, cache per user session

**Deliverable**: Modular notification system with email and telegram working

---

## Phase 8: Background Jobs & Scheduler

### 8.1 Scheduler Implementation
Create `internal/services/checker.go`:

- Use `github.com/robfig/cron/v3` for scheduling
- Default schedule: check every 6 hours
- Configurable via environment variable

### 8.2 Domain Checking Job
1. Query all active domains
2. For each domain:
   - Perform WHOIS lookup
   - Update expiry date if changed
   - Check against notification thresholds
   - Trigger notifications if needed

### 8.3 Notification Check Job
1. Query notification thresholds
2. For each threshold:
   - Find domains expiring in that timeframe
   - Skip if notification already sent
   - Send notification
   - Log result

### 8.4 Job Persistence
- Jobs run on server startup
- Handle graceful shutdown
- Log job execution times and results

### 8.5 Manual Trigger
- Endpoint to trigger immediate check: `POST /admin/check`
- Protected by authentication
- Returns job status

**Deliverable**: Background scheduler with domain checking and notifications

---

## Phase 9: Testing & Quality Assurance

### 9.1 Unit Tests
Create test files for each package:
- `internal/config/config_test.go`
- `internal/database/*_test.go`
- `internal/services/whois_test.go` (mock WHOIS server)
- `internal/services/notifications/*_test.go` (mock email/telegram)

### 9.2 Handler Tests
Create `internal/handlers/*_test.go`:
- Test all endpoints
- Use `httptest` package
- Mock database layer

### 9.3 Integration Tests
- Full user flow: register → login → add domain → check dashboard
- Notification flow: trigger notification check
- Background job execution

### 9.4 Test Coverage
- Aim for 80%+ coverage on core functionality
- Critical paths: 100% coverage

### 9.5 Linting
- Run `golangci-lint` or `gofmt`
- Fix all warnings
- Configure CI to fail on lint errors

**Deliverable**: Comprehensive test suite with good coverage

---

## Phase 10: Documentation & Release

### 10.1 README Improvements
- Add screenshots
- Installation instructions
- Configuration examples
- API documentation if applicable

### 10.2 Contribution Guide
- How to submit issues
- How to create pull requests
- Coding standards
- Testing requirements

### 10.3 License
- Add MIT LICENSE file
- Update module path in go.mod

### 10.4 GitHub Setup
- Create GitHub repository
- Push code
- Set up branches (main, develop)
- Configure branch protection

### 10.5 CI/CD (Optional)
- GitHub Actions workflow:
  - Run tests on push
  - Run linting
  - Build binary
  - Publish release

### 10.6 Deployment Guide
- Docker setup (Dockerfile)
- Docker Compose for development
- Production deployment notes
- Environment configuration

**Deliverable**: Production-ready project with complete documentation

---

## Implementation Order Summary

1. **Phase 1**: Project Setup - Foundation
2. **Phase 2**: Database & Models - Data layer
3. **Phase 3**: User Auth - Security layer
4. **Phase 4**: Domain CRUD - Core feature
5. **Phase 5**: WHOIS Service - External integration
6. **Phase 6**: Dashboard & UI - User experience
7. **Phase 7**: Notification System - Alerting
8. **Phase 8**: Background Jobs - Automation
9. **Phase 9**: Testing - Quality assurance
10. **Phase 10**: Documentation - Polish & release

---

## Notes for AI Agent

- Implement one phase at a time
- Run tests after each phase
- Commit changes after completing each phase
- Use environment variables for all configuration
- Follow Go conventions: `camelCase` for exports, `PascalCase` for acronyms
- Keep handlers thin, push logic to services
- Use context for cancellation and timeouts
- Log all errors with context
- Handle edge cases gracefully
- Add comments for complex logic
- Write tests before or alongside implementation
