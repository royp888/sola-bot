# Open Source Release Guide

This repository has been cleaned for public release.

## Public Scope

Included in the open-source version:

- Telegram bot
- Admin API
- Admin web panel
- Worker
- Points system
- Group moderation
- Scheduled posts
- Lottery
- Multi-tenant ownership isolation

Not included in the public version:

- USDT payment
- Membership system
- any private production account data

## Cleanup Completed

The local repository cleanup has already removed common non-public artifacts, including:

- compiled binaries
- local runtime folders
- local logs
- reviewer zip packages
- local environment file
- frontend dependency and build output folders
- local test result folders

## What Stays In The Repo

Keep these files and directories in the public repository:

- source code under `cmd/`, `internal/`, `web/src/`, and `database/`
- `README.md`
- `.env.example`
- `Dockerfile`
- `docker-compose.yml`
- `nginx.conf`
- `LICENSE`

## Pre-Publish Checklist

1. Verify `.env.example` contains placeholders only.
2. Verify `config.yaml` contains no private credentials.
3. Run a fresh build from a clean machine or container.
4. Confirm Docker deployment works with copied example config.
5. Add screenshots or demo GIFs if you want a more polished public repo.

## Suggested Repository Metadata

Description:

Sola is an open-source Telegram operations platform with bot automation, moderation, points, scheduled posts, lottery workflows, and a Vue-based admin panel.

Suggested topics:

- telegram-bot
- golang
- vue3
- gin
- gorm
- redis
- postgres
- element-plus
- admin-panel
- telegram

## Suggested First Public Tag

- `v0.1.0`

## License

This project uses the MIT License.
