# Domain Minder

Never lose a domain to expiry again. Domain Minder monitors your domains and sends notifications when they're approaching expiration.

![Dashboard Preview](docs/dashboard.png)

## Features

- **Visual Dashboard**: Color-coded progress bars showing time remaining for each domain
- **Automatic Monitoring**: Background WHOIS checks every 6 hours
- **Multi-Channel Notifications**: Email and Telegram notifications
- **Email Verification**: Ensures notifications go to the right address
- **Customizable Thresholds**: Configure when you want to be notified
- **Self-Hosted**: Run on your own infrastructure

## Quick Start

### Prerequisites

- Go 1.21+
- SQLite3

### Installation

```bash
git clone https://github.com/CaffeinatedTech/domain-minder.git
cd domain-minder
go build -o domain-minder ./cmd/server/
./domain-minder
```

Access at http://localhost:9000

### Docker

```bash
docker compose up -d
```

## Configuration

Configure via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_PATH` | SQLite database file path | `data/domain_minder.db` |
| `PORT` | Server port | `9000` |
| `SESSION_SECRET` | Session encryption key | (required) |
| `CHECK_INTERVAL` | Domain check frequency | `6h` |
| `SMTP_HOST` | SMTP server hostname | (optional) |
| `SMTP_PORT` | SMTP port | `587` |
| `SMTP_USER` | SMTP username | (optional) |
| `SMTP_PASS` | SMTP password | (optional) |
| `SMTP_FROM` | From address for emails | `noreply@localhost` |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token | (optional) |

### Example .env file

```bash
DB_PATH=data/domain_minder.db
PORT=9000
SESSION_SECRET=your-super-secret-key-change-me
CHECK_INTERVAL=6h

# Optional: Email notifications
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=your-email@example.com
SMTP_PASS=your-smtp-password
SMTP_FROM=noreply@yourdomain.com

# Optional: Telegram notifications
TELEGRAM_BOT_TOKEN=your-bot-token
```

## Notification Schedule

Default thresholds: 90, 60, 30, 14, 7, 3, 1 days before expiry

Daily notifications during final week before expiry.

Customize in Settings page.

## API

Domain Minder is primarily a web application. The following endpoints are available:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/dashboard` | User dashboard (auth required) |
| GET | `/domains` | List domains (auth required) |
| POST | `/domains` | Add domain (auth required) |
| POST | `/domains/:id/delete` | Delete domain (auth required) |
| GET | `/settings` | User settings (auth required) |
| POST | `/admin/check` | Trigger domain check (auth required) |

## Development

```bash
# Install dependencies
go mod tidy

# Run tests
go test ./...

# Build binary
go build -o domain-minder ./cmd/server/

# Run with custom config
DB_PATH=/path/to/db SESSION_SECRET=dev ./domain-minder
```

## Deployment

### Docker

```bash
# Build image
docker build -t domain-minder .

# Run container
docker run -d \
  --name domain-minder \
  -p 9000:9000 \
  -v /path/to/data:/app/data \
  -e SESSION_SECRET=your-secret \
  domain-minder
```

### Systemd

Create `/etc/systemd/system/domain-minder.service`:

```ini
[Unit]
Description=Domain Minder - Domain Expiry Monitoring
After=network.target

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/domain-minder
ExecStart=/opt/domain-minder/domain-minder
Environment=DB_PATH=/opt/domain-minder/data/domain_minder.db
Environment=SESSION_SECRET=your-secret
Restart=always

[Install]
WantedBy=multi-user.target
```

### Nginx Proxy

```nginx
server {
    listen 80;
    server_name domain-minder.example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl;
    server_name domain-minder.example.com;

    ssl_certificate /etc/letsencrypt/live/domain-minder.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/domain-minder.example.com/privkey.pem;

    location / {
        proxy_pass http://localhost:9000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Screenshots

### Dashboard
![Dashboard showing domains with progress bars](docs/dashboard.png)

### Add Domain
![Add domain form with WHOIS lookup](docs/add-domain.png)

### Settings
![Notification preferences and thresholds](docs/settings.png)

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make changes and add tests
4. Run tests: `go test ./...`
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.
