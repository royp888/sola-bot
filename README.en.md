# Sola

[简体中文](./README.md) | English

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3-42b883?logo=vue.js&logoColor=white)](https://vuejs.org/)
[![Telegram](https://img.shields.io/badge/Telegram-Bot-26A5E4?logo=telegram&logoColor=white)](https://core.telegram.org/bots)

Sola is an open-source Telegram operations platform for group-centric use cases. It includes a real Telegram bot, a web admin panel, and a dedicated worker process. Rather than being a single-purpose bot script, Sola is intended as an engineering-ready foundation for long-term operations, secondary development, and production deployment.

## Project Positioning

Sola is built for a common gap in the Telegram bot ecosystem: many open-source bots solve one isolated problem well, but do not provide a usable admin backend, task execution layer, data persistence, or multi-group management model. This project brings the bot, API, web admin, and worker into one codebase so you can extend it into a real product.

It is a good fit if you want to:

- build an operations-focused Telegram bot product
- manage chats, users, scheduled posts, and lotteries from a web panel
- start from a practical codebase instead of assembling the entire stack from scratch

## Features

### Bot Features

- Real Telegram bot integration with both `polling` and `webhook`
- Group points system
  - points by message type
  - per-chat point configuration
  - anti-spam cooldown controls
  - commands such as `/points`, `/rank`, and `/points_config`
- Group moderation
  - ban and unban
  - mute and unmute
  - warnings and violation records
  - join verification
- Lottery system
  - button-based participation
  - keyword-based participation
  - created from the admin panel and joined inside the group
- Scheduled posting
  - one-time tasks
  - recurring tasks
  - auto-delete support

### Web Admin

- Admin login with JWT session handling
- Chat binding and multi-tenant isolation
- User and points management
- Point configuration and point logs
- Group settings, ban records, and violation records
- Scheduled post management
- Lottery management
- Statistics dashboard

### Engineering Capabilities

- Split API, bot, and worker processes
- PostgreSQL + Redis
- SQL migrations for schema management
- One-command Docker Compose deployment
- Owner-based access isolation
- Audit logging and operation traceability

## Architecture Overview

```text
Telegram Users / Groups
          |
          v
      Telegram Bot
          |
          v
+-----------------------+
|  cmd/bot              |
|  Message handling     |
|  Group actions        |
+-----------------------+
          |
          +--------------------+
          |                    |
          v                    v
+-----------------------+  +-----------------------+
|  cmd/api              |  |  cmd/worker           |
|  Admin API            |  |  Async and scheduled  |
|                       |  |  task execution       |
+-----------------------+  +-----------------------+
          |                    |
          +---------+----------+
                    |
                    v
          PostgreSQL + Redis
                    |
                    v
             Vue 3 Admin Web
```

## Tech Stack

### Backend

- Go
- [gotgbot/v2](https://github.com/PaulSonOfLars/gotgbot)
- [gin](https://github.com/gin-gonic/gin)
- [gorm](https://gorm.io/) + PostgreSQL
- Redis
- gocron
- JWT

### Frontend

- Vue 3
- Vite
- Element Plus
- ECharts

### Deployment

- Docker
- Docker Compose
- Nginx

## Project Structure

```text
cmd/
  api/        admin API entrypoint
  bot/        Telegram bot entrypoint
  worker/     background worker entrypoint
internal/
  api/        HTTP handlers, middleware, auth
  bot/        Telegram handlers, commands, flows
  config/     configuration loading
  model/      GORM models
  service/    business logic
  store/      database and Redis setup
web/          Vue 3 admin panel
database/
  migrations/ SQL migration files
```

## Quick Start

### 1. Configure environment variables

Copy the example file and fill in the required values:

```bash
cp .env.example .env
```

At minimum, configure:

- `SOLA_BOT_TOKEN`
- `SOLA_DATABASE_DSN`
- `SOLA_JWT_SECRET`
- `SOLA_APP_ADMIN_USERNAME`
- `SOLA_APP_ADMIN_PASSWORD_HASH` or `SOLA_APP_ADMIN_PASSWORD`

For production, prefer `SOLA_APP_ADMIN_PASSWORD_HASH` and avoid storing plain-text passwords.

### 2. Start the full stack

```bash
docker compose up -d --build
```

The default compose setup starts:

- `postgres`
- `redis`
- `migrate`
- `api`
- `bot`
- `worker`
- `nginx`

Notes:

- `migrate` runs `database/migrations/*.up.sql`
- `api` is only exposed inside the compose network by default
- `nginx` serves the external admin entrypoint

If you want to access the API directly for local debugging:

```bash
docker compose --profile direct-api up -d api-direct
```

### 3. Run the frontend locally

```bash
cd web
npm install
npm run dev
```

### 4. Run backend services locally

```bash
go mod tidy
go run ./cmd/api
go run ./cmd/bot
go run ./cmd/worker
```

## Configuration

`config.yaml` works well as a local default configuration file, while production deployments should primarily rely on environment variables.

Common configuration keys include:

- `app.allowed_origins`
- `app.admin_username`
- `app.admin_password_hash`
- `bot.token`
- `bot.mode`
- `bot.mini_app_url`
- `database.dsn`
- `redis.addr`
- `jwt.secret`
- `ai_filter.*`

The `.env.example` file reflects the actual variable names used by the project and is the best starting point for deployment.

## Security and Isolation

The current codebase already includes several important baseline protections:

- bcrypt-based admin password verification
- Redis-backed rate limiting for login
- CORS allowlist instead of reflecting arbitrary origins
- ownership checks for sensitive chat-scoped endpoints
- frontend session handling that avoids `localStorage`
- audit and violation records for operational traceability

That said, this should not be treated as a complete production security program by default. Before going live, you should still harden TLS, domain setup, outbound network access, secret management, logging, alerting, and least-privilege deployment.

## Validation Status

At this stage, the project is best described as:

- buildable
- deployable
- functionally connected across its core flows
- suitable as a secondary-development foundation

Baseline verification already performed includes:

- `go test ./...`
- `npm --prefix web run build`
- regression coverage for key multi-tenant isolation paths

You should still validate the following in your own environment:

- whether the bot has the right admin permissions in the target group
- whether your server can reach Telegram reliably
- whether scheduled posts execute as expected in your deployment
- whether lottery, moderation, and join verification flows run correctly in a real group
- whether restart behavior matches your operational expectations

## Database and Migrations

Database schema changes are managed through `database/migrations/` rather than relying on runtime `AutoMigrate` behavior.

Recommended practice:

- manage all schema changes with SQL migrations
- include index and partial-index changes in the same migration workflow
- migrate before starting API, bot, and worker services
- include rollback steps in your release process

## Production Recommendations

If you plan to deploy Sola in a real environment, at minimum you should:

1. use a strong random `JWT secret`
2. use bcrypt for admin password storage
3. expose the admin surface through Nginx only
4. configure persistent storage for PostgreSQL and Redis
5. verify Telegram network reachability from the target server
6. enable logging, backups, and alerting
7. validate all moderation flows in a test group before using a production group

## Known Boundaries

There are still some clear boundaries in the current version:

- worker scheduling and statistical queries can be optimized further
- some list and reporting paths may need additional SQL or index tuning at larger scale
- real Telegram behavior still depends on your bot permissions and deployment network

## Contributing

Issues and pull requests are welcome.

Please keep these conventions in mind:

1. do not commit real secrets, real runtime data, or local logs
2. keep changes focused instead of bundling unrelated refactors
3. update documentation when behavior changes
4. include migrations when database schema changes are introduced

## Open Source Checklist

Before publishing or sharing the repository, make sure you have not committed:

- a real `.env`
- a real Telegram bot token
- local database dumps or exported data
- local runtime logs
- frontend build artifacts
- private test assets

## License

This project is licensed under the MIT License. See [LICENSE](./LICENSE).
