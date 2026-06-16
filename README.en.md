# Sola

[简体中文](./README.md) | English

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3-42b883?logo=vue.js&logoColor=white)](https://vuejs.org/)
[![Telegram](https://img.shields.io/badge/Telegram-Bot-26A5E4?logo=telegram&logoColor=white)](https://core.telegram.org/bots)

Sola is an open-source Telegram operations platform built around real group workflows. It includes a real Telegram bot, a web admin panel, Mini App pages, and a dedicated worker process. Rather than being a single-purpose command bot template, Sola is meant to be a more production-oriented foundation for long-term operations, secondary development, and deployment.

## Project Positioning

Many Telegram bot repositories solve one isolated problem well, such as keyword filtering, anti-spam, sign-in, or lotteries, but stop short of providing an admin backend, task execution layer, operational data model, or multi-chat isolation. Sola tries to put those pieces into one codebase so you can continue building an actual product on top of it.

It is especially suitable if you want to:

- build a long-running operations-focused Telegram bot product
- manage chat settings, users, scheduled posts, and lotteries from a web panel
- keep moderation, points, risk control, and operational data in one system
- start from a practical codebase instead of assembling the full stack from scratch

## Features

### Bot Capabilities

- Real Telegram bot integration with both `polling` and `webhook`
- Core entry and utility commands
  - `/start`, `/menu`, `/settings`, `/help`, `/info`
  - `/bind` for chat and channel binding
  - `/check_admin` for permission checks
- Group points system
  - points by message type
  - per-chat point configuration
  - cooldown-based anti-spam controls
  - rankings, point logs, and manual point adjustments
  - `/points`, `/rank`, `/sign`, `/points_config`, `/set_points`, `/set_cooldown`, `/points_toggle`
- Group moderation and risk control
  - ban, unban, mute, unmute, kick, and warnings
  - welcome messages, warning limits, whitelist support, and permission checks
  - join verification with button, multi-choice, and quiz-poll interactions
  - keyword filtering, link restrictions, forward restrictions, and unverified-user restrictions
  - AI-based secondary spam and advertisement judgement through OpenAI-compatible APIs
  - violation records, resolution states, and audit traces
  - bulk message deletion (from a replied message to latest, or by count)
  - admin promote/demote and custom title assignment
  - member report flow that notifies group admins
  - ghost-user cleanup (ban deleted/cancelled Telegram accounts)
  - granular admin permissions: verify, keyword, and points can each be granted independently
  - `/ban`, `/bans`, `/unban`, `/mute`, `/unmute`, `/kick`, `/warn`, `/warns`, `/unwarn`
  - `/purge`, `/del`
  - `/promote`, `/demote`, `/set_title`
  - `/report`, `/ban_ghosts`
  - `/violations`, `/resolve_violation`, `/ignore_violation`
  - `/adminconfig`, `/set_welcome`, `/set_warn_limit`, `/verify_toggle`, `/set_verify`, `/verify_stats`
- Group rules
  - `/setrules` to set group rules (reply to a message or attach text inline)
  - `/clearrules` to clear the current rules
  - `/rules` to display rules (available to all members)
- Content operations helpers
  - keyword rules
  - auto replies
  - message templates
  - invite link tracking
  - level and growth configuration
  - sed-style inline text correction: reply to a message with `s/old/new` and the bot re-sends it with the corrected text (supports `/`, `|`, `_`, `:` as delimiters; `i` and `g` flags)
  - `/add_keyword`, `/del_keyword`, `/keywords`
  - `/add_reply`, `/del_reply`, `/replies`
  - `/add_template`, `/del_template`, `/templates`
  - `/invite_create`, `/invite_delete`, `/invites`
  - `/set_level`, `/levels`, `/add_level`, `/del_level`
- Scheduled posting
  - one-time tasks
  - recurring tasks
  - auto-delete support
  - failure counting with auto-disable protection
  - `/posts`, `/publish`, `/post_create`, `/post_toggle`, `/post_delete`
- Lottery system
  - button-based participation
  - keyword-based participation
  - in-group announcements and result publishing
  - created from the admin panel, joined inside the group, and drawn automatically by the worker
  - `/lottery` and related interaction panels
- Statistics commands
  - `/stat`, `/stat_week`, `/stats`

### Web Admin

- Admin login with JWT sessions
- Chat binding and multi-tenant isolation
- Overview and operations analytics pages
- Bot list page
  - shows status, language, bound chat count, and heartbeat time
  - currently more of an observability page than a full bot lifecycle management surface
- Chat and channel pages
- User management
  - user list
  - point adjustment
  - ban, mute, and unmute actions
- Points pages
  - point rule configuration
  - point logs
  - ranking queries
- Moderation and risk-control pages
  - group settings
  - bans and warnings
  - violation records
  - audit logs
- Operations pages
  - keywords
  - auto replies
  - content templates
  - invite link tracking
  - level rules
- Scheduled post management
  - create, edit, toggle, and delete
  - supports media, inline buttons, and auto-delete configuration
- Lottery management
  - create lotteries
  - inspect participants and results
- Backup and restore
  - business-config or full-data export
  - JSON restore

### Mini App / Lightweight Pages

The repository already contains a `web/src/mini` surface with:

- dashboard
- chat settings
- quick publish
- lottery page

The `bot.mini_app_url` configuration is also present as an entry point for Telegram Mini App integration. At the current stage, this is best treated as a lightweight operations surface and a base for further extension.

### Worker and Engineering Capabilities

- split API, bot, and worker processes
- PostgreSQL + Redis
- SQL migrations for schema management
- one-command Docker Compose deployment
- owner-based access isolation
- audit logging and operation traceability
- worker startup recovery for enabled scheduled posts
- failure counting and auto-disable protection for broken scheduled jobs
- timeout scanning for join verification
- scheduled lottery draw with `poll_answer` update type for real-time participation
- in-memory TTL cache for admin member lists to reduce Bot API round-trips
- graceful shutdown closes DB alongside Redis

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
          +--------------------+-------------------+
          |                    |                   |
          v                    v                   v
+-----------------------+  +-----------------------+  +-----------------------+
|  cmd/api              |  |  cmd/worker           |  |  web / web/mini       |
|  Admin API            |  |  Jobs / async tasks   |  |  Admin + Mini frontend|
+-----------------------+  +-----------------------+  +-----------------------+
          |                    |                   |
          +---------+----------+-------------------+
                    |
                    v
          PostgreSQL + Redis
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

## Repository Structure

```text
cmd/
  api/        Admin API entry
  bot/        Telegram bot entry
  worker/     Background worker entry
internal/
  api/        HTTP handlers, middleware, auth
  bot/        Telegram handlers, commands, interaction flows
  config/     Configuration loading
  model/      GORM models
  service/    Business logic
  store/      DB and Redis bootstrap
web/          Vue 3 admin panel
web/mini/     Telegram Mini App / lightweight frontend
database/
  migrations/ SQL migrations
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

- the bot management page is currently more about status visibility than full multi-bot lifecycle management
- the Mini App already has page structure and integration hooks, but it is still best extended further in real deployments
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
