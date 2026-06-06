# Sola

English | [中文（简体）](./README.md)

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
- **Group moderation**: ban, mute, kick, warn, keyword filter, spam scoring, link domain whitelist/blacklist
- **Join verification**: 5 modes (button / captcha / multi-choice / Telegram Poll / math), difficulty levels, whitelist, unverified user restrictions
- **Scheduled posts**: text / photo / video / document + HTML formatting + Inline Keyboard buttons, worker execution + optional auto-delete
- **Instant publish**: `/publish` with HTML rich text + Inline Keyboard buttons
- **Chinese natural-language commands**: send check-in, points, rank, lottery, add-points in plain Chinese
- Lottery system with button and keyword participation flows
- **AI spam detection**: optional LLM-based secondary check via DeepSeek/OpenAI/custom proxy
- **Audit logging**: all moderation actions auto-recorded, with admin panel query page
- Vue 3 + Element Plus admin panel
- Task-oriented admin navigation for multi-group operators
- Docker Compose deployment

## Engineering Capabilities

- Tenant-scoped chat ownership isolation for admin APIs
- Cursor-based pagination for large admin lists and audit-heavy views
- Explicit SQL migrations for schema and index changes instead of production AutoMigrate reliance
- Query-shape-oriented index tuning and N+1 cleanup for points, violations, lotteries, and scheduled jobs
- Route-level lazy loading and dependency chunk splitting for the admin frontend
- Bot handler panic recovery prevents single-handler crashes from taking down the service
- Async audit log writes that never block the main business flow

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

The default compose stack now runs a one-shot `migrate` service first. It applies `database/migrations/*.up.sql` in filename order before starting `api`, `bot`, or `worker`.

The default compose stack only publishes `nginx` to the host. The API container is reachable only on the internal compose network unless you explicitly opt into direct host exposure:

```bash
docker compose --profile direct-api up -d api-direct
```

Swagger is disabled by default in `config.yaml`. Enable it explicitly for development or controlled admin environments with `SOLA_APP_ENABLE_SWAGGER=true`.

### 3. Frontend env and Telegram login

Frontend builds read `web/.env.production` in production and `web/.env.local` for local overrides. Do not rely on editing `web/.env.example` alone.

Recommended values:

- `VITE_API_BASE_URL=/api`
- `VITE_APP_NAME=Your admin title`
- `VITE_BOT_USERNAME=your_bot_username` (without `@`)

To enable the Telegram Login Widget, make sure all of the following are true:

- rerun `npm run build` after changing frontend env vars
- production `nginx.conf` allows `https://telegram.org` scripts, the Telegram Login Widget requirement for `unsafe-eval`, and `https://oauth.telegram.org` iframe / form sources
- run `/setdomain` in `@BotFather` for the bot and set the final hostname, for example `www.example.com`
- if Cloudflare is enabled, verify direct or DNS-only access first, then switch to proxy mode with SSL set to `Full (strict)`

Typical refresh flow:

```bash
cd web
npm run build
cd ..
docker compose up -d --force-recreate nginx
```

### 4. Local frontend development

```bash
cd web
npm install
npm run dev
```

### 5. Local Go development

```bash
go mod tidy
go run ./cmd/api
go run ./cmd/bot
go run ./cmd/worker
```

## Production Readiness

The public version already covers the baseline operational boundaries expected from a long-lived admin system: admin login protection, chat ownership enforcement, reduced frontend token exposure, controlled API surface area, explicit schema evolution, and scalable list handling.

The most important defaults are:

- production startup rejects weak default JWT/admin credentials
- `database.auto_migrate` is disabled by default
- admin APIs are expected to sit behind Nginx instead of direct exposure
- Swagger stays off unless explicitly enabled for development or controlled environments
- large admin lists should prefer cursor pagination over deep offset scans

## Database Migrations and Schema Changes

Sola treats database changes as an explicit release artifact rather than a runtime side effect.

Recommended practice:

- manage all schema and index changes through `database/migrations/`
- keep missing tables, composite indexes, and partial indexes in migration files
- maintain up/down migration pairs together
- use README for strategy, and SQL files for exact migration details

This makes upgrades, rollback planning, and production review easier than relying on implicit migration at process startup.

### Running Migrations For First Deploys And Upgrades

Because `database.auto_migrate` is disabled by default, run database migrations before a first deployment and before rolling out upgraded services.

- `docker compose up -d --build` now runs unapplied `*.up.sql` files automatically through the one-shot `migrate` service
- outside compose, keep applying the SQL files under `database/migrations/` through your existing deployment or database migration workflow
- on first deploy, run all `*.up.sql` files in order
- on upgrades, run only the new migrations that have not been applied yet
- complete migration and rollback verification before starting the upgraded API, bot, or worker processes

## Pagination and List Performance

Large admin lists are no longer expected to rely on deep `OFFSET` pagination by default. Cursor-based pagination is available for the high-churn views where stable scrolling matters more than arbitrary page numbers.

Typical examples include:

- violation lists
- template lists
- invite-link lists
- point logs and other audit-style admin lists

The intent is not to expose cursor encoding internals in the README, but to make the operational posture clear: list queries are being shaped for growing datasets instead of only for small demo installs.

## Frontend Loading Optimizations

The admin frontend has been tuned for real operator workflows rather than a single all-in-one bundle:

- route-level lazy loading for view pages
- dependency splitting for Vue, ECharts, and general vendor assets
- on-demand chart capability registration instead of loading the full chart stack into the initial path
- task-oriented navigation and denser list management surfaces for repeated daily use

The goal is to reduce initial download cost and make heavier admin views feel more focused, not to chase synthetic benchmark numbers.

## Deployment Notes

- expose the system through Nginx by default; avoid bypassing the reverse proxy with direct API exposure unless it is a deliberate temporary debugging path
- in the default compose setup, the API binds to `127.0.0.1` and stays behind the reverse proxy unless you explicitly enable the direct API profile
- keep `app.enable_swagger=false` in production
- set explicit database connection pool limits when API, bot, and worker share one PostgreSQL instance
- use `pg_stat_statements` and `EXPLAIN ANALYZE` against real workloads before adding or changing indexes

### PostgreSQL Observability

For PostgreSQL statement-level visibility, enable `pg_stat_statements` at the database layer:

```yaml
services:
  postgres:
    command:
      - postgres
      - -c
      - shared_preload_libraries=pg_stat_statements
      - -c
      - pg_stat_statements.max=10000
      - -c
      - pg_stat_statements.track=all
```

Then run once per database:

```sql
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
```

The repository does not auto-enable this extension because it depends on database startup flags controlled by the deployment environment.

When reviewing slow paths, prioritize:

- point logs and leaderboard queries
- violation list queries
- scheduled-post scanning and due-job evaluation
- lottery listing and winner-selection queries

After identifying the real SQL shape, validate index usage with `EXPLAIN ANALYZE`.

## Security Notes

- Admin password supports bcrypt hash verification
- Admin login includes Redis-based rate limiting
- CORS uses an allowlist instead of reflecting arbitrary origins
- Sensitive chat resources are protected by ownership checks
- Frontend session token is kept in memory instead of `localStorage`
- Moderation actions (ban/mute/kick/keyword hit) auto-recorded to `audit_logs`, queryable via admin panel

## Join Verification

Sola provides 5 verification modes configurable via `/set_verify`:

| Type | Command | Description |
|------|---------|-------------|
| `button` | `/set_verify type button` | Tap "I'm human" to pass |
| `captcha` | `/set_verify type captcha` | Enter a random numeric code |
| `multi_choice` | `/set_verify type multi_choice` | Custom question + multi-choice buttons |
| `poll` | `/set_verify type poll` | Native Telegram quiz poll |
| `math` | `/set_verify type math` | Random math problem (e.g. 7+3=?), 4 options |

Additional capabilities:
- **Difficulty levels**: `/set_verify difficulty easy|medium|hard` (affects timeout and retries)
- **Verification whitelist**: `/allowuser @user` skips verification
- **Unverified restrictions**: auto-delete URL and media messages before verification
- **Stats**: `/verify_stats` shows pass/fail/timeout counts

## Rich Media Announcements

Both scheduled posts and instant publish support:

```text
Plain text + <b>HTML</b> formatting + Inline Keyboard buttons  ✅
Photo    + <b>HTML</b> caption  + Inline Keyboard buttons  ✅
Video    + <b>HTML</b> caption  + Inline Keyboard buttons  ✅
Document + <b>HTML</b> caption  + Inline Keyboard buttons  ✅
```

Button JSON example (stored in `inline_keyboard_json` field):

```json
[
  [{"text":"Details","url":"https://example.com"}, {"text":"Join","callback_data":"join"}],
  [{"text":"DM me","url":"tg://user?id=123456"}]
]
```

Supported HTML tags: `<b>` `<i>` `<u>` `<s>` `<a href>` `<code>` `<pre>` `<blockquote>` `<tg-spoiler>`

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

- ✅ Join verification enhancement (Poll / Math / multi-choice / difficulty / whitelist)
- ✅ Link domain whitelist/blacklist
- ✅ Operation audit logging system
- ✅ Chinese natural-language commands
- ✅ Rich media announcements (HTML + photo/video + buttons)
- ✅ Bot menu emoji beautification
- ✅ AI spam detection (DeepSeek/OpenAI)
- [ ] Add better repository screenshots and demo material
- [ ] Multi-language i18n support
- [ ] Add more automated integration coverage for bot flows
- [ ] Channel-to-group message sync

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
