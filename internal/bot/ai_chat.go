package bot

import (
	"errors"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
)

// registerAIChatHandlers registers the AI chat commands (/ai, /gpt).
func (a *App) registerAIChatHandlers(d *ext.Dispatcher) {
	d.AddHandler(handlers.NewCommand("ai", a.wrap(a.handleAIChat, a.RateLimit("cmd:ai", 1))))
	d.AddHandler(handlers.NewCommand("gpt", a.wrap(a.handleAIChat, a.RateLimit("cmd:gpt", 1))))
}

// handleAIChat answers a slash-command AI question (/ai ... or /gpt ...).
// It works in private chats, groups and supergroups.
func (a *App) handleAIChat(b *gotgbot.Bot, ctx *ext.Context) error {
	prompt := strings.TrimSpace(strings.Join(commandArgs(ctx), " "))
	if prompt == "" {
		return sendText(b, ctx, "用法：/ai <你的问题>\n例如：/ai 帮我写一段群欢迎语", nil)
	}

	return a.answerAI(b, ctx, prompt)
}

// handleAIChatNatural handles a natural-language AI question prefixed with
// "ai " or "AI " inside groups (e.g. "ai 帮我写一段欢迎语").
func (a *App) handleAIChatNatural(b *gotgbot.Bot, ctx *ext.Context, raw string) error {
	fields := strings.Fields(raw)
	if len(fields) < 2 {
		return sendText(b, ctx, "用法：ai <你的问题>\n例如：ai 帮我写一段群欢迎语", nil)
	}

	prompt := strings.TrimSpace(strings.Join(fields[1:], " "))
	return a.answerAI(b, ctx, prompt)
}

// answerAI sends the prompt to the configured LLM and returns its reply.
func (a *App) answerAI(b *gotgbot.Bot, ctx *ext.Context, prompt string) error {
	if a.services.AiFilter == nil {
		return sendText(b, ctx, "⚠️ AI 对话未启用：请在管理后台「AI 设置」中配置模型与密钥。", nil)
	}

	scope := requestScope(ctx)

	// Show "typing" while waiting for the LLM response.
	if scope.Chat.ID != 0 {
		_, _ = b.SendChatAction(scope.Chat.ID, "typing", nil)
	}

	reply, err := a.services.AiFilter.Chat(scope.Context, prompt)
	if err != nil {
		if errors.Is(err, ErrAINotConfigured) {
			return sendText(b, ctx, "⚠️ AI 对话未启用：请在管理后台「AI 设置」中配置模型与密钥。", nil)
		}
		return sendText(b, ctx, "❌ AI 调用失败："+err.Error(), nil)
	}

	return sendText(b, ctx, reply, nil)
}
