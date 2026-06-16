# Cloudflare Turnstile + Mini App 入群验证 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在现有五种验证类型（button/captcha/multi_choice/poll/math）之外，新增第六种 `turnstile` 类型：用户申请入群时，Bot 给其发私信附带 Mini App WebApp 按钮，用户在 Mini App 内完成 CF Turnstile 验证，后端验证通过后自动批准入群请求。

**Architecture:** Bot 监听 `chat_join_request` 更新（已在 allowed_updates），当群组 VerifyType==turnstile 时向用户私信发 WebApp 按钮；Mini App 新增 `/verify` 页面嵌入 Turnstile widget；API 层暴露一个无需 JWT 的公开端点 `POST /api/verify/turnstile`，用 HMAC 签名防止伪造，校验 CF token 后调 Telegram approveChatJoinRequest/declineChatJoinRequest。

**Tech Stack:** Go (gotgbot/v2, gin, crypto/hmac), Vue 3 + TypeScript (Mini App 前端), Cloudflare Turnstile JS widget (`https://challenges.cloudflare.com/turnstile/v0/api.js`), CF Turnstile server-side verify API (`https://challenges.cloudflare.com/turnstile/v0/siteverify`), Redis (存 pending join request 防重放)

---

## 文件变更地图

| 文件 | 操作 | 职责 |
|------|------|------|
| `internal/config/config.go` | 修改 | 新增 Turnstile 配置块（SiteKey, SecretKey, VerifySecret） |
| `internal/api/types.go` | 修改 | 新增 TurnstileVerifyRequest / TurnstileVerifyResponse |
| `internal/api/verify_handler.go` | 新建 | POST /api/verify/turnstile 端点逻辑 |
| `internal/api/router.go` | 修改 | 注册公开端点 + 传 Turnstile 配置给 Dependencies |
| `internal/api/handlers.go` | 修改 | Dependencies 加 Turnstile 字段 |
| `internal/bot/verify.go` | 修改 | 新增 handleChatJoinRequest + sendTurnstileChallenge + HMAC 签名 |
| `internal/bot/app.go` | 修改 | registerVerifyHandlers 里注册 chat_join_request handler |
| `internal/config/config.go` | 修改 | bindEnv 加 Turnstile 环境变量绑定 |
| `web/src/mini/views/Verify.vue` | 新建 | Mini App 验证页面，嵌入 CF Turnstile widget |
| `web/src/mini/router/index.ts` | 修改 | 加 /verify 路由 |
| `.env.example` | 修改 | 记录 SOLA_TURNSTILE_* 变量 |
| `README.md` | 修改 | 功能总览 + Bot 能力章节补 turnstile 类型 |
| `README.en.md` | 修改 | 同步英文说明 |
| `README.zh-CN.md` | 修改 | 同步路线图已完成条目 |

---

## Task 1: Config — 新增 Turnstile 配置字段

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1.1: 在 Config 结构体里加 Turnstile 块**

打开 `internal/config/config.go`，在 `AiFilter AiFilterConfig` 行后面（`Redis` 块之前）插入：

```go
Turnstile struct {
    SiteKey      string `mapstructure:"site_key"`
    SecretKey    string `mapstructure:"secret_key"`
    VerifySecret string `mapstructure:"verify_secret"`
} `mapstructure:"turnstile"`
```

完整 Config 相关部分变成：

```go
type Config struct {
    App      struct { ... } `mapstructure:"app"`
    Bot      struct { ... } `mapstructure:"bot"`
    Database DatabaseConfig  `mapstructure:"database"`
    AiFilter AiFilterConfig  `mapstructure:"ai_filter"`
    Turnstile struct {
        SiteKey      string `mapstructure:"site_key"`
        SecretKey    string `mapstructure:"secret_key"`
        VerifySecret string `mapstructure:"verify_secret"`
    } `mapstructure:"turnstile"`
    Redis struct { ... } `mapstructure:"redis"`
    JWT   struct { ... } `mapstructure:"jwt"`
}
```

- [ ] **Step 1.2: 在 bindEnv 函数里加环境变量绑定**

在 `bindEnv` 的 `bindings` map 里追加三行：

```go
"turnstile.site_key":      "SOLA_TURNSTILE_SITE_KEY",
"turnstile.secret_key":    "SOLA_TURNSTILE_SECRET_KEY",
"turnstile.verify_secret": "SOLA_TURNSTILE_VERIFY_SECRET",
```

- [ ] **Step 1.3: 验证编译**

```powershell
cd "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot"
go build ./internal/config/...
```

Expected: 无报错输出。

- [ ] **Step 1.4: 更新 .env.example**

打开 `.env.example`，在 AI_FILTER 块后面加：

```env
# Cloudflare Turnstile (type=turnstile 验证模式)
SOLA_TURNSTILE_SITE_KEY=
SOLA_TURNSTILE_SECRET_KEY=
# 用于对 join_request 链接做 HMAC 签名，随机 32 字节 base64 即可
SOLA_TURNSTILE_VERIFY_SECRET=
```

- [ ] **Step 1.5: Commit**

```powershell
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" add internal/config/config.go .env.example
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" commit -m "feat(config): add Turnstile site/secret key and verify_secret fields"
```

---

## Task 2: API Types — 请求/响应结构体

**Files:**
- Modify: `internal/api/types.go`

- [ ] **Step 2.1: 在 types.go 末尾追加**

```go
// TurnstileVerifyRequest is posted by the Mini App after the user passes CF Turnstile.
type TurnstileVerifyRequest struct {
    ChatID  int64  `json:"chat_id"`
    UserID  int64  `json:"user_id"`
    Sig     string `json:"sig"`  // HMAC-SHA256(chat_id|user_id|exp, verify_secret), hex
    Exp     int64  `json:"exp"`  // Unix timestamp the link expires
    CFToken string `json:"cf_token"`
}

// TurnstileVerifyResponse is returned to the Mini App.
type TurnstileVerifyResponse struct {
    OK      bool   `json:"ok"`
    Message string `json:"message,omitempty"`
}
```

- [ ] **Step 2.2: 验证编译**

```powershell
go build ./internal/api/...
```

Expected: 无报错。

- [ ] **Step 2.3: Commit**

```powershell
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" add internal/api/types.go
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" commit -m "feat(api): add TurnstileVerifyRequest/Response types"
```

---

## Task 3: API Handler — POST /api/verify/turnstile

**Files:**
- Create: `internal/api/verify_handler.go`

- [ ] **Step 3.1: 创建文件**

新建 `internal/api/verify_handler.go`，完整内容如下：

```go
package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const cfSiteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// VerifyTurnstile handles POST /api/verify/turnstile.
// It is a public endpoint (no JWT). It:
//  1. Validates the HMAC signature produced by the bot when issuing the Mini App link.
//  2. Calls the CF Turnstile server-side siteverify API.
//  3. On success: calls Telegram approveChatJoinRequest via Bot API.
//  4. On failure: calls Telegram declineChatJoinRequest.
func (s *Server) VerifyTurnstile(c *gin.Context) {
	var req TurnstileVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, TurnstileVerifyResponse{OK: false, Message: "invalid request body"})
		return
	}

	// 1. Signature & expiry check
	secret := s.deps.TurnstileVerifySecret
	if secret == "" {
		c.JSON(http.StatusServiceUnavailable, TurnstileVerifyResponse{OK: false, Message: "turnstile not configured"})
		return
	}
	if time.Now().Unix() > req.Exp {
		c.JSON(http.StatusUnprocessableEntity, TurnstileVerifyResponse{OK: false, Message: "verification link expired"})
		return
	}
	expectedSig := turnstileHMAC(req.ChatID, req.UserID, req.Exp, secret)
	if !hmac.Equal([]byte(req.Sig), []byte(expectedSig)) {
		c.JSON(http.StatusForbidden, TurnstileVerifyResponse{OK: false, Message: "invalid signature"})
		return
	}

	// 2. CF Turnstile verification
	cfSecret := s.deps.TurnstileSecretKey
	if cfSecret == "" {
		c.JSON(http.StatusServiceUnavailable, TurnstileVerifyResponse{OK: false, Message: "turnstile secret not configured"})
		return
	}
	cfOK, err := verifyCFToken(c.Request.Context(), cfSecret, req.CFToken)
	if err != nil {
		c.JSON(http.StatusBadGateway, TurnstileVerifyResponse{OK: false, Message: "failed to verify captcha"})
		return
	}

	botToken := s.deps.BotToken
	if !cfOK {
		// Decline the join request silently
		if botToken != "" {
			_ = telegramJoinAction(c.Request.Context(), botToken, req.ChatID, req.UserID, false)
		}
		c.JSON(http.StatusOK, TurnstileVerifyResponse{OK: false, Message: "captcha verification failed"})
		return
	}

	// 3. Approve
	if botToken != "" {
		if err := telegramJoinAction(c.Request.Context(), botToken, req.ChatID, req.UserID, true); err != nil {
			c.JSON(http.StatusBadGateway, TurnstileVerifyResponse{OK: false, Message: "failed to approve join request"})
			return
		}
	}
	c.JSON(http.StatusOK, TurnstileVerifyResponse{OK: true, Message: "verified"})
}

// turnstileHMAC computes HMAC-SHA256(chat_id|user_id|exp, secret) as a lowercase hex string.
func turnstileHMAC(chatID, userID, exp int64, secret string) string {
	msg := fmt.Sprintf("%d|%d|%d", chatID, userID, exp)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

// verifyCFToken calls the CF Turnstile siteverify endpoint.
func verifyCFToken(ctx context.Context, secret, token string) (bool, error) {
	form := url.Values{
		"secret":   {secret},
		"response": {token},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfSiteverifyURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return false, err
	}
	var result struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, err
	}
	return result.Success, nil
}

// telegramJoinAction calls approveChatJoinRequest or declineChatJoinRequest via Bot API.
func telegramJoinAction(ctx context.Context, botToken string, chatID, userID int64, approve bool) error {
	method := "approveChatJoinRequest"
	if !approve {
		method = "declineChatJoinRequest"
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/%s", botToken, method)
	body, _ := json.Marshal(map[string]any{
		"chat_id": chatID,
		"user_id": userID,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("telegram %s: %s %s", method, strconv.Itoa(resp.StatusCode), string(b))
	}
	return nil
}
```

- [ ] **Step 3.2: 验证编译**

```powershell
go build ./internal/api/...
```

Expected: 报错 `s.deps.TurnstileVerifySecret undefined` 和 `s.deps.TurnstileSecretKey undefined` 和 `s.deps.BotToken undefined`，这是正常的 —— 下一 Task 补 Dependencies 字段。

- [ ] **Step 3.3: Commit 文件（先不跑 build）**

```powershell
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" add internal/api/verify_handler.go
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" commit -m "feat(api): add VerifyTurnstile handler"
```

---

## Task 4: API Dependencies + Router — 接通端点

**Files:**
- Modify: `internal/api/handlers.go`
- Modify: `internal/api/router.go`

- [ ] **Step 4.1: 在 Dependencies 接口 / 结构体里加字段**

打开 `internal/api/handlers.go`，找到 `Dependencies` 结构体（或接口）。在已有字段后追加：

```go
TurnstileVerifySecret string
TurnstileSecretKey    string
BotToken              string
```

如果 `Dependencies` 是一个 `struct`，直接加字段即可。如果是 `interface`，加对应方法并在实现类型上实现。

> **注意：** 先 `grep -n "Dependencies" internal/api/handlers.go` 确认它是 struct 还是 interface。根据实际情况编辑。

- [ ] **Step 4.2: 找到 Dependencies 初始化的地方，传入配置**

通常在 `cmd/api/main.go` 或 `internal/bootstrap/bootstrap.go`。搜索 `Dependencies{` 或 `NewDependencies`。

找到后，加入：

```go
TurnstileVerifySecret: cfg.Turnstile.VerifySecret,
TurnstileSecretKey:    cfg.Turnstile.SecretKey,
BotToken:              cfg.Bot.Token,
```

- [ ] **Step 4.3: 在 router.go 注册公开端点**

打开 `internal/api/router.go`，在 `r.GET("/healthz", server.Health)` 附近（非 JWT 保护区域）追加：

```go
// Public: CF Turnstile verification callback from Mini App — no JWT needed.
r.POST("/api/verify/turnstile", server.VerifyTurnstile)
```

确保这一行在 `secured.Use(JWTMiddleware(...))` 之外。

- [ ] **Step 4.4: 验证编译通过**

```powershell
go build ./...
```

Expected: 零错误。

- [ ] **Step 4.5: Commit**

```powershell
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" add internal/api/handlers.go internal/api/router.go cmd/api/main.go
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" commit -m "feat(api): wire Turnstile handler into Dependencies and router"
```

---

## Task 5: Bot — 处理 chat_join_request + 发 Mini App 验证私信

**Files:**
- Modify: `internal/bot/verify.go`
- Modify: `internal/bot/app.go`（或 ops_commands.go，找 registerCoreHandlers）

- [ ] **Step 5.1: 在 verify.go 末尾追加 HMAC 工具函数和 join_request handler**

打开 `internal/bot/verify.go`，在文件末尾（`buildMathOptions` 函数之后）追加：

```go
// ──────────────────────────────────────────
// Turnstile: chat_join_request flow
// ──────────────────────────────────────────

// GenerateTurnstileLink builds the Mini App URL the user will open for CF verification.
// Format: {miniAppURL}/#/verify?chat={chatID}&user={userID}&sig={hmac}&exp={unix}
func (a *App) GenerateTurnstileLink(chatID, userID int64) string {
	expiry := time.Now().Add(10 * time.Minute).Unix()
	sig := turnstileBotHMAC(chatID, userID, expiry, a.options.TurnstileVerifySecret)
	return fmt.Sprintf("%s/#/verify?chat=%d&user=%d&sig=%s&exp=%d",
		a.miniAppURL, chatID, userID, sig, expiry)
}

// turnstileBotHMAC mirrors the API handler's turnstileHMAC — keep them identical.
func turnstileBotHMAC(chatID, userID, exp int64, secret string) string {
	msg := fmt.Sprintf("%d|%d|%d", chatID, userID, exp)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

// handleChatJoinRequest is invoked for every chat_join_request update.
func (a *App) handleChatJoinRequest(b *gotgbot.Bot, ctx *ext.Context) error {
	if ctx == nil || ctx.ChatJoinRequest == nil || a.services.Admin == nil {
		return nil
	}
	req := ctx.ChatJoinRequest
	chatID := req.Chat.Id
	userID := req.From.Id

	cfg, err := a.services.Admin.GetConfig(context.Background(), chatID)
	if err != nil || !cfg.VerifyEnabled || cfg.VerifyType != "turnstile" {
		// Not our concern — let the request stay pending or be handled manually.
		return nil
	}

	// Guard: mini_app_url and verify_secret must be configured.
	if a.miniAppURL == "" || a.options.TurnstileVerifySecret == "" {
		return nil
	}

	link := a.GenerateTurnstileLink(chatID, userID)
	name := strings.TrimSpace(req.From.FirstName + " " + req.From.LastName)
	if name == "" {
		name = strconv.FormatInt(userID, 10)
	}
	text := fmt.Sprintf("👋 %s，请完成人机验证后加入 %s。\n\n点击下方按钮，在页面内通过验证即可自动入群。",
		name, req.Chat.Title)

	_, err = b.SendMessageWithContext(context.Background(), userID, text, &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{{
					Text:   "🔒 完成验证入群",
					WebApp: &gotgbot.WebAppInfo{Url: link},
				}},
			},
		},
	})
	// Ignore send errors (user may have blocked the bot).
	_ = err
	return nil
}
```

- [ ] **Step 5.2: 在 verify.go 顶部 import 块加缺失的包**

`import` 已有 `context`、`fmt`、`strconv`、`strings`、`time`。  
还需要追加：

```go
"crypto/hmac"
"crypto/sha256"
"encoding/hex"
```

以及确认 `github.com/PaulSonOfLars/gotgbot/v2` 已经 import（已有）。

- [ ] **Step 5.3: 在 Options 结构体里加 TurnstileVerifySecret 字段**

打开 `internal/bot/types.go`，找到：

```go
type Options struct {
    DefaultLocale string
    MiniAppURL    string
    Features      Features
}
```

改为：

```go
type Options struct {
    DefaultLocale        string
    MiniAppURL           string
    Features             Features
    TurnstileVerifySecret string
}
```

- [ ] **Step 5.4: 在 cmd/bot/main.go 里把 TurnstileVerifySecret 传入 Options**

打开 `cmd/bot/main.go`，找到：

```go
app := botapp.New(botServices, botapp.Options{
    DefaultLocale: cfg.Bot.DefaultLocale,
    MiniAppURL:    cfg.Bot.MiniAppURL,
    Features:      botapp.NewFeatures(cfg.Bot.DisabledFeatures),
})
```

改为：

```go
app := botapp.New(botServices, botapp.Options{
    DefaultLocale:        cfg.Bot.DefaultLocale,
    MiniAppURL:           cfg.Bot.MiniAppURL,
    Features:             botapp.NewFeatures(cfg.Bot.DisabledFeatures),
    TurnstileVerifySecret: cfg.Turnstile.VerifySecret,
})
```

- [ ] **Step 5.5: 注册 chat_join_request handler**

找到 `registerVerifyHandlers`（位于 `internal/bot/verify.go` 第 19 行），在末尾追加：

```go
d.AddHandler(handlers.NewChatJoinRequest(chatjoinrequest.All, a.handleChatJoinRequest))
```

同时在 verify.go import 块加：

```go
"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/chatjoinrequest"
```

> **注意：** `handlers.NewChatJoinRequest` 和 `chatjoinrequest.All` 的包路径需与项目使用的 gotgbot 版本一致。  
> 如果该版本无此 filter 包，使用 `handlers.NewChatJoinRequest(nil, a.handleChatJoinRequest)` 即可。

- [ ] **Step 5.6: 验证 /set_verify type turnstile 的校验逻辑已放行**

打开 `internal/bot/verify.go`，找到 `handleSetVerify` 里的：

```go
if val != "button" && val != "captcha" && val != "multi_choice" && val != "poll" && val != "math" {
    return sendText(b, ctx, "验证类型只能是 button、captcha、multi_choice、poll 或 math。", nil)
}
```

改为：

```go
if val != "button" && val != "captcha" && val != "multi_choice" && val != "poll" && val != "math" && val != "turnstile" {
    return sendText(b, ctx, "验证类型只能是 button、captcha、multi_choice、poll、math 或 turnstile。", nil)
}
```

- [ ] **Step 5.7: 完整编译**

```powershell
go build ./...
```

Expected: 零错误。

- [ ] **Step 5.8: Commit**

```powershell
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" add internal/bot/verify.go internal/bot/types.go cmd/bot/main.go
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" commit -m "feat(bot): handle chat_join_request with Turnstile Mini App challenge"
```

---

## Task 6: Mini App 前端 — Verify.vue 页面

**Files:**
- Create: `web/src/mini/views/Verify.vue`
- Modify: `web/src/mini/router/index.ts`

- [ ] **Step 6.1: 创建 Verify.vue**

新建 `web/src/mini/views/Verify.vue`，完整内容：

```vue
<template>
  <div class="verify-page">
    <template v-if="state === 'loading'">
      <div class="status">正在初始化验证…</div>
    </template>

    <template v-else-if="state === 'invalid'">
      <div class="status error">{{ errorMsg }}</div>
    </template>

    <template v-else-if="state === 'pending'">
      <div class="card">
        <h2>入群验证</h2>
        <p>请完成下方人机验证以加入群组。</p>
        <div id="cf-turnstile-widget"></div>
        <p v-if="submitError" class="error-text">{{ submitError }}</p>
      </div>
    </template>

    <template v-else-if="state === 'success'">
      <div class="status success">
        <span class="icon">✅</span>
        <p>验证通过！入群申请已批准。</p>
        <p class="hint">可以关闭此页面，返回 Telegram 加入群组。</p>
      </div>
    </template>

    <template v-else-if="state === 'failed'">
      <div class="status error">
        <span class="icon">❌</span>
        <p>{{ errorMsg || '验证失败，请联系群管理员。' }}</p>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'

type State = 'loading' | 'invalid' | 'pending' | 'success' | 'failed'

const state = ref<State>('loading')
const errorMsg = ref('')
const submitError = ref('')

// Parse URL params from hash: /#/verify?chat=X&user=Y&sig=Z&exp=T
function parseParams() {
  const hash = window.location.hash // e.g. "#/verify?chat=..."
  const qIndex = hash.indexOf('?')
  if (qIndex === -1) return null
  const params = new URLSearchParams(hash.slice(qIndex + 1))
  const chatId = params.get('chat')
  const userId = params.get('user')
  const sig = params.get('sig')
  const exp = params.get('exp')
  if (!chatId || !userId || !sig || !exp) return null
  return { chatId: Number(chatId), userId: Number(userId), sig, exp: Number(exp) }
}

async function submitVerification(cfToken: string, p: ReturnType<typeof parseParams>) {
  if (!p) return
  submitError.value = ''
  try {
    const res = await fetch('/api/verify/turnstile', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        chat_id: p.chatId,
        user_id: p.userId,
        sig: p.sig,
        exp: p.exp,
        cf_token: cfToken,
      }),
    })
    const data = await res.json()
    if (data.ok) {
      state.value = 'success'
    } else {
      errorMsg.value = data.message || '验证失败'
      state.value = 'failed'
    }
  } catch {
    submitError.value = '网络错误，请稍后重试。'
  }
}

function loadTurnstileWidget(siteKey: string, p: ReturnType<typeof parseParams>) {
  const script = document.createElement('script')
  script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
  script.async = true
  script.defer = true
  script.onload = () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const w = (window as any)
    if (!w.turnstile) {
      state.value = 'failed'
      errorMsg.value = 'Turnstile 加载失败'
      return
    }
    w.turnstile.render('#cf-turnstile-widget', {
      sitekey: siteKey,
      callback: (token: string) => {
        submitVerification(token, p)
      },
      'error-callback': () => {
        state.value = 'failed'
        errorMsg.value = '验证组件出错，请刷新重试。'
      },
    })
  }
  document.head.appendChild(script)
}

onMounted(async () => {
  const p = parseParams()
  if (!p) {
    errorMsg.value = '无效的验证链接，请重新申请入群。'
    state.value = 'invalid'
    return
  }

  // Check expiry client-side as a quick guard (server also checks)
  if (Date.now() / 1000 > p.exp) {
    errorMsg.value = '验证链接已过期，请重新申请入群。'
    state.value = 'invalid'
    return
  }

  // Fetch the CF Turnstile site key from the API
  try {
    const res = await fetch('/api/verify/turnstile/config')
    if (res.ok) {
      const data = await res.json()
      state.value = 'pending'
      loadTurnstileWidget(data.site_key, p)
    } else {
      // Fallback: try reading from env at build time
      const siteKey = import.meta.env.VITE_TURNSTILE_SITE_KEY as string
      if (siteKey) {
        state.value = 'pending'
        loadTurnstileWidget(siteKey, p)
      } else {
        errorMsg.value = '验证服务暂不可用。'
        state.value = 'failed'
      }
    }
  } catch {
    const siteKey = import.meta.env.VITE_TURNSTILE_SITE_KEY as string
    if (siteKey) {
      state.value = 'pending'
      loadTurnstileWidget(siteKey, p)
    } else {
      errorMsg.value = '验证服务暂不可用。'
      state.value = 'failed'
    }
  }
})
</script>

<style scoped>
.verify-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: var(--tg-theme-bg-color, #fff);
  color: var(--tg-theme-text-color, #222);
  font-family: system-ui, sans-serif;
}

.card {
  text-align: center;
  max-width: 360px;
  width: 100%;
}

.card h2 {
  margin-bottom: 8px;
  font-size: 1.2rem;
}

.card p {
  color: var(--tg-theme-hint-color, #666);
  margin-bottom: 20px;
  font-size: 0.9rem;
}

#cf-turnstile-widget {
  display: flex;
  justify-content: center;
  min-height: 65px;
}

.status {
  text-align: center;
  font-size: 1rem;
}

.status .icon {
  font-size: 2.5rem;
  display: block;
  margin-bottom: 12px;
}

.status p {
  margin: 6px 0;
}

.status .hint {
  font-size: 0.8rem;
  color: var(--tg-theme-hint-color, #888);
}

.status.success { color: #2ab44a; }
.status.error   { color: #e53935; }
.error-text     { color: #e53935; font-size: 0.85rem; margin-top: 8px; }
</style>
```

> **说明：** 前端获取 site_key 的优先顺序：
> 1. 调 `/api/verify/turnstile/config`（Task 7 新增的公开端点，返回 `{site_key}`）
> 2. 回退到构建时环境变量 `VITE_TURNSTILE_SITE_KEY`

- [ ] **Step 6.2: 在 router/index.ts 加路由**

打开 `web/src/mini/router/index.ts`，在现有 import 后追加：

```ts
const Verify = () => import('@/mini/views/Verify.vue')
```

在 `routes` 数组末尾追加：

```ts
{
  path: '/verify',
  name: 'verify',
  component: Verify,
  meta: { title: '入群验证' },
},
```

- [ ] **Step 6.3: 在 .env.example 加前端 VITE 变量（备用）**

```env
# Mini App 前端构建时备用（优先由 /api/verify/turnstile/config 提供）
VITE_TURNSTILE_SITE_KEY=
```

- [ ] **Step 6.4: Commit**

```powershell
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" add web/src/mini/views/Verify.vue web/src/mini/router/index.ts .env.example
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" commit -m "feat(miniapp): add Turnstile verification page and route"
```

---

## Task 7: API — 新增 GET /api/verify/turnstile/config 端点（暴露 site_key）

前端需要 site_key 来渲染 Turnstile widget，但不应硬编码在前端构建产物里（方便线上换 key 无需重建）。加一个公开端点返回 site_key。

**Files:**
- Modify: `internal/api/verify_handler.go`
- Modify: `internal/api/router.go`

- [ ] **Step 7.1: 在 verify_handler.go 追加 handler**

在 `VerifyTurnstile` 函数之后追加：

```go
// TurnstileConfig returns the public CF Turnstile site key for the Mini App to use.
// This endpoint is intentionally public — site keys are not secret.
func (s *Server) TurnstileConfig(c *gin.Context) {
    siteKey := s.deps.TurnstileSiteKey
    if siteKey == "" {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "turnstile not configured"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"site_key": siteKey})
}
```

- [ ] **Step 7.2: 在 Dependencies 加 TurnstileSiteKey 字段**

打开 `internal/api/handlers.go`，在上一步加的字段旁追加：

```go
TurnstileSiteKey string
```

并在初始化 Dependencies 的地方（`cmd/api/main.go`）传入：

```go
TurnstileSiteKey: cfg.Turnstile.SiteKey,
```

- [ ] **Step 7.3: 在 router.go 注册**

在 `r.POST("/api/verify/turnstile", server.VerifyTurnstile)` 旁边加：

```go
r.GET("/api/verify/turnstile/config", server.TurnstileConfig)
```

- [ ] **Step 7.4: 完整编译**

```powershell
go build ./...
```

Expected: 零错误。

- [ ] **Step 7.5: Commit**

```powershell
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" add internal/api/verify_handler.go internal/api/handlers.go internal/api/router.go cmd/api/main.go
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" commit -m "feat(api): expose GET /api/verify/turnstile/config for Mini App site_key"
```

---

## Task 8: 单元测试 — HMAC 工具函数

**Files:**
- Create: `internal/api/verify_handler_test.go`

- [ ] **Step 8.1: 创建测试文件**

```go
package api

import (
	"testing"
)

func TestTurnstileHMAC_Deterministic(t *testing.T) {
	sig1 := turnstileHMAC(123, 456, 9999999, "secret")
	sig2 := turnstileHMAC(123, 456, 9999999, "secret")
	if sig1 != sig2 {
		t.Fatalf("HMAC not deterministic: %s vs %s", sig1, sig2)
	}
	if len(sig1) != 64 { // SHA-256 hex = 64 chars
		t.Fatalf("expected 64-char hex, got %d: %s", len(sig1), sig1)
	}
}

func TestTurnstileHMAC_DifferentInputs(t *testing.T) {
	sig1 := turnstileHMAC(123, 456, 9999999, "secret")
	sig2 := turnstileHMAC(999, 456, 9999999, "secret")
	sig3 := turnstileHMAC(123, 456, 9999999, "other-secret")
	if sig1 == sig2 {
		t.Fatal("different chatID should produce different HMAC")
	}
	if sig1 == sig3 {
		t.Fatal("different secret should produce different HMAC")
	}
}
```

- [ ] **Step 8.2: 运行测试**

```powershell
cd "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot"
go test ./internal/api/... -run TestTurnstileHMAC -v
```

Expected:
```
--- PASS: TestTurnstileHMAC_Deterministic
--- PASS: TestTurnstileHMAC_DifferentInputs
PASS
```

- [ ] **Step 8.3: Commit**

```powershell
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" add internal/api/verify_handler_test.go
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" commit -m "test(api): HMAC determinism tests for Turnstile verify"
```

---

## Task 9: 文档更新 + 最终推送

**Files:**
- Modify: `README.md`
- Modify: `README.en.md`
- Modify: `README.zh-CN.md`

- [ ] **Step 9.1: README.md — 功能总览表格**

找到 `| 群组管理 | ...` 行，在"/set_verify 验证开关"描述里追加 `turnstile`：

```
| 群规管理 / 验证 | ... verify_toggle / set_verify（支持 button/captcha/multi_choice/poll/math/**turnstile**） |
```

在"入群验证"节的 `type` 选项说明里加一行：

```
- `turnstile` — Cloudflare Turnstile 人机验证，通过 Telegram Mini App 完成
```

- [ ] **Step 9.2: README.en.md — 验证类型说明**

在 `/set_verify` 介绍段落里加：

```
  - `turnstile` — Cloudflare Turnstile via Telegram Mini App (requires `SOLA_TURNSTILE_*` config)
```

- [ ] **Step 9.3: README.zh-CN.md — 路线图勾选**

在路线图末尾加：

```markdown
- ✅ Cloudflare Turnstile + Mini App 入群验证（/set_verify type turnstile）
```

- [ ] **Step 9.4: 最终 build + test**

```powershell
cd "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot"
go build ./...
go test ./...
```

Expected: 零错误，所有测试通过。

- [ ] **Step 9.5: Commit + Push**

```powershell
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" add README.md README.en.md README.zh-CN.md
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" commit -m "docs: document Turnstile Mini App verify type across all READMEs"
git -C "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot" push origin main
```

---

## 自检清单

### Spec coverage

| 需求 | 对应 Task |
|------|----------|
| Config 字段 SiteKey/SecretKey/VerifySecret | Task 1 |
| API TurnstileVerifyRequest/Response | Task 2 |
| POST /api/verify/turnstile 端点（HMAC + CF verify + Telegram approve） | Task 3-4 |
| Dependencies 传入 BotToken + Turnstile keys | Task 4 |
| Bot handleChatJoinRequest + WebApp 私信 | Task 5 |
| VerifyType "turnstile" 校验放行 | Task 5.6 |
| TurnstileVerifySecret 传入 Options | Task 5.3-5.4 |
| Mini App Verify.vue 页面 | Task 6 |
| /verify 路由 | Task 6.2 |
| GET /api/verify/turnstile/config（site_key） | Task 7 |
| HMAC 单元测试 | Task 8 |
| README 三份更新 | Task 9 |
| .env.example | Task 1.4, Task 6.3 |

### 潜在注意点

1. **gotgbot 版本兼容**：`handlers.NewChatJoinRequest` 和 `chatjoinrequest.All` 的包路径需在 go.mod 的 gotgbot 版本里存在；若不存在，参考其他 handler 注册方式（可用 `handlers.NewChatJoinRequest(nil, ...)` 或自定义 filter）。

2. **群组设置**：用户侧需要在 BotFather 那边启用群组的「入群申请」审核（Telegram 设置 → 群组 → Approve New Members），才能产生 `chat_join_request` 更新。

3. **Mini App HTTPS**：CF Turnstile widget 要求页面在 HTTPS 下加载，Mini App URL 必须是 HTTPS。

4. **WebApp 按钮限制**：`InlineKeyboardButton.WebApp` 只在群组私信中可用（不能在群内消息里），这与我们的流程（发私信）一致。

5. **`bot_token` 在 API 服务里**：目前 bot token 只在 cmd/bot 里使用，API 服务用它调 Telegram 需要确认 cfg.Bot.Token 可从 API 进程的 config 读到（是的，两个进程读同一份 config）。
