# sola-bot 优化任务清单（供 Codex 执行）

> 工作目录：`internal/bot/`  
> 构建命令：`go build ./...`（完成每项任务后必须通过）  
> 执行顺序：按编号顺序执行，每项完成后运行构建验证。

---

## ✅ 已完成（P1–P3，Task 1–11）

以下任务已由 Codex 实现并经人工审核验证通过（`go build / go vet / go test` 全绿）。

| Task | 内容 | 结果 |
|------|------|------|
| 1 | `admin.go` 拆分 → `admin.go`(294行) + `verify.go`(814行) + `moderation.go`(515行) | ✅ |
| 2 | `button` 验证改为简单点击确认，新增 `sendButtonChallenge` | ✅ |
| 3 | `cmd/bot/main.go` 注册 BotCommands（群成员/管理员/私聊三 scope） | ✅ |
| 4 | `CallbackData()` 加 64 字节越界保护，超出返回 `noop` | ✅ |
| 5 | `/help` 分角色输出（成员 8 行 / 管理员 30 行） | ✅ |
| 6 | `routePrivateCallback` 顶层 `needsChat` map，`selectedPrivateChat` 只调一次 | ✅ |
| 7 | 删除冗余别名 `/settings` `/mod` `/points_rank` | ✅ |
| 8 | verify 回调域统一为 `verify:check` / `verify:answer` | ✅ |
| 9 | `scheduleVerifyTimeoutKick`：`time.AfterFunc` + Redis key 检查自动踢人 | ✅ |
| 10 | 关键词 `reply_text` 在 warn/delete 动作时同步触发 | ✅ |
| 11 | `poll_answer` allowed update 已注册 | ✅ |

---

## ✅ 用户已实现功能（勿重复实现）

以下功能由测试用户贡献，已在部署环境生效，Codex **不得重复实现或覆盖**。

| 功能 | 说明 |
|------|------|
| 二次验证（Turnstile） | 主验证通过后弹 Cloudflare Turnstile 人机验证，已打通前后端 API |
| 内容锁定（6 项） | 锁定链接/媒体/转发/贴纸/语音/GIF，独立开关，管理员豁免 |
| 防刷屏（AntiFlood） | 窗口内消息频率超限自动禁言/踢出/封禁，可配时长 |
| 欢送消息 | 成员离开/被踢时发欢送消息，30 秒后自动删 |
| 自动删除 bot 消息 | bot 发消息后 N 秒自动撤回，范围 5–300s |
| 账户安全模块 | 后台多管理员 CRUD、super_admin/admin 角色、密码修改、权限隔离 |
| Handler 中间件（a.wrap） | 所有 handler 已包含日志、panic 恢复、执行时长记录 |
| ext.ContinueGroups 过滤 | 已过滤正常跳过日志，日志不再刷屏 |

---

## P5 — Bug 修复（优先于 P4，按严重程度排）

> 这些是测试用户报告的真实 bug，全部为代码层可修复，每项修完必须 `go build ./...` 验证。

---

### Bug 1：管理员/群主入群也触发验证（高危）

**现象：** 群主或管理员入群，`handleNewChatMembers`（`verify.go`）没有检查用户角色，触发验证循环，反复发验证码。

**修复：** 在 `handleNewChatMembers` 对每个新成员调用 `GetChatMember` 检查角色，`creator` 和 `administrator` 直接跳过：

```go
// verify.go: handleNewChatMembers 内，遍历 member 时加前置检查
member, err := b.GetChatMemberWithContext(scope.Context, scope.Chat.ID, newMember.Id, nil)
if err == nil {
    switch member.(type) {
    case gotgbot.ChatMemberOwner, gotgbot.ChatMemberAdministrator:
        continue // 跳过管理员验证
    }
}
```

---

### Bug 2：`handleCaptchaReply` 未注册（高危）

**现象：** 代码中已实现 captcha 回复处理 handler，但 `app.go` 的 `Register()` 中没有注册，验证码回复消息无人处理。

**修复：** 在 `app.go` 的 `Register` 中，加在 `handleNewChatMembers` 注册之后：

```go
dispatcher.AddHandler(handlers.NewMessage(message.All, a.handleCaptchaReply))
```

同时检查 `antiflood` 和 `locks.go` 的 handler 是否也有同样的漏注册问题，一并补上。

---

### Bug 3：handler 注册顺序导致验证拦截自动回复（高危）

**现象：** 当前注册顺序：`captcha → checkUnverifiedMessage → 中文命令 → 自动回复`。未验证用户发消息被步骤 2 删除后，自动回复找不到原消息，报 `Bad Request: message to be replied not found`。

**修复：** 调整 `app.go` 注册顺序，自动回复 handler（`handleAutoReply`）必须在 `checkUnverifiedMessage`（验证拦截删消息）**之前**注册：

```go
// 正确顺序
dispatcher.AddHandler(handlers.NewMessage(message.All, a.handleAutoReply))       // 先处理自动回复
dispatcher.AddHandler(handlers.NewMessage(message.All, a.handleCaptchaReply))     // 再处理验证码
dispatcher.AddHandler(handlers.NewMessage(message.All, a.checkUnverifiedMessage)) // 最后删未验证用户消息
dispatcher.AddHandler(handlers.NewMessage(message.All, a.handleChineseCommand))
dispatcher.AddHandler(handlers.NewMessage(message.All, a.handleMessageModeration))
dispatcher.AddHandler(handlers.NewMessage(message.All, a.handleMessagePoints))
```

---

### Bug 4：管理员密码明文比对（高危）

**现象：** 后台登录接口直接比对明文密码，数据库泄露即泄露所有密码。

**修复：** 引入 `golang.org/x/crypto/bcrypt`（已在 Go 标准生态），将比对逻辑改为：

```go
import "golang.org/x/crypto/bcrypt"

// 比对时
if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(inputPassword)); err != nil {
    return ErrInvalidCredentials
}

// 存储新密码时
hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
if err != nil {
    return err
}
// 存 string(hash) 到 DB
```

同步更新账户安全模块的密码修改接口。

---

### Bug 5：数据库缺字段 `verify_second_enabled`（中危）

**现象：** 新增功能依赖的字段没有自动迁移，只能手动 `ALTER TABLE`，导致部署后立即报错。

**修复：** 在迁移脚本目录（`migrations/` 或 `db/migrations/`）新建一个迁移文件，包含所有累积的缺失字段：

```sql
-- up
ALTER TABLE chat_admin_configs ADD COLUMN IF NOT EXISTS verify_second_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE chat_admin_configs ADD COLUMN IF NOT EXISTS goodbye_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE chat_admin_configs ADD COLUMN IF NOT EXISTS goodbye_text TEXT NOT NULL DEFAULT '';
ALTER TABLE chat_admin_configs ADD COLUMN IF NOT EXISTS auto_delete_bot_msg_seconds INTEGER NOT NULL DEFAULT 0;
ALTER TABLE chat_admin_configs ADD COLUMN IF NOT EXISTS rules_text TEXT NOT NULL DEFAULT '';
-- 以及 content lock 相关字段（links/media/forward/sticker/voice/gif）
ALTER TABLE chat_admin_configs ADD COLUMN IF NOT EXISTS lock_links BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE chat_admin_configs ADD COLUMN IF NOT EXISTS lock_media BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE chat_admin_configs ADD COLUMN IF NOT EXISTS lock_forward BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE chat_admin_configs ADD COLUMN IF NOT EXISTS lock_sticker BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE chat_admin_configs ADD COLUMN IF NOT EXISTS lock_voice BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE chat_admin_configs ADD COLUMN IF NOT EXISTS lock_gif BOOLEAN NOT NULL DEFAULT FALSE;

-- down
-- ALTER TABLE chat_admin_configs DROP COLUMN IF EXISTS verify_second_enabled; (按需写)
```

确保 `bootstrap/migrate.go`（或同等入口）在启动时自动执行所有 pending 迁移。

---

### Bug 6：`ext.ContinueOrder` 不存在（编译失败）

**现象：** `gotgbot v2.0.0-rc.35` 没有 `ext.ContinueOrder` 常量，代码直接编译失败。

**修复：** 全局搜索 `ext.ContinueOrder`，替换为实际存在的常量 `ext.ContinueGroups`：

```bash
grep -rn "ContinueOrder" internal/
```

将所有命中行的 `ext.ContinueOrder` 替换为 `ext.ContinueGroups`。

---

### ⚠️ 平台限制（非代码 bug，文档说明即可）

| 问题 | 说明 | 应对 |
|------|------|------|
| 隐藏成员群组验证不触发 | `has_hidden_members=true` 时 TG API 不发 `chat_member` 更新，bot 无法感知入群事件 | 群主需关闭「隐藏群成员」设置；admin 面板可加提示 |
| Docker 构建超时导致旧镜像 | Go 编译约 100s，terminal 超时会 kill 进程，新代码未进镜像 | 部署后执行 `docker images` 核对时间戳；建议 CI/CD 或设置足够长的 timeout |
| 409 Conflict 轮询冲突 | 快速重启时旧连接未释放，新容器约 30s 内收不到更新 | 部署流程改为 `docker compose stop && sleep 10 && docker compose start` |

---

## P4 — 新功能扩展（Task 12–20）

> 参考来源：WilliamButcherBot、Alita_Robot、TLG_AntiJoin2SpamBot  
> 新增外部依赖：Task 20 需 `go get github.com/mmcdole/gofeed`

---

### Task 12：`/purge` 批量删除消息

**问题：** 群里只有 `/del` 删单条，处理刷屏需要手动一条条操作。

**参考：** WilliamButcherBot `admin.py` — purge 接收数量参数，按 100 条分批删。

**操作：** 新建 `internal/bot/purge.go`（或追加到 `moderation.go`）：

```go
func (a *App) handlePurge(b *gotgbot.Bot, ctx *ext.Context) error {
    scope := requestScope(ctx)
    if err := a.requireTelegramManager(b, ctx); err != nil {
        return err
    }
    msg := ctx.Message
    if msg == nil {
        return sendText(b, ctx, "用法：回复目标消息 /purge，或 /purge 100 删最近100条。", nil)
    }
    fromID := msg.MessageId
    if msg.ReplyToMessage != nil {
        fromID = msg.ReplyToMessage.MessageId
    }
    count := 100
    if args := commandArgs(ctx); len(args) > 0 {
        if n, err := strconv.Atoi(args[0]); err == nil && n > 0 && n <= 500 {
            count = n
        }
    }
    var ids []int64
    for i := 0; i < count; i++ {
        ids = append(ids, fromID-int64(i))
    }
    deleted := 0
    for i := 0; i < len(ids); i += 100 {
        end := i + 100
        if end > len(ids) {
            end = len(ids)
        }
        batch := make([]int64, end-i)
        copy(batch, ids[i:end])
        if _, err := b.DeleteMessagesWithContext(scope.Context, scope.Chat.ID, batch, nil); err == nil {
            deleted += len(batch)
        }
    }
    _, _ = b.DeleteMessageWithContext(scope.Context, scope.Chat.ID, msg.MessageId, nil)
    notice, _ := b.SendMessageWithContext(scope.Context, scope.Chat.ID,
        fmt.Sprintf("已删除 %d 条消息。", deleted), nil)
    if notice != nil {
        go deleteMessageLater(b, scope.Chat.ID, notice.MessageId, 5*time.Second)
    }
    return nil
}

func (a *App) handleDel(b *gotgbot.Bot, ctx *ext.Context) error {
    if err := a.requireTelegramManager(b, ctx); err != nil {
        return err
    }
    if ctx.Message == nil || ctx.Message.ReplyToMessage == nil {
        return sendText(b, ctx, "请回复要删除的消息后发送 /del。", nil)
    }
    scope := requestScope(ctx)
    _, _ = b.DeleteMessageWithContext(scope.Context, scope.Chat.ID, ctx.Message.ReplyToMessage.MessageId, nil)
    _, _ = b.DeleteMessageWithContext(scope.Context, scope.Chat.ID, ctx.Message.MessageId, nil)
    return nil
}
```

`app.go` 注册：
```go
dispatcher.AddHandler(handlers.NewCommand("purge", a.wrap(a.handlePurge, a.RateLimit("cmd:purge", 1))))
dispatcher.AddHandler(handlers.NewCommand("del",   a.wrap(a.handleDel,   a.RateLimit("cmd:del",   1))))
```

---

### Task 13：`/promote /demote /set_title` 管理员晋升降级

**问题：** sola-bot 无法通过 bot 晋升/降级管理员，必须手动在 Telegram 设置里操作。

**参考：** WilliamButcherBot `admin.py` — promote 给基础权限，demote 清空，set_title 设自定义头衔。

**操作：** 追加到 `moderation.go`（或新建 `admin_promote.go`）：

```go
func (a *App) handlePromote(b *gotgbot.Bot, ctx *ext.Context) error {
    scope := requestScope(ctx)
    if err := a.requireTelegramManager(b, ctx); err != nil {
        return err
    }
    targetID, _, err := moderationTarget(ctx)
    if err != nil {
        return sendText(b, ctx, "用法：回复目标用户消息 /promote，或 /promote 用户ID。", nil)
    }
    yes := true
    if _, err := b.PromoteChatMemberWithContext(scope.Context, scope.Chat.ID, targetID,
        &gotgbot.PromoteChatMemberOpts{
            CanDeleteMessages:  true,
            CanRestrictMembers: true,
            CanInviteUsers:     true,
            CanPinMessages:     &yes,
            CanManageChat:      true,
        }); err != nil {
        return sendText(b, ctx, "晋升失败："+err.Error(), nil)
    }
    a.invalidateAdminCache(scope.Chat.ID)
    return sendText(b, ctx, fmt.Sprintf("✅ 已晋升用户 %d 为管理员（基础权限）。", targetID), nil)
}

func (a *App) handleDemote(b *gotgbot.Bot, ctx *ext.Context) error {
    scope := requestScope(ctx)
    if err := a.requireTelegramManager(b, ctx); err != nil {
        return err
    }
    targetID, _, err := moderationTarget(ctx)
    if err != nil {
        return sendText(b, ctx, "用法：回复目标用户消息 /demote。", nil)
    }
    if _, err := b.PromoteChatMemberWithContext(scope.Context, scope.Chat.ID, targetID,
        &gotgbot.PromoteChatMemberOpts{}); err != nil {
        return sendText(b, ctx, "降级失败："+err.Error(), nil)
    }
    a.invalidateAdminCache(scope.Chat.ID)
    return sendText(b, ctx, fmt.Sprintf("✅ 已撤销用户 %d 的管理员权限。", targetID), nil)
}

func (a *App) handleSetTitle(b *gotgbot.Bot, ctx *ext.Context) error {
    scope := requestScope(ctx)
    if err := a.requireTelegramManager(b, ctx); err != nil {
        return err
    }
    targetID, args, err := moderationTarget(ctx)
    if err != nil || len(args) == 0 {
        return sendText(b, ctx, "用法：回复目标用户消息 /set_title 自定义头衔。", nil)
    }
    title := strings.TrimSpace(strings.Join(args, " "))
    if _, err := b.SetChatAdministratorCustomTitleWithContext(scope.Context, scope.Chat.ID, targetID, title, nil); err != nil {
        return sendText(b, ctx, "设置头衔失败："+err.Error(), nil)
    }
    return sendText(b, ctx, fmt.Sprintf("✅ 已将用户 %d 的管理员头衔设为「%s」。", targetID, title), nil)
}
```

`app.go` 注册：
```go
dispatcher.AddHandler(handlers.NewCommand("promote",   a.wrap(a.handlePromote,  a.RateLimit("cmd:promote",   1))))
dispatcher.AddHandler(handlers.NewCommand("demote",    a.wrap(a.handleDemote,   a.RateLimit("cmd:demote",    1))))
dispatcher.AddHandler(handlers.NewCommand("set_title", a.wrap(a.handleSetTitle, a.RateLimit("cmd:set_title", 1))))
```

---

### Task 14：`/setrules` / `/rules`（群规 PM 发送）

**问题：** sola-bot 无群规系统，群主只能把规则手动发到群里刷屏。

**参考：** WilliamButcherBot `rules.py` — `/rules` 发按钮，用户点击后在私聊收到群规，不占群聊空间。

**操作：**

1. `types.go` 的 `ChatAdminConfig` 添加字段，`ChatAdminConfigPatch` 同步添加指针字段：
   ```go
   // ChatAdminConfig
   RulesText string

   // ChatAdminConfigPatch
   RulesText *string
   ```

2. 新建 `internal/bot/rules.go`：

```go
package bot

import (
    "fmt"
    "strconv"
    "strings"

    "github.com/PaulSonOfLars/gotgbot/v2"
    "github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func (a *App) handleSetRules(b *gotgbot.Bot, ctx *ext.Context) error {
    scope := requestScope(ctx)
    if err := a.requireTelegramManager(b, ctx); err != nil {
        return err
    }
    if a.services.Admin == nil {
        return sendText(b, ctx, "群组配置服务尚未接入。", nil)
    }
    text := strings.TrimSpace(strings.Join(commandArgs(ctx), " "))
    if ctx.Message != nil && ctx.Message.ReplyToMessage != nil && text == "" {
        text = ctx.Message.ReplyToMessage.Text
    }
    if text == "" {
        return sendText(b, ctx, "用法：/setrules 群规内容，或回复一条消息。", nil)
    }
    if _, err := a.services.Admin.UpdateConfig(scope.Context, scope.Chat.ID,
        ChatAdminConfigPatch{RulesText: &text}); err != nil {
        return err
    }
    return sendText(b, ctx, "✅ 群规已保存。", nil)
}

func (a *App) handleClearRules(b *gotgbot.Bot, ctx *ext.Context) error {
    scope := requestScope(ctx)
    if err := a.requireTelegramManager(b, ctx); err != nil {
        return err
    }
    if a.services.Admin == nil {
        return sendText(b, ctx, "群组配置服务尚未接入。", nil)
    }
    empty := ""
    if _, err := a.services.Admin.UpdateConfig(scope.Context, scope.Chat.ID,
        ChatAdminConfigPatch{RulesText: &empty}); err != nil {
        return err
    }
    return sendText(b, ctx, "✅ 群规已清除。", nil)
}

func (a *App) handleRules(b *gotgbot.Bot, ctx *ext.Context) error {
    scope := requestScope(ctx)
    // 私聊：解析 deep link 参数 rules_{chatID} 直接发群规
    if scope.Chat.Type == "private" {
        args := commandArgs(ctx)
        if len(args) > 0 && strings.HasPrefix(args[0], "rules_") {
            chatID, err := strconv.ParseInt(strings.TrimPrefix(args[0], "rules_"), 10, 64)
            if err == nil && a.services.Admin != nil {
                cfg, _ := a.services.Admin.GetConfig(scope.Context, chatID)
                if strings.TrimSpace(cfg.RulesText) != "" {
                    return sendText(b, ctx, "📋 群规\n\n"+cfg.RulesText, nil)
                }
            }
        }
        return sendText(b, ctx, "该群暂无群规。", nil)
    }
    // 群组：发按钮引导私聊查看
    botInfo, err := b.GetMe(nil)
    if err != nil {
        return err
    }
    deepLink := fmt.Sprintf("https://t.me/%s?start=rules_%d", botInfo.Username, scope.Chat.ID)
    return sendText(b, ctx, "📋 群规", &gotgbot.SendMessageOpts{
        ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
            {{Text: "点击查看群规", URL: deepLink}},
        }},
    })
}
```

3. `app.go` 注册：
```go
dispatcher.AddHandler(handlers.NewCommand("setrules",   a.wrap(a.handleSetRules,   a.RateLimit("cmd:setrules",   1))))
dispatcher.AddHandler(handlers.NewCommand("clearrules", a.wrap(a.handleClearRules, a.RateLimit("cmd:clearrules", 1))))
dispatcher.AddHandler(handlers.NewCommand("rules",      a.wrap(a.handleRules,      a.RateLimit("cmd:rules",      1))))
```

---

### Task 15：`s/pattern/replacement/` Sed 纠错

**问题：** 群成员发错字时需要重发，不方便。

**参考：** WilliamButcherBot `regex.py` — 回复消息后发 `s/错/对/`，bot 替换后输出；有 ReDoS 超时保护。

**操作：** 新建 `internal/bot/sed.go`：

```go
package bot

import (
    "context"
    "fmt"
    "regexp"
    "strings"
    "time"

    "github.com/PaulSonOfLars/gotgbot/v2"
    "github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// sedPattern 匹配 s/from/to[/flags]，支持 / | _ : 四种分隔符
var sedPattern = regexp.MustCompile(`^s([/|_:])(.+)\1(.*)\1([gi]*)$`)

func (a *App) handleSed(b *gotgbot.Bot, ctx *ext.Context) error {
    if ctx.Message == nil || ctx.Message.ReplyToMessage == nil {
        return nil
    }
    text := strings.TrimSpace(ctx.Message.Text)
    m := sedPattern.FindStringSubmatch(text)
    if m == nil || m[2] == "" {
        return nil
    }
    from, to, flags := m[2], m[3], m[4]
    target := ctx.Message.ReplyToMessage.Text
    if target == "" {
        target = ctx.Message.ReplyToMessage.Caption
    }
    if target == "" {
        return nil
    }

    reFlags := "(?m)"
    if strings.Contains(flags, "i") {
        reFlags = "(?im)"
    }

    done := make(chan string, 1)
    go func() {
        re, err := regexp.Compile(reFlags + from)
        if err != nil {
            done <- ""
            return
        }
        if strings.Contains(flags, "g") {
            done <- re.ReplaceAllString(target, to)
        } else {
            // 只替换第一处
            done <- re.ReplaceAllLiteralString(
                re.ReplaceAllStringFunc(target, func(s string) string { return to }),
                to,
            )
        }
    }()

    timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    var result string
    select {
    case result = <-done:
    case <-timeoutCtx.Done():
        return nil
    }
    if result == "" || result == target {
        return nil
    }
    if len(result) > 4096 {
        result = result[:4096]
    }
    scope := requestScope(ctx)
    _, _ = b.SendMessageWithContext(scope.Context, scope.Chat.ID,
        fmt.Sprintf("✏️ %s", result),
        &gotgbot.SendMessageOpts{
            ReplyParameters: &gotgbot.ReplyParameters{
                MessageId: ctx.Message.ReplyToMessage.MessageId,
            },
        })
    return nil
}
```

`app.go` 注册（加在 `handleMessage` 系列之前）：
```go
dispatcher.AddHandler(handlers.NewMessage(
    func(msg *gotgbot.Message) bool {
        return msg != nil && len(msg.Text) > 2 &&
            msg.Text[0] == 's' && strings.ContainsAny(string(msg.Text[1]), "/|_:")
    },
    a.handleSed,
))
```

---

### Task 16：`/ban_ghosts` 注销账号清理

**问题：** 群里积累大量注销账号，Bot API 无法枚举全部成员，需基于 sola-bot 自身记录过的入群用户来检查。

**参考：** WilliamButcherBot `admin.py` — 遍历成员列表，封禁 IsDeleted 账号。

**操作：**

1. `AdminService` 接口（`types.go`）追加两个方法：
   ```go
   RecordSeenUser(ctx context.Context, chatID, userID int64) error
   ListSeenUsers(ctx context.Context, chatID int64) ([]int64, error)
   ```

2. `verify.go` 的 `handleNewChatMembers` 中，对每个 member 入群时记录：
   ```go
   if a.services.Admin != nil {
       _ = a.services.Admin.RecordSeenUser(scope.Context, scope.Chat.ID, member.Id)
   }
   ```

3. `moderation.go` 追加 handler：
   ```go
   func (a *App) handleBanGhosts(b *gotgbot.Bot, ctx *ext.Context) error {
       scope := requestScope(ctx)
       if err := a.requireTelegramManager(b, ctx); err != nil {
           return err
       }
       if a.services.Admin == nil {
           return sendText(b, ctx, "群组管理服务尚未接入。", nil)
       }
       userIDs, err := a.services.Admin.ListSeenUsers(scope.Context, scope.Chat.ID)
       if err != nil {
           return err
       }
       kicked := 0
       for _, uid := range userIDs {
           member, err := b.GetChatMemberWithContext(scope.Context, scope.Chat.ID, uid, nil)
           if err != nil {
               continue
           }
           u := member.GetUser()
           // Deleted Account 特征：FirstName 为空且非 bot
           if u.FirstName == "" && !u.IsBot {
               if _, err := b.BanChatMemberWithContext(scope.Context, scope.Chat.ID, uid,
                   &gotgbot.BanChatMemberOpts{RevokeMessages: false}); err == nil {
                   kicked++
               }
           }
       }
       return sendText(b, ctx, fmt.Sprintf("✅ 已清理 %d 个注销账号。", kicked), nil)
   }
   ```

4. `app.go` 注册：
   ```go
   dispatcher.AddHandler(handlers.NewCommand("ban_ghosts", a.wrap(a.handleBanGhosts, a.RateLimit("cmd:ban_ghosts", 1))))
   ```

---

### Task 17：Filters 增强（词边界匹配 + `~` 删触发消息）

**问题：** `handleAutoReply`（`auto_reply.go`）用 `strings.Contains` 子串匹配，误触发率高（"ban"匹配到"band"）；且不支持触发后删除原消息。

**参考：** WilliamButcherBot `filters.py` — 词边界正则；`~keyword` 前缀表示触发后删原消息。

**操作：** 修改 `auto_reply.go` 的 `handleAutoReply`，替换匹配逻辑：

```go
func (a *App) handleAutoReply(b *gotgbot.Bot, ctx *ext.Context) error {
    if a.services.AutoReply == nil || ctx == nil || ctx.Message == nil {
        return nil
    }
    scope := requestScope(ctx)
    msg := ctx.Message
    text := strings.TrimSpace(msg.Text + " " + msg.Caption)
    if text == "" {
        return nil
    }
    matches, err := a.services.AutoReply.MatchAll(scope.Context, msg.Chat.Id, text)
    if err != nil || len(matches) == 0 {
        return nil
    }
    for _, match := range matches {
        keyword := match.Keyword
        deleteOnTrigger := strings.HasPrefix(keyword, "~")
        if deleteOnTrigger {
            keyword = strings.TrimPrefix(keyword, "~")
        }
        // 词边界正则（不再用 strings.Contains）
        pattern := `(?i)(^|[^\w])` + regexp.QuoteMeta(keyword) + `([^\w]|$)`
        re, err := regexp.Compile(pattern)
        if err != nil || !re.MatchString(text) {
            continue
        }
        if deleteOnTrigger {
            _, _ = b.DeleteMessageWithContext(scope.Context, msg.Chat.Id, msg.MessageId, nil)
        }
        if strings.TrimSpace(match.ReplyText) == "" {
            continue
        }
        var replyTo *gotgbot.ReplyParameters
        if !deleteOnTrigger {
            replyTo = &gotgbot.ReplyParameters{MessageId: msg.MessageId}
        }
        _, _ = b.SendMessageWithContext(scope.Context, msg.Chat.Id, match.ReplyText,
            &gotgbot.SendMessageOpts{ReplyParameters: replyTo})
    }
    return nil
}
```

`handleAddReply` 无需改动，关键词以 `~` 开头直接存入 DB 即可。

---

### Task 18：管理员列表 Redis 缓存

**问题：** `requireTelegramManager` 每次都调用 `GetChatAdministrators`，高频群触发限速。

**参考：** Alita_Robot — 管理员列表缓存 1 小时，晋升/降级时主动失效。

**操作：** 新建 `internal/bot/admin_cache.go`：

```go
package bot

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/PaulSonOfLars/gotgbot/v2"
)

const adminCacheTTL = time.Hour

func adminCacheKey(chatID int64) string {
    return fmt.Sprintf("admins:%d", chatID)
}

func (a *App) getCachedAdmins(ctx context.Context, b *gotgbot.Bot, chatID int64) ([]gotgbot.ChatMember, error) {
    if a.services.Redis != nil {
        if raw, err := a.services.Redis.Get(ctx, adminCacheKey(chatID)).Bytes(); err == nil {
            var members []gotgbot.ChatMember
            if json.Unmarshal(raw, &members) == nil {
                return members, nil
            }
        }
    }
    members, err := b.GetChatAdministratorsWithContext(ctx, chatID, nil)
    if err != nil {
        return nil, err
    }
    if a.services.Redis != nil {
        if raw, err := json.Marshal(members); err == nil {
            _ = a.services.Redis.Set(ctx, adminCacheKey(chatID), raw, adminCacheTTL).Err()
        }
    }
    return members, nil
}

func (a *App) invalidateAdminCache(chatID int64) {
    if a.services.Redis != nil {
        _ = a.services.Redis.Del(context.Background(), adminCacheKey(chatID)).Err()
    }
}
```

将 `requireTelegramManager` 中调用 `GetChatAdministrators` 的地方替换为 `a.getCachedAdmins(...)`。

`handlePromote` / `handleDemote` 结尾已调用 `a.invalidateAdminCache(scope.Chat.ID)`（见 Task 13）。

---

### Task 19：`/report` 举报通知管理员

**问题：** 群成员无法快速通知管理员处理违规内容。

**参考：** WilliamButcherBot `admin.py` — 回复消息 `/report` → @ 所有管理员，5 分钟冷却防滥用，不能举报管理员。

**操作：** `moderation.go` 追加：

```go
func (a *App) handleReport(b *gotgbot.Bot, ctx *ext.Context) error {
    scope := requestScope(ctx)
    if scope.Chat.Type == "private" {
        return sendText(b, ctx, "请在群组中回复你要举报的消息后发送 /report。", nil)
    }
    if ctx.Message == nil || ctx.Message.ReplyToMessage == nil {
        return sendText(b, ctx, "请回复目标消息后发送 /report。", nil)
    }
    // 5 分钟冷却
    if a.services.Redis != nil {
        key := fmt.Sprintf("report_cd:%d:%d", scope.Chat.ID, scope.Actor.ID)
        if exists, _ := a.services.Redis.Exists(context.Background(), key).Result(); exists > 0 {
            return sendText(b, ctx, "举报冷却中，请 5 分钟后再试。", nil)
        }
        _ = a.services.Redis.Set(context.Background(), key, "1", 5*time.Minute).Err()
    }
    admins, err := a.getCachedAdmins(scope.Context, b, scope.Chat.ID)
    if err != nil {
        return err
    }
    // 不能举报管理员
    if reported := ctx.Message.ReplyToMessage.From; reported != nil {
        for _, admin := range admins {
            if admin.GetUser().Id == reported.Id {
                return sendText(b, ctx, "不能举报管理员。", nil)
            }
        }
    }
    var mentions []string
    for _, admin := range admins {
        u := admin.GetUser()
        if !u.IsBot && u.Username != "" {
            mentions = append(mentions, "@"+u.Username)
        }
    }
    text := fmt.Sprintf("⚠️ %s 举报了一条消息，请管理员处理。\n%s",
        scope.Actor.FirstName, strings.Join(mentions, " "))
    _, _ = b.SendMessageWithContext(scope.Context, scope.Chat.ID, text,
        &gotgbot.SendMessageOpts{
            ReplyParameters: &gotgbot.ReplyParameters{
                MessageId: ctx.Message.ReplyToMessage.MessageId,
            },
        })
    _, _ = b.DeleteMessageWithContext(scope.Context, scope.Chat.ID, ctx.Message.MessageId, nil)
    return nil
}
```

`app.go` 注册：
```go
dispatcher.AddHandler(handlers.NewCommand("report", a.wrap(a.handleReport, a.RateLimit("cmd:report", 1))))
```

---

### Task 20：RSS 订阅推送

**问题：** 群/频道需要自动拉取外部内容（博客、新闻源）推送，目前只有手动定时发帖。

**参考：** WilliamButcherBot `rss.py` — 每群一个订阅，后台轮询，有新条目推送到群。

**依赖：** `go get github.com/mmcdole/gofeed`

**操作：**

1. `types.go` 新增接口和类型：
   ```go
   type RSSService interface {
       GetFeed(ctx context.Context, chatID int64) (feedURL, lastTitle string, err error)
       SetFeed(ctx context.Context, chatID int64, feedURL, lastTitle string) error
       DeleteFeed(ctx context.Context, chatID int64) error
       ListAllFeeds(ctx context.Context) ([]RSSFeed, error)
   }

   type RSSFeed struct {
       ChatID    int64
       URL       string
       LastTitle string
   }
   ```
   在 `Services` struct 追加 `RSS RSSService`。

2. 新建 `internal/bot/rss.go`：

```go
package bot

import (
    "context"
    "fmt"
    "net/url"
    "strings"
    "time"

    "github.com/PaulSonOfLars/gotgbot/v2"
    "github.com/PaulSonOfLars/gotgbot/v2/ext"
    "github.com/mmcdole/gofeed"
)

func (a *App) handleAddFeed(b *gotgbot.Bot, ctx *ext.Context) error {
    scope := requestScope(ctx)
    if err := a.requireTelegramManager(b, ctx); err != nil {
        return err
    }
    if a.services.RSS == nil {
        return sendText(b, ctx, "RSS 服务尚未接入。", nil)
    }
    args := commandArgs(ctx)
    if len(args) == 0 {
        return sendText(b, ctx, "用法：/add_feed https://example.com/feed.xml", nil)
    }
    feedURL := strings.TrimSpace(args[0])
    if u, err := url.Parse(feedURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
        return sendText(b, ctx, "URL 格式无效，请使用 http/https 链接。", nil)
    }
    fp := gofeed.NewParser()
    feed, err := fp.ParseURL(feedURL)
    if err != nil {
        return sendText(b, ctx, "无法解析 RSS/Atom 源："+err.Error(), nil)
    }
    if err := a.services.RSS.SetFeed(scope.Context, scope.Chat.ID, feedURL, feed.Title); err != nil {
        return err
    }
    return sendText(b, ctx, fmt.Sprintf("✅ 已订阅：%s\n有新内容时会自动推送到本群。", feed.Title), nil)
}

func (a *App) handleRmFeed(b *gotgbot.Bot, ctx *ext.Context) error {
    scope := requestScope(ctx)
    if err := a.requireTelegramManager(b, ctx); err != nil {
        return err
    }
    if a.services.RSS == nil {
        return sendText(b, ctx, "RSS 服务尚未接入。", nil)
    }
    if err := a.services.RSS.DeleteFeed(scope.Context, scope.Chat.ID); err != nil {
        return err
    }
    return sendText(b, ctx, "✅ 已取消 RSS 订阅。", nil)
}

func (a *App) handleListFeeds(b *gotgbot.Bot, ctx *ext.Context) error {
    scope := requestScope(ctx)
    if a.services.RSS == nil {
        return sendText(b, ctx, "RSS 服务尚未接入。", nil)
    }
    feedURL, lastTitle, err := a.services.RSS.GetFeed(scope.Context, scope.Chat.ID)
    if err != nil || feedURL == "" {
        return sendText(b, ctx, "当前群未订阅任何 RSS 源。", nil)
    }
    return sendText(b, ctx, fmt.Sprintf("📡 当前订阅\n%s\n最新条目：%s", feedURL, lastTitle), nil)
}

// PollRSSFeeds 由后台 goroutine 定期调用（建议 15 分钟一次）
func (a *App) PollRSSFeeds(b *gotgbot.Bot) {
    if a.services.RSS == nil {
        return
    }
    feeds, err := a.services.RSS.ListAllFeeds(context.Background())
    if err != nil {
        return
    }
    fp := gofeed.NewParser()
    for _, f := range feeds {
        feed, err := fp.ParseURL(f.URL)
        if err != nil || feed == nil || len(feed.Items) == 0 {
            continue
        }
        latest := feed.Items[0]
        if latest.Title == f.LastTitle {
            continue
        }
        text := fmt.Sprintf("📡 %s\n\n%s\n%s", feed.Title, latest.Title, latest.Link)
        _, _ = b.SendMessageWithContext(context.Background(), f.ChatID, text, nil)
        _ = a.services.RSS.SetFeed(context.Background(), f.ChatID, f.URL, latest.Title)
    }
}
```

3. `cmd/bot/main.go` 启动后追加轮询 goroutine：
   ```go
   go func() {
       for {
           time.Sleep(15 * time.Minute)
           botApp.PollRSSFeeds(tgBot)
       }
   }()
   ```

4. `app.go` 注册：
   ```go
   dispatcher.AddHandler(handlers.NewCommand("add_feed", a.wrap(a.handleAddFeed,   a.RateLimit("cmd:add_feed", 1))))
   dispatcher.AddHandler(handlers.NewCommand("rm_feed",  a.wrap(a.handleRmFeed,    a.RateLimit("cmd:rm_feed",  1))))
   dispatcher.AddHandler(handlers.NewCommand("feeds",    a.wrap(a.handleListFeeds, a.RateLimit("cmd:feeds",    1))))
   ```

---

## 验证步骤（每项 Task 完成后执行）

```bash
go build ./...   # 必须无错误
go vet ./...     # 必须无警告
```

Task 16 完成后额外确认：`AdminService` 接口的两个新方法有对应实现，否则编译报错。  
Task 20 完成后额外确认：`go.mod` 中出现 `github.com/mmcdole/gofeed`。

---

## 执行建议顺序

**P5 Bug 优先：** Bug1（管理员跳过验证）→ Bug2（handler 注册）→ Bug3（handler 顺序）→ Bug4（bcrypt）→ Bug5（DB 迁移）→ Bug6（ContinueGroups）

**P4 新功能：** Task 18（缓存）→ Task 12（purge）→ Task 13（promote）→ Task 19（report）→ Task 17（filters）→ Task 14（rules）→ Task 15（sed）→ Task 16（ban_ghosts）→ Task 20（RSS）

---

## 💡 架构建议（后续迭代参考）

> 非紧急，不需要 Codex 立即执行，供项目长期规划参考。

| # | 建议 | 说明 |
|---|------|------|
| A1 | 插件化模块 | 验证/内容锁/防刷/积分等做成独立模块，通过 `config.yaml` 或 `admin_configs` 按需加载 |
| A2 | 自动化 DB 迁移 | 从手动 `ALTER TABLE` 迁移到 `golang-migrate` 或 `goose`，每个功能一个 `.up.sql` + `.down.sql` |
| A3 | Handler 注册自动化 | 改用显式 `Register()` 方法或反射，消除手动在 `app.go` 逐行添加导致的漏注册问题 |
| A4 | 批量命令支持 | 批量踢人/禁言等，当前所有命令只支持逐条操作 |
| A5 | 关键词规则数据库化 | 当前 `keywords.go` 硬编码规则，改为 DB 存储后台可编辑 |
| A6 | i18n 多语言 | bot 回复全部硬编码中文，多群不同语言需要统一 `i18n.go` |
| A7 | 定时任务可视化 | worker 定时任务当前无 UI，后台加管理界面 |
| A8 | 统一日志面板 | 日志分散在 docker logs 和系统日志，缺少统一的日志查看入口 |
| A9 | 权限粒度细化 | 当前 super_admin/admin 两级，可扩展更细粒度角色 |
