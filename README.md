# Domain Minder

Never lose a domain to expiry again. Domain Minder monitors your domains and notifies you when they're approaching expiration.

## Features

- **Dashboard View**: Visual countdown bars showing time remaining for each domain
- **Color-Coded Warnings**: Progress bars change color as expiry approaches (green → yellow → red)
- **Multi-Channel Notifications**: Email and Telegram notifications with configurable thresholds
- **WHOIS Integration**: Shows registrar information in notifications
- **Background Monitoring**: Automatic checking without user intervention
- **Modular Design**: Easy to add new notification channels

## Tech Stack

- **Backend**: Go with Echo v4
- **Frontend**: HTMX for dynamic updates
- **Database**: SQLite3
- **Notifications**: Email, Telegram (extensible)

## Getting Started

### Prerequisites

- Go 1.21+
- SQLite3

### Installation

```bash
git clone https://github.com/yourusername/domain-minder.git
cd domain-minder
go build -o domain-minder
./domain-minder
```

Then open http://localhost:9000 in your browser.

## Configuration

Configure via environment variables or `.env` file:

- `DB_PATH` - SQLite database file path (default: `data/domain_minder.db`)
- `PORT` - Server port (default: `9000`)
- `SESSION_SECRET` - Secret for session encryption
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS` - Email settings
- `TELEGRAM_BOT_TOKEN` - Telegram bot token

## Notification Thresholds

Default notification schedule:
- 90 days before expiry
- 60 days before expiry
- 30 days before expiry
- 14 days before expiry
- 7 days before expiry
- 3 days before expiry
- 1 day before expiry
- Daily notifications in final week

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

MIT License - see LICENSE file for details.
