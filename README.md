# Sola

中文（简体） | [English](./README.en.md)

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
- 面向多群运营的任务式后台导航与管理页面
- 支持 Docker Compose 部署

## 工程能力

- 多租户群归属隔离，限制后台接口只能操作所属群资源
- 大表后台列表支持 cursor 分页，适合持续滚动查询和审计列表
- 数据库结构通过显式 SQL 迁移管理，不依赖生产环境 AutoMigrate
- 补齐高频复合索引、部分索引与 N+1 查询优化，适配积分、违规、抽奖、定时任务等场景
- 管理后台支持路由懒加载、依赖拆包和重资源页面按需加载

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

默认 compose 编排会先运行一次性 `migrate` 服务，按文件名顺序执行 `database/migrations/*.up.sql`，成功后再启动 `api`、`bot`、`worker`。

默认 compose 编排只向宿主机暴露 `nginx`，`api` 容器仅在 compose 内部网络可达；如需临时直连调试，请显式开启直连 profile：

```bash
docker compose --profile direct-api up -d api-direct
```

`config.yaml` 已将 Swagger 默认关闭。如需在开发或受控运维环境开启，请显式设置 `SOLA_APP_ENABLE_SWAGGER=true`。

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

## 生产可用性

公开版本已经覆盖后台登录、跨群资源访问、前端令牌存放、接口暴露面控制等基础安全边界，也补齐了后台大列表、数据库结构变更和前端加载成本的工程治理。

重点约束包括：

- 生产环境必须使用非默认的 `SOLA_JWT_SECRET` 与 `SOLA_APP_ADMIN_PASSWORD_HASH` / 强密码，否则启动会被拒绝
- `database.auto_migrate` 默认关闭，建议只通过 `database/migrations/` 中的显式迁移变更表结构
- 后台 API 默认通过 Nginx 暴露，避免绕过反向代理直接访问
- Swagger 默认关闭，仅建议在本地联调或受控环境临时开启
- 对于日志、模板、邀请链接、违规记录等后台列表，优先使用 cursor 分页而不是深层 `OFFSET`

## 数据库迁移与表结构变更

Sola 将数据库变更视为显式发布内容，而不是运行时副作用。

建议实践：

- 所有表结构和索引变更通过 `database/migrations/` 统一管理
- 缺失表结构、复合索引和部分索引优先通过迁移脚本落库
- 迁移与回滚脚本保持成对维护，避免线上结构漂移
- 不把迁移文件号、索引名称列表当作 README 的核心内容，具体细节以 SQL 文件为准

这种方式更适合持续部署、问题回滚和多人协作，也更适合生产环境进行审计。

### 首次部署与升级执行

由于 `database.auto_migrate` 默认关闭，首次部署和版本升级前都应先执行数据库迁移。

- `docker compose up -d --build` 会通过一次性 `migrate` 服务自动执行尚未应用的 `*.up.sql`
- 不使用 compose 时，仍应通过现有发布系统或数据库变更流程执行 `database/migrations/` 下的 SQL 文件
- 首次部署时，按顺序执行全部 `*.up.sql`
- 版本升级时，只执行尚未应用的新迁移文件
- 在应用新版本 API、Bot、Worker 之前，先完成迁移并确认回滚脚本可用

## 分页与列表性能

后台的高频列表不再默认依赖深层 `OFFSET` 翻页，而是逐步切换为更适合大表的 cursor 分页语义。

这类接口主要包括：

- 违规记录列表
- 模板列表
- 邀请链接列表
- 积分流水与部分审计类列表

这样做的目标不是暴露底层分页编码细节，而是降低深分页带来的数据库扫描成本，并让管理后台在数据增长后仍保持稳定查询体验。

## 前端加载优化

管理后台已经做了面向真实使用场景的加载优化：

- 路由页面按需懒加载
- Vue 基础依赖、ECharts 和通用 vendor 进行拆包
- 统计页图表改为按需能力注册，避免整包图表库进入首屏
- 后台导航与管理页围绕运营任务流重新组织，减少功能散落感

目标是降低后台首次加载和重资源页面的下载压力，而不是追求孤立的构建跑分。

## 生产部署建议

- 默认通过 Nginx 暴露服务，`docker-compose.yml` 中 API 仅绑定到本机 `127.0.0.1`，避免绕过反向代理直接访问
- 保持 `app.enable_swagger=false`，只在本地联调时临时开启
- PostgreSQL 建议启用 `pg_stat_statements`，并基于真实 SQL 执行 `EXPLAIN ANALYZE` 校准索引，避免盲目堆叠写入成本
- API、Bot、Worker 同时连接数据库时，建议显式设置连接池上限，避免生产环境连接数失控

### PostgreSQL 观测建议

如果要把 PostgreSQL 查询观测真正落地到 `pg_stat_statements`，还需要在数据库启动层启用预加载参数：

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

随后对目标数据库执行一次：

```sql
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
```

仓库没有自动启用该扩展，是因为它依赖数据库实例启动参数，最终仍需由部署环境控制。

排查慢查询时，优先关注：

- 积分流水列表与积分排行榜
- 违规记录列表
- 定时任务与帖子扫描
- 抽奖列表与开奖相关查询

确认 SQL 形态后，再执行 `EXPLAIN ANALYZE` 验证索引命中情况。

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

## 许可证

本项目基于 MIT 许可证开源，详见 [LICENSE](./LICENSE)。
