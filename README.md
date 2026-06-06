# Sola

中文（简体） | [English](./README.en.md)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3-42b883?logo=vue.js&logoColor=white)](https://vuejs.org/)
[![Telegram](https://img.shields.io/badge/Telegram-Bot-26A5E4?logo=telegram&logoColor=white)](https://core.telegram.org/bots)

Sola 是一个面向 Telegram 群组运营的开源 Bot 平台，提供 Bot 服务、Web 管理后台和独立 Worker。它的定位不是“单文件命令机器人”，而是一套适合长期运营、二次开发和部署上线的工程化基础仓库。

## 项目定位

Sola 主要解决的是这类问题：很多 Telegram Bot 仓库能完成某一个单点功能，但缺少后台、任务系统、数据沉淀和多群管理能力。这个项目把 Bot、API、后台和 Worker 放进同一个工程，方便你直接在它之上继续做产品。

它适合：

- 想做运营型 Telegram Bot 产品的人
- 需要 Web 后台管理群配置、用户、抽奖和发帖任务的人
- 希望基于现成代码继续扩展功能，而不是从零搭框架的人

## 功能概览

### Bot 侧

- 真实 Telegram Bot 接入，支持 `polling` 和 `webhook`
- 群组积分系统
  - 按消息类型计分
  - 每群独立积分配置
  - 冷却时间防刷
  - `/points`、`/rank`、`/points_config` 等命令
- 群组管理
  - 封禁、解封
  - 禁言、解除禁言
  - 警告与违规记录
  - 入群验证
- 抽奖系统
  - 按钮参与
  - 口令参与
  - 后台创建，群内交互
- 定时发帖
  - 一次性任务
  - 循环任务
  - 自动删除

### Web 后台

- 管理员登录与 JWT 会话
- 群组绑定与多租户隔离
- 用户列表与积分管理
- 积分配置与积分流水
- 群组配置、封禁记录、违规记录
- 定时发帖管理
- 抽奖管理
- 数据统计页面

### 工程能力

- API / Bot / Worker 三进程拆分
- PostgreSQL + Redis
- SQL migrations 管理表结构
- Docker Compose 一键编排
- Owner 归属校验和后台隔离
- 审计日志与操作留痕

## 架构概览

```text
Telegram Users / Groups
          |
          v
      Telegram Bot
          |
          v
+-----------------------+
|  cmd/bot              |
|  消息处理 / 群内动作   |
+-----------------------+
          |
          +--------------------+
          |                    |
          v                    v
+-----------------------+  +-----------------------+
|  cmd/api              |  |  cmd/worker           |
|  管理后台 API         |  |  定时任务 / 异步任务   |
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

## 安全与隔离

当前版本已经补上几项很关键的基础安全边界：

- 管理员密码支持 bcrypt 哈希校验
- 登录接口支持 Redis 限流
- CORS 使用白名单，不反射任意 Origin
- 关键读写接口增加群组归属校验
- 前端会话不存储在 `localStorage`
- 后台保留审计和违规记录

不过这并不等于默认就是完整的生产安全方案。正式上线前，仍建议自行完善域名、TLS、网络出口、日志采集、监控告警和最小权限部署。

## 当前验证状态

当前仓库更适合被描述为：

- 可构建
- 可部署
- 核心功能链路已打通
- 适合作为二次开发基础仓库

已完成的基础验证包括：

- `go test ./...`
- `npm --prefix web run build`
- 关键多租户隔离接口回归

仍建议你在自己的环境中继续验证：

- Bot 在真实群中是否具备管理员权限
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

- Worker 调度和统计查询还有优化空间
- 大数据量场景下，部分列表和统计仍可继续做索引与 SQL 优化
- Telegram 真机行为依赖你的 Bot 权限和部署网络

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
