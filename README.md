# Sola

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3-42b883?logo=vue.js&logoColor=white)](https://vuejs.org/)
[![Telegram](https://img.shields.io/badge/Telegram-Bot-26A5E4?logo=telegram&logoColor=white)](https://core.telegram.org/bots)

Sola is an open-source Telegram operations suite built with Go + Vue.

It includes a real Telegram bot, a web admin panel, and background workers for long-running automation. The project is designed for community and group operators who want a product-style system rather than a single-purpose script bot.

## Features

- Telegram bot with real polling or webhook access
- Group and supergroup operations
- Points system with configurable scoring and cooldowns
- Rankings and point logs
- Group moderation tools: ban, mute, kick, warn, join verification
- Scheduled posts with worker execution and optional auto-delete
- Lottery system with button and keyword participation flows
- Admin web panel built with Vue 3 + Element Plus
- Multi-tenant chat ownership isolation for admin APIs
- Docker Compose deployment

## Not Included

The following modules are intentionally not included in this public version:

- USDT payment
- Membership system

## Tech Stack

### Backend

- Go
- `gotgbot/v2`
- `gin`
- `gorm` + PostgreSQL
- Redis
- `gocron`
- JWT

### Frontend

- Vue 3
- Vite
- Element Plus
- ECharts

## Project Structure

```text
cmd/
  api/        Admin API entrypoint
  bot/        Telegram bot entrypoint
  worker/     Background worker entrypoint
internal/
  api/        HTTP handlers and middleware
  bot/        Telegram handlers and bot flows
  config/     Configuration loading
  model/      Database models
  service/    Business logic
  store/      Persistence bootstrap
web/          Vue admin panel
database/
  migrations/ SQL migrations
```

## Quick Start

### 1. Prepare environment

Copy the example environment file and fill in real values:

```bash
cp .env.example .env
```

Required values:

- `SOLA_BOT_TOKEN`
- `SOLA_DATABASE_DSN`
- `SOLA_JWT_SECRET`
- `SOLA_APP_ADMIN_USERNAME`
- `SOLA_APP_ADMIN_PASSWORD_HASH` or `SOLA_APP_ADMIN_PASSWORD`

For production, prefer `SOLA_APP_ADMIN_PASSWORD_HASH`.

### 2. Run with Docker Compose

```bash
docker compose up -d --build
```

Services:

- API
- Bot
- Worker
- PostgreSQL
- Redis
- Nginx

### 3. Local frontend development

```bash
cd web
npm install
npm run dev
```

### 4. Local Go development

```bash
go mod tidy
go run ./cmd/api
go run ./cmd/bot
go run ./cmd/worker
```

## Security Notes

- Admin password supports bcrypt hash verification
- Admin login includes Redis-based rate limiting
- CORS uses an allowlist instead of reflecting arbitrary origins
- Sensitive chat resources are protected by ownership checks
- Frontend session token is kept in memory instead of `localStorage`

## Configuration Notes

`config.yaml` is only a local default. In production, prefer environment variables.

Important settings:

- `app.allowed_origins`
- `app.admin_username`
- `app.admin_password_hash`
- `bot.mode`
- `database.dsn`
- `redis.addr`
- `jwt.secret`

## Roadmap

- Add better repository screenshots and demo material
- Improve modular plugin-style bot features
- Continue polishing admin UX for multi-group operators
- Add more automated integration coverage for bot flows

## Contributing

Issues and pull requests are welcome.

When contributing:

1. Keep secrets and local runtime data out of commits.
2. Prefer small, focused changes.
3. Follow the existing Go and Vue project structure.
4. Update docs when behavior changes.

## Publish Checklist

Before publishing publicly, make sure you do not commit:

- real `.env`
- real bot tokens
- local runtime data
- local logs
- compiled binaries
- private test assets

## License

This project is licensed under the MIT License. See [LICENSE](./LICENSE).
