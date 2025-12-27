# Phase 6: Dashboard & UI

## What Should Already Exist

- `/home/adam/projects/domain-minder/internal/handlers/domains.go` with domain CRUD handlers
- `/home/adam/projects/domain-minder/internal/handlers/auth.go` with auth handlers
- `/home/adam/projects/domain-minder/internal/handlers/settings.go` with settings handlers
- `/home/adam/projects/domain-minder/internal/models/models.go` with Domain struct

## Context

This phase implements the UI templates using HTML with HTMX for dynamic interactions. The dashboard displays domains with color-coded progress bars showing days remaining. Color scheme: green (>60 days), yellow (30-60), orange (14-30), red (<14), pulsing red (<7).

## Agent Rules

1. Create files ONLY in `/home/adam/projects/domain-minder/internal/templates/`
2. Use HTMX CDN for dynamic interactions (hx-delete, hx-post, hx-target, hx-swap)
3. Templates must be valid HTML5
4. Progress bars must show days remaining with appropriate colors
5. Dashboard must show email verification banner when applicable
6. Include loading states (hx-indicator) for HTMX operations
7. Responsive design for mobile devices
8. Do NOT add CSS frameworks - write custom CSS
9. Do NOT create new handlers - only create templates
10. Update existing handlers to use templates

## Tasks

### 6.1 Create Base Template

Create `/home/adam/projects/domain-minder/internal/templates/layout.html`:

```go
package templates

const Layout = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Domain Minder</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: #f5f5f5;
            color: #333;
            line-height: 1.6;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        header {
            background: #2c3e50;
            color: white;
            padding: 15px 0;
            margin-bottom: 30px;
        }
        header .container {
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        header h1 {
            font-size: 1.5rem;
        }
        header a {
            color: white;
            text-decoration: none;
            margin-left: 20px;
        }
        header a:hover {
            text-decoration: underline;
        }
        .card {
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            padding: 20px;
            margin-bottom: 20px;
        }
        .card h2 {
            margin-bottom: 15px;
            color: #2c3e50;
        }
        .btn {
            display: inline-block;
            padding: 8px 16px;
            background: #3498db;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            text-decoration: none;
            font-size: 14px;
        }
        .btn:hover {
            background: #2980b9;
        }
        .btn-danger {
            background: #e74c3c;
        }
        .btn-danger:hover {
            background: #c0392b;
        }
        .btn-small {
            padding: 4px 8px;
            font-size: 12px;
        }
        form {
            margin: 0;
        }
        input[type="text"],
        input[type="email"],
        input[type="password"],
        input[type="date"],
        select,
        textarea {
            width: 100%;
            padding: 8px 12px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 14px;
            margin-bottom: 10px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: 500;
        }
        .alert {
            padding: 15px;
            border-radius: 4px;
            margin-bottom: 20px;
        }
        .alert-warning {
            background: #fff3cd;
            border: 1px solid #ffc107;
            color: #856404;
        }
        .alert-success {
            background: #d4edda;
            border: 1px solid #28a745;
            color: #155724;
        }
        .alert-error {
            background: #f8d7da;
            border: 1px solid #dc3545;
            color: #721c24;
        }
        .banner {
            padding: 15px;
            border-radius: 4px;
            margin-bottom: 20px;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .banner-warning {
            background: #fff3cd;
            border: 1px solid #ffc107;
        }
        .banner-critical {
            background: #f8d7da;
            border: 1px solid #dc3545;
            animation: pulse 2s infinite;
        }
        @keyframes pulse {
            0% { opacity: 1; }
            50% { opacity: 0.8; }
            100% { opacity: 1; }
        }
        .progress-container {
            background: #e9ecef;
            border-radius: 10px;
            height: 24px;
            position: relative;
            overflow: hidden;
        }
        .progress-bar {
            height: 100%;
            border-radius: 10px;
            transition: width 0.3s ease;
            display: flex;
            align-items: center;
            justify-content: flex-end;
            padding-right: 10px;
            color: white;
            font-size: 12px;
            font-weight: bold;
        }
        .progress-green { background: #28a745; }
        .progress-yellow { background: #ffc107; color: #333; }
        .progress-orange { background: #fd7e14; }
        .progress-red { background: #dc3545; }
        .progress-critical {
            background: #dc3545;
            animation: pulse 1s infinite;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #eee;
        }
        th {
            background: #f8f9fa;
            font-weight: 600;
        }
        .htmx-indicator {
            opacity: 0;
            transition: opacity 0.3s;
        }
        .htmx-request .htmx-indicator {
            opacity: 1;
        }
        .htmx-request.htmx-indicator {
            opacity: 1;
        }
        .loading {
            text-align: center;
            padding: 20px;
            color: #666;
        }
        .domain-row {
            transition: background 0.2s;
        }
        .domain-row:hover {
            background: #f8f9fa;
        }
        .actions {
            display: flex;
            gap: 5px;
        }
        @media (max-width: 768px) {
            .container {
                padding: 10px;
            }
            table {
                display: block;
                overflow-x: auto;
            }
        }
    </style>
</head>
<body>
    <header>
        <div class="container">
            <h1>Domain Minder</h1>
            <nav>
                {{ if .IsAuthenticated }}
                    <a href="/dashboard">Dashboard</a>
                    <a href="/domains">Domains</a>
                    <a href="/settings">Settings</a>
                    <form method="POST" action="/logout" style="display:inline;">
                        <button type="submit" style="background:none;border:none;color:white;cursor:pointer;font-size:14px;margin-left:20px;">Logout</button>
                    </form>
                {{ else }}
                    <a href="/login">Login</a>
                    <a href="/register">Register</a>
                {{ end }}
            </nav>
        </div>
    </header>
    <main class="container">
        {{ .Content }}
    </main>
</body>
</html>
`
```

### 6.2 Create Dashboard Template

Create `/home/adam/projects/domain-minder/internal/templates/dashboard.html`:

```go
package templates

const Dashboard = `
{{ if .ShowEmailWarning }}
<div class="banner banner-warning">
    <span><strong>Warning:</strong> Your email is not verified. Email notifications are disabled until you verify.</span>
    <button class="btn btn-small" hx-post="/verify/resend" hx-target="this" hx-swap="outerHTML">
        Resend Verification Email
    </button>
</div>
{{ end }}

<div class="card">
    <h2>Summary</h2>
    <p><strong>Total Domains:</strong> {{ .TotalDomains }}</p>
    <p><strong>Expiring Soon (30 days):</strong> {{ .ExpiringSoon }}</p>
    <p><strong>Expired:</strong> {{ .Expired }}</p>
</div>

<div class="card">
    <h2>Your Domains</h2>
    {{ if eq .TotalDomains 0 }}
        <p>No domains added yet. <a href="/domains">Add your first domain</a></p>
    {{ else }}
        <table>
            <thead>
                <tr>
                    <th>Domain</th>
                    <th>Registrar</th>
                    <th>Days Left</th>
                    <th>Status</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody hx-get="/domains/list" hx-trigger="load, domainUpdated from:body">
                <tr>
                    <td colspan="5" class="loading">Loading domains...</td>
                </tr>
            </tbody>
        </table>
    {{ end }}
</div>

<div class="card">
    <h2>Quick Actions</h2>
    <a href="/domains/new" class="btn">Add New Domain</a>
</div>
`

func DashboardData struct {
    TotalDomains  int
    ExpiringSoon  int
    Expired       int
    ShowEmailWarning bool
    IsAuthenticated bool
}
```

### 6.3 Create Domain List Template

Create `/home/adam/projects/domain-minder/internal/templates/domain_list.html`:

```go
package templates

const DomainList = `
{{ range .Domains }}
<tr class="domain-row">
    <td>
        <a href="/domains/{{ .ID }}">{{ .Name }}</a>
    </td>
    <td>{{ .Registrar }}</td>
    <td>
        <div class="progress-container">
            {{ $color := .Color }}
            {{ $percent := .Percent }}
            {{ $daysLeft := .DaysLeft }}
            <div class="progress-bar {{ $color }}" style="width: {{ $percent }}%;">
                {{ $daysLeft }}d
            </div>
        </div>
    </td>
    <td>
        <span class="status-badge status-{{ .StatusClass }}">{{ .Status }}</span>
    </td>
    <td class="actions">
        <button class="btn btn-small" hx-post="/domains/{{ .ID }}/check" hx-target="closest tr" hx-swap="outerHTML">
            Check
        </button>
        <form method="POST" action="/domains/{{ .ID }}/delete" hx-target="closest tr" hx-swap="outerHTML" style="display:inline;">
            <button type="submit" class="btn btn-small btn-danger" onclick="return confirm('Delete this domain?')">Delete</button>
        </form>
    </td>
</tr>
{{ else }}
<tr>
    <td colspan="5" style="text-align:center;">No domains found</td>
</tr>
{{ end }}
`
```

### 6.4 Create Domain Form Templates

Create `/home/adam/projects/domain-minder/internal/templates/forms.html`:

```go
package templates

const AddDomain = `
<div class="card">
    <h2>Add New Domain</h2>
    <form method="POST" action="/domains" hx-post="/domains" hx-target="#result" hx-swap="innerHTML">
        <label>Domain Name</label>
        <input type="text" name="name" placeholder="example.com" required
               hx-get="/domains/check-name" hx-trigger="blur" hx-target="#name-feedback">
        <div id="name-feedback"></div>
        <small>WHOIS lookup will be performed to find expiry date and registrar.</small>
        <br><br>
        <button type="submit" class="btn">Add Domain</button>
        <a href="/domains" class="btn" style="background:#6c757d;">Cancel</a>
    </form>
    <div id="result"></div>
</div>
`

const EditDomain = `
<div class="card">
    <h2>Edit Domain</h2>
    <form method="POST" action="/domains/{{ .Domain.ID }}">
        <label>Domain Name</label>
        <input type="text" name="name" value="{{ .Domain.Name }}" required>
        
        <label>Registrar</label>
        <input type="text" name="registrar" value="{{ .Domain.Registrar }}">
        
        <label>Expiry Date</label>
        <input type="date" name="expiry_date" value="{{ .Domain.ExpiryDate.Format "2006-01-02" }}" required>
        
        <label>Status</label>
        <select name="status">
            <option value="active"{{ if eq .Domain.Status "active" }} selected{{ end }}>Active</option>
            <option value="expired"{{ if eq .Domain.Status "expired" }} selected{{ end }}>Expired</option>
            <option value="pending"{{ if eq .Domain.Status "pending" }} selected{{ end }}>Pending</option>
        </select>
        
        <label>Notes</label>
        <textarea name="notes" rows="3">{{ .Domain.Notes }}</textarea>
        
        <button type="submit" class="btn">Save Changes</button>
        <a href="/domains" class="btn" style="background:#6c757d;">Cancel</a>
    </form>
</div>
`
```

### 6.5 Create Settings Template

Create `/home/adam/projects/domain-minder/internal/templates/settings.html`:

```go
package templates

const Settings = `
{{ if .ShowEmailWarning }}
<div class="banner banner-warning">
    <span><strong>Email Not Verified:</strong> Click to resend verification email.</span>
    <button class="btn btn-small" hx-post="/verify/resend" hx-target="this" hx-swap="outerHTML">
        Resend Verification Email
    </button>
</div>
{{ end }}

<div class="card">
    <h2>Profile</h2>
    <p><strong>Email:</strong> {{ .User.Email }}
    {{ if .User.EmailVerified }}
        <span style="color:green;">(Verified)</span>
    {{ else }}
        <span style="color:red;">(Not Verified)</span>
    {{ end }}
    </p>
</div>

<div class="card">
    <h2>Notification Preferences</h2>
    <form method="POST" action="/settings/notifications" hx-post="/settings/notifications" hx-target="#settings-result" hx-swap="innerHTML">
        <label>
            <input type="checkbox" name="notification_email"{{ if .User.NotificationEmail }} checked{{ end }}>
            Email notifications
        </label>
        <br>
        <label>
            <input type="checkbox" name="notification_telegram"{{ if .User.NotificationTelegram }} checked{{ end }}>
            Telegram notifications
        </label>
        <br>
        <label>Telegram Chat ID</label>
        <input type="text" name="telegram_chat_id" value="{{ .TelegramChatID }}" placeholder="Your Telegram Chat ID">
        <small>Required for Telegram notifications</small>
        <br><br>
        <button type="submit" class="btn">Save Preferences</button>
    </form>
</div>

<div class="card">
    <h2>Notification Thresholds</h2>
    <form method="POST" action="/settings/thresholds" hx-post="/settings/thresholds" hx-target="#settings-result" hx-swap="innerHTML">
        <label>Days before expiry (comma-separated)</label>
        <input type="text" name="thresholds" value="{{ .User.NotificationThresholds }}" placeholder="90, 60, 30, 14, 7, 3, 1">
        <small>Example: 90, 60, 30, 14, 7, 3, 1</small>
        <br><br>
        <button type="submit" class="btn">Save Thresholds</button>
    </form>
</div>

<div id="settings-result"></div>
`
```

### 6.6 Create Auth Templates

Create `/home/adam/projects/domain-minder/internal/templates/auth.html`:

```go
package templates

const Register = `
<div class="card" style="max-width:400px;margin:0 auto;">
    <h2>Create Account</h2>
    <form method="POST" action="/register">
        <label>Email</label>
        <input type="email" name="email" required>
        
        <label>Password</label>
        <input type="password" name="password" required minlength="8">
        <small>Minimum 8 characters</small>
        
        <label>Confirm Password</label>
        <input type="password" name="confirm_password" required>
        
        <button type="submit" class="btn" style="width:100%;margin-top:10px;">Register</button>
    </form>
    <p style="text-align:center;margin-top:15px;">
        Already have an account? <a href="/login">Login</a>
    </p>
</div>
`

const Login = `
<div class="card" style="max-width:400px;margin:0 auto;">
    <h2>Login</h2>
    <form method="POST" action="/login">
        <label>Email</label>
        <input type="email" name="email" required>
        
        <label>Password</label>
        <input type="password" name="password" required>
        
        <button type="submit" class="btn" style="width:100%;margin-top:10px;">Login</button>
    </form>
    <p style="text-align:center;margin-top:15px;">
        Don't have an account? <a href="/register">Register</a>
    </p>
</div>
`

const VerifyEmail = `
<div class="card" style="max-width:400px;margin:0 auto;text-align:center;">
    <h2>Email Verification</h2>
    <p>Thank you for registering! Please check your email for a verification link.</p>
    <p>If you didn't receive the email, check your spam folder or request a new one.</p>
    <button class="btn" hx-post="/verify/resend" hx-target="this" hx-swap="outerHTML">
        Resend Verification Email
    </button>
</div>
`

const Verified = `
<div class="card" style="max-width:400px;margin:0 auto;text-align:center;">
    <h2 style="color:green;">Email Verified!</h2>
    <p>Your email has been successfully verified.</p>
    <a href="/dashboard" class="btn">Go to Dashboard</a>
</div>
`
```

### 6.7 Create Template Helper Functions

Create `/home/adam/projects/domain-minder/internal/templates/helpers.go`:

```go
package templates

import (
    "html/template"
    "strings"
    "time"
)

var Templates *template.Template

func init() {
    tmpl := template.New("templates")
    
    funcs := template.FuncMap{
        "formatDate": formatDate,
        "daysLeft":   daysLeft,
        "colorClass": colorClass,
        "percent":    percentRemaining,
        "eq":         func(a, b string) bool { return a == b },
    }
    
    tmpl = tmpl.Funcs(funcs)
    
    const baseTemplate = Layout
    tmpl = template.Must(tmpl.Parse(baseTemplate))
    
    Templates = tmpl
}

func formatDate(t time.Time) string {
    return t.Format("January 2, 2006")
}

func daysLeft(expiry time.Time) int {
    return int(time.Until(expiry).Hours() / 24)
}

func colorClass(daysLeft int) string {
    switch {
    case daysLeft < 7:
        return "progress-critical"
    case daysLeft < 14:
        return "progress-red"
    case daysLeft < 30:
        return "progress-orange"
    case daysLeft < 60:
        return "progress-yellow"
    default:
        return "progress-green"
    }
}

func percentRemaining(expiry time.Time, maxDays int) float64 {
    days := daysLeft(expiry)
    if days <= 0 {
        return 100
    }
    if days > maxDays {
        return 5 // Minimum width to show something
    }
    return float64(days) / float64(maxDays) * 100
}

func JoinStrings(strs []string, sep string) string {
    return strings.Join(strs, sep)
}
```

### 6.8 Update Handlers to Use Templates

Update `/home/adam/projects/domain-minder/internal/handlers/dashboard.go`:

```go
package handlers

import (
    "net/http"
    "html/template"
    "github.com/yourusername/domain-minder/internal/middleware"
    "github.com/yourusername/domain-minder/internal/templates"
    "github.com/labstack/echo/v4"
)

type DashboardHandler struct{}

func NewDashboardHandler() *DashboardHandler {
    return &DashboardHandler{}
}

func (h *DashboardHandler) Show(c echo.Context) error {
    user := middleware.GetCurrentUser(c)
    if user == nil {
        return c.Redirect(http.StatusSeeOther, "/login")
    }

    var buf strings.Builder
    data := struct {
        User            *models.User
        ShowEmailWarning bool
    }{
        User:             user,
        ShowEmailWarning: !user.EmailVerified && user.NotificationEmail,
    }
    
    if err := templates.Templates.ExecuteTemplate(&buf, "base.html", data); err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }
    
    c.Response().Header().Set("Content-Type", "text/html")
    return c.String(http.StatusOK, buf.String())
}
```

Note: The template execution needs to be refactored to work with the Layout pattern. Create a Render helper function.

## Deliverables

- [ ] `/home/adam/projects/domain-minder/internal/templates/layout.html` with base HTML, CSS, and header/footer
- [ ] `/home/adam/projects/domain-minder/internal/templates/dashboard.html` with summary stats and domain list
- [ ] `/home/adam/projects/domain-minder/internal/templates/domain_list.html` with progress bars and colors
- [ ] `/home/adam/projects/domain-minder/internal/templates/forms.html` with add/edit domain forms
- [ ] `/home/adam/projects/domain-minder/internal/templates/settings.html` with notification preferences
- [ ] `/home/adam/projects/domain-minder/internal/templates/auth.html` with login/register forms
- [ ] `/home/adam/projects/domain-minder/internal/templates/helpers.go` with template functions
- [ ] Color-coded progress bars: green (>60), yellow (30-60), orange (14-30), red (<14), pulsing (<7)
- [ ] HTMX integration for dynamic updates
- [ ] Email verification banner on dashboard
- [ ] Responsive design

## Verification

```bash
cd /home/adam/projects/domain-minder

# Build
go build -o domain-minder ./cmd/server/

# Start server
./domain-minder &
sleep 2

# Register a new user
curl -X POST http://localhost:8080/register \
  -d "email=test@example.com&password=test1234&confirm_password=test1234" \
  -c cookies.txt -b cookies.txt -L

# Check dashboard
curl http://localhost:8080/dashboard -b cookies.txt

# Add some domains with WHOIS
curl -X POST http://localhost:8080/domains \
  -d "name=example.com" \
  -b cookies.txt -L

# Check domains page for progress bars
curl http://localhost:8080/domains -b cookies.txt

# Test HTMX by checking page loads with HTMX script
curl http://localhost:8080/dashboard -b cookies.txt | grep -i htmx

# Cleanup
pkill -f domain-minder
rm -f domain_minder.db cookies.txt
```

## Next Phase

After completing verification, proceed to **Phase 7: Notifications**.
