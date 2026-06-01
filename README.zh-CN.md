# Sola

[English](./README.md) | 中文（简体）

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3-42b883?logo=vue.js&logoColor=white)](https://vuejs.org/)
[![Telegram](https://img.shields.io/badge/Telegram-Bot-26A5E4?logo=telegram&logoColor=white)](https://core.telegram.org/bots)

Sola 是一个基于 Go + Vue 构建的开源 Telegram 运营平台。

它包含真实可接入的 Telegram Bot、Web 管理后台，以及用于长期任务执行的后台 Worker，适合群组和社区运营者将机器人能力做成长期产品，而不只是单一脚本。

## 功能特性

- 支持 Telegram Bot 真实 polling 或 webhook 接入
- 支持群组与超级群管理场景
- 可配置积分系统，支持不同消息类型分值与冷却时间
- 积分排行榜与积分流水
- 群管功能：封禁、禁言、踢人、警告、入群验证
- 定时发帖，支持 Worker 执行与可选自动删除
- 抽奖系统，支持按钮参与和口令参与
- 基于 Vue 3 + Element Plus 的管理后台
- 多租户群归属隔离，防止跨群误操作
- 支持 Docker Compose 部署

## 不包含的模块

以下模块在当前公开版本中不包含：

- USDT 收款
- 会员系统

## 技术栈

### 后端

- Go
- `gotgbot/v2`
- `gin`
- `gorm` + PostgreSQL
- Redis
- `gocron`
- JWT

### 前端

- Vue 3
- Vite
- Element Plus
- ECharts

## 项目结构

```text
cmd/
  api/        管理后台 API 入口
  bot/        Telegram Bot 入口
  worker/     后台任务 Worker 入口
internal/
  api/        HTTP 处理与中间件
  bot/        Telegram 处理器与交互流程
  config/     配置加载
  model/      数据模型
  service/    业务逻辑
  store/      存储初始化
web/          Vue 管理后台
database/
  migrations/ SQL 迁移文件
```

## 快速开始

### 1. 准备环境变量

复制示例环境文件并填写真实配置：

```bash
cp .env.example .env
```

至少需要配置：

- `SOLA_BOT_TOKEN`
- `SOLA_DATABASE_DSN`
- `SOLA_JWT_SECRET`
- `SOLA_APP_ADMIN_USERNAME`
- `SOLA_APP_ADMIN_PASSWORD_HASH` 或 `SOLA_APP_ADMIN_PASSWORD`

生产环境建议优先使用 `SOLA_APP_ADMIN_PASSWORD_HASH`。

### 2. 使用 Docker Compose 启动

```bash
docker compose up -d --build
```

启动服务包括：

- API
- Bot
- Worker
- PostgreSQL
- Redis
- Nginx

### 3. 本地前端开发

```bash
cd web
npm install
npm run dev
```

### 4. 本地 Go 开发

```bash
go mod tidy
go run ./cmd/api
go run ./cmd/bot
go run ./cmd/worker
```

## 安全说明

- 管理员密码支持 bcrypt 哈希校验
- 管理员登录带有基于 Redis 的限流
- CORS 使用白名单，而不是反射任意来源
- 涉及群资源的管理接口带有归属校验
- 前端会话令牌保存在内存中，而不是 `localStorage`

## 配置说明

`config.yaml` 仅适合作为本地默认配置。生产环境建议优先使用环境变量。

重点配置项包括：

- `app.allowed_origins`
- `app.admin_username`
- `app.admin_password_hash`
- `bot.mode`
- `database.dsn`
- `redis.addr`
- `jwt.secret`

## 路线图

- 补充更完整的仓库截图和演示材料
- 继续增强机器人功能模块化能力
- 持续优化多群运营场景下的后台体验
- 增加更多 Bot 流程的自动化测试覆盖

## 参与贡献

欢迎提交 Issue 和 Pull Request。

提交代码时建议注意：

1. 不要提交密钥、真实运行数据和本地环境文件。
2. 尽量保持改动小而聚焦。
3. 遵循现有 Go 与 Vue 项目结构。
4. 如果行为有变化，请同步更新文档。

## 公开发布检查

公开发布前，请确认不要提交以下内容：

- 真实 `.env`
- 真实 bot token
- 本地运行数据
- 本地日志
- 编译产物
- 私有测试素材

## License

本项目基于 MIT License 开源，详见 [LICENSE](./LICENSE)。
