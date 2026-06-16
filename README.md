# Sola

中文（简体） | [English](./README.en.md)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3-42b883?logo=vue.js&logoColor=white)](https://vuejs.org/)
[![Telegram](https://img.shields.io/badge/Telegram-Bot-26A5E4?logo=telegram&logoColor=white)](https://core.telegram.org/bots)

Sola 是一个面向 Telegram 群组运营的开源 Bot 平台，集成真实 Bot、Web 管理后台、Mini App 页面和独立 Worker，适合继续二开成长期运营产品，而不是只停留在单一命令机器人阶段。

## 一句话看项目

如果你想做的不是“一个会回指令的 Telegram Bot”，而是一套能真正运营群组、沉淀数据、跑任务、做风控、配后台的系统，Sola 就是为这个场景准备的基础仓库。

## 适合谁

- 想做长期运营型 Telegram Bot 产品的人
- 需要用后台集中管理群配置、用户、发帖任务和抽奖活动的人
- 希望把群管、积分、风控、运营数据放进统一系统的人
- 想基于现成代码二开，而不是从零拼装整套栈的人

## 核心卖点

- 真正的 `Bot + API + Web Admin + Worker + Mini App` 一体化结构
- 支持群积分、群管风控、抽奖、定时发帖、内容模板、自动回复等完整运营链路
- 已接入 AI 垃圾广告二次判定能力，可对接 OpenAI 兼容接口
- 后台按群组归属隔离，适合多群、多 owner 的运营场景
- 使用 PostgreSQL + Redis + SQL migrations，方便持续迭代和正式部署
- 自带 Docker Compose，适合直接拉起本地或服务器环境

## 功能总览

| 模块 | 已实现能力 |
| --- | --- |
| Bot 接入 | `polling`、`webhook`、群组 / 频道绑定、权限检查 |
| 积分系统 | 按消息类型计分、冷却防刷、排行榜、流水、手动调分、签到 |
| 群组管理 | 封禁、解封、禁言、解除禁言、踢出、警告、批量删消息、权限提升/降级、头衔设置、清理注销账号、欢迎语、验证开关 |
| 风控审核 | 关键词过滤、链接限制、转发限制、未验证用户限制、AI 广告判定、违规记录、成员举报 |
| 群规管理 | 设置/清除/查看群规，群规持久化存储 |
| 内容运营 | 自动回复、消息模板、邀请链接追踪、等级成长体系、sed 行内文本修正 |
| 定时发帖 | 一次性任务、循环任务、自动删除、失败自动禁用 |
| 抽奖系统 | 按钮参与、口令参与、后台创建、群内参与、自动开奖（poll_answer 实时接收） |
| 后台系统 | 登录、概览、统计、群组管理、用户管理、备份恢复、审计日志 |
| Mini App | 仪表盘、群设置、快捷发布、抽奖页面 |
| 工程基础 | Docker Compose、Owner 隔离、SQL migrations、审计留痕、Worker 恢复机制、细粒度管理员权限、管理员缓存 |

## Bot 能力

### 基础交互与入口命令

- `/start`、`/menu`、`/settings`、`/help`、`/info`
- `/bind` 绑定群组 / 频道
- `/check_admin` 检查机器人权限

### 群组积分系统

- 按消息类型计分
- 每群独立积分配置
- 冷却时间防刷
- 排行榜、积分流水、手动调分
- `/points`、`/rank`、`/sign`
- `/points_config`、`/set_points`、`/set_cooldown`、`/points_toggle`

### 群组管理与风控

- 封禁、解封、禁言、解除禁言、踢出、警告
- 欢迎语、警告上限、白名单、权限检查
- 入群验证，支持按钮、多选、quiz poll、数学题和 Cloudflare Turnstile（Mini App）交互
- 关键词过滤、链接限制、转发限制、未验证用户限制
- AI 垃圾广告二次判定，可接 OpenAI 兼容接口
- 违规记录、处理状态与审计留痕
- 批量删消息（从回复位置到最新，或指定数量）
- 管理员权限提升/降级、自定义头衔设置
- 举报机制，成员举报后自动通知群管理员
- 清理已注销账号（ghost 用户）
- 细粒度管理员权限：验证、关键词、积分分别独立授权
- `/ban`、`/bans`、`/unban`、`/mute`、`/unmute`、`/kick`
- `/warn`、`/warns`、`/unwarn`
- `/purge`、`/del`
- `/promote`、`/demote`、`/set_title`
- `/report`、`/ban_ghosts`
- `/violations`、`/resolve_violation`、`/ignore_violation`
- `/adminconfig`、`/set_welcome`、`/set_warn_limit`、`/verify_toggle`、`/set_verify`、`/verify_stats`

### 群规

- `/setrules` 设置群规（支持回复消息或直接附文本）
- `/clearrules` 清除当前群规
- `/rules` 任何成员均可查看当前群规

### 内容运营辅助

- 关键词规则
- 自动回复
- 消息模板
- 邀请链接追踪
- 等级与成长体系
- sed 行内文本修正：回复某条消息并发送 `s/旧词/新词`，Bot 会用修正后的版本重新发出该消息（支持 `/`、`|`、`_`、`:` 作为分隔符，支持 `i` 忽略大小写、`g` 全局替换标志）
- `/add_keyword`、`/del_keyword`、`/keywords`
- `/add_reply`、`/del_reply`、`/replies`
- `/add_template`、`/del_template`、`/templates`
- `/invite_create`、`/invite_delete`、`/invites`
- `/set_level`、`/levels`、`/add_level`、`/del_level`

### 定时发帖与抽奖

- 定时发帖支持一次性任务、循环任务、自动删除和失败自动禁用
- 抽奖支持按钮参与、口令参与、群内公告和开奖结果发布
- `/posts`、`/publish`、`/post_create`、`/post_toggle`、`/post_delete`
- `/lottery` 及配套交互面板
- `/stat`、`/stat_week`、`/stats`

## Web 管理后台

后台当前已经有完整页面骨架和主要运营页面：

- 管理员登录与 JWT 会话
- 群组绑定与多租户隔离
- 概览页与运营统计页
- 机器人列表页
  - 展示在线状态、语言、绑定聊天数、心跳时间
  - 当前更偏状态查看，不是完整的机器人生命周期管理
- 群组 / 频道页
- 用户管理
  - 用户列表
  - 调分
  - 封禁、禁言、解除禁言
- 积分系统页面
  - 积分规则配置
  - 积分流水
  - 排行榜查询
- 群管与风控页面
  - 群组设置
  - 封禁与警告
  - 违规记录
  - 审计日志
- 运营工具页面
  - 关键词规则
  - 自动回复
  - 内容模板
  - 邀请链接追踪
  - 等级规则
- 定时发帖管理
  - 创建、编辑、启停、删除
  - 支持媒体、按钮和自动删除配置
- 抽奖管理
  - 创建抽奖
  - 查看参与情况与开奖结果
- 备份恢复
  - 配置或全量数据导出
  - JSON 文件恢复

## Mini App

仓库内已经包含 `web/src/mini` 页面结构：

- 仪表盘
- 群设置
- 快捷发布
- 抽奖页面

同时预留了 `bot.mini_app_url` 配置，可作为 Telegram Mini App 的接入基础。当前更适合作为轻量运营入口和后续二开的基座。


## 架构概览

```mermaid
flowchart TD
    TG["Telegram Users / Groups"] --> BOT["Bot Runtime\ncmd/bot"]
    BOT --> API["Admin API\ncmd/api"]
    BOT --> WORKER["Worker\ncmd/worker"]
    API --> WEB["Web Admin\nweb/src"]
    API --> MINI["Mini App\nweb/src/mini"]
    API --> PG["PostgreSQL"]
    API --> REDIS["Redis"]
    WORKER --> PG
    WORKER --> REDIS
    BOT --> PG
    BOT --> REDIS
```

## 技术栈

### 后端

- Go
- [gotgbot/v2](https://github.com/PaulSonOfLars/gotgbot)
- [gin](https://github.com/gin-gonic/gin)
- [gorm](https://gorm.io/) + PostgreSQL
- Redis
- gocron
- JWT

### 前端

- Vue 3
- Vite
- Element Plus
- ECharts

### 部署

- Docker
- Docker Compose
- Nginx

## 目录结构

```text
cmd/
  api/        管理后台 API 入口
  bot/        Telegram Bot 入口
  worker/     后台任务 Worker 入口
internal/
  api/        HTTP handler、中间件、鉴权
  bot/        Telegram handler、命令与交互流程
  config/     配置加载
  model/      GORM 模型
  service/    业务逻辑
  store/      DB / Redis 初始化
web/          Vue 3 管理后台
web/mini/     Telegram Mini App / 轻量前端
database/
  migrations/ SQL 迁移脚本
```

## 快速开始

### 1. 配置环境变量

复制模板并填写必要配置：

```bash
cp .env.example .env
```

至少需要设置：

- `SOLA_BOT_TOKEN`
- `SOLA_DATABASE_DSN`
- `SOLA_JWT_SECRET`
- `SOLA_APP_ADMIN_USERNAME`
- `SOLA_APP_ADMIN_PASSWORD_HASH` 或 `SOLA_APP_ADMIN_PASSWORD`

生产环境建议使用 `SOLA_APP_ADMIN_PASSWORD_HASH`，不要长期保留明文密码。

### 2. 启动整套服务

```bash
docker compose up -d --build
```

默认会启动：

- `postgres`
- `redis`
- `migrate`
- `api`
- `bot`
- `worker`
- `nginx`

其中：

- `migrate` 会执行 `database/migrations/*.up.sql`
- `api` 默认只在容器网络中暴露
- `nginx` 对外提供后台入口

如果你需要本机直接调试 API，可以临时启用：

```bash
docker compose --profile direct-api up -d api-direct
```

### 3. 本地启动前端

```bash
cd web
npm install
npm run dev
```

### 4. 本地启动后端服务

```bash
go mod tidy
go run ./cmd/api
go run ./cmd/bot
go run ./cmd/worker
```

## 配置说明

`config.yaml` 适合做本地默认配置，生产环境建议优先使用环境变量覆盖。

常用配置项包括：

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

`.env.example` 已给出当前项目实际使用的变量名，可直接作为部署起点。

## Worker 与工程能力

- API / Bot / Worker 三进程拆分
- PostgreSQL + Redis
- SQL migrations 管理表结构
- Docker Compose 一键编排
- Owner 归属校验和后台隔离
- 审计日志与操作留痕
- Worker 启动自动恢复已启用的定时发帖任务
- 定时任务失败计数与自动禁用保护
- 入群验证超时扫描处理
- 定时开奖与异步任务调度

## 安全与隔离

当前版本已经补上几项关键基础安全边界：

- 管理员密码支持 bcrypt 哈希校验
- 登录接口支持 Redis 限流
- CORS 使用白名单，不反射任意 Origin
- 关键读写接口增加群组归属校验
- 前端会话不存储在 `localStorage`
- 后台保留审计和违规记录

不过这并不等于默认就是完整的生产安全方案。正式上线前，仍建议你继续完善域名、TLS、网络出口、日志采集、监控告警和最小权限部署。

## 当前验证状态

当前仓库更准确的描述是：

- 可以构建
- 可以部署
- 核心链路已经接通
- 适合作为二次开发基础仓库

已完成的基础验证包括：

- `go test ./...`
- `npm --prefix web run build`
- 关键多租户隔离接口回归

仍建议你在自己的真实环境中继续验证：

- Bot 在目标群中是否具备管理员权限
- 服务器到 Telegram 的网络是否稳定
- 定时发帖是否按预期执行
- 抽奖、封禁、禁言、入群验证是否完整跑通
- 重启后的任务与状态恢复是否符合预期

## 数据库与迁移

项目使用 `database/migrations/` 管理数据库结构变更，而不是依赖运行时随手 `AutoMigrate`。

推荐做法：

- 所有表结构变更走 SQL migration
- 索引变更和 partial index 一并纳入迁移
- 先迁移数据库，再启动 API / Bot / Worker
- 把回滚脚本纳入发布流程

## 生产部署建议

如果你准备正式上线，建议至少做到这些：

1. 使用强随机 `JWT secret`
2. 使用 bcrypt 管理员密码哈希
3. 仅通过 Nginx 暴露后台入口
4. 为 PostgreSQL 和 Redis 配置持久化
5. 提前确认服务器到 Telegram 的连通性
6. 启用基础日志、告警和备份
7. 先在测试群验证所有群管链路

## 已知边界

当前版本仍有一些明确边界：

- 机器人管理页当前更偏状态查看，不是完整的多 Bot 生命周期平台
- Mini App 已有基础页面与接入位，但仍适合继续二开扩展
- Worker 调度和统计查询还有优化空间
- 大数据量场景下，部分列表和统计仍可继续做索引与 SQL 优化
- Telegram 真实行为仍然依赖你的 Bot 权限和部署网络

## 贡献方式

欢迎提交 Issue 和 Pull Request。

建议遵循以下约定：

1. 不提交真实密钥、真实运行数据和本地日志
2. 改动尽量聚焦，不顺手大改无关模块
3. 行为变化时同步更新文档
4. 数据库结构变更请配套 migration

## 开源前检查

公开仓库前，请确认没有提交以下内容：

- 真实 `.env`
- 真实 Telegram Bot Token
- 本地数据库文件或导出数据
- 本地运行日志
- 前端构建产物
- 私有测试素材

## License

This project is licensed under the MIT License. See [LICENSE](./LICENSE).
