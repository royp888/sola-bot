package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
)

func (a *App) registerAutoReplyHandlers(d *ext.Dispatcher) {
	d.AddHandler(handlers.NewCommand("add_reply", a.wrap(a.handleAddReply, a.RateLimit("cmd:add_reply", 1))))
	d.AddHandler(handlers.NewCommand("del_reply", a.wrap(a.handleDelReply, a.RateLimit("cmd:del_reply", 1))))
	d.AddHandler(handlers.NewCommand("replies", a.wrap(a.handleListReplies, a.RateLimit("cmd:replies", 1))))
}

func (a *App) handleAutoReply(b *gotgbot.Bot, ctx *ext.Context) error {
	if a.services.AutoReply == nil || ctx == nil || ctx.Message == nil {
		return nil
	}
	msg := ctx.Message
	if msg.Chat.Type != "group" && msg.Chat.Type != "supergroup" {
		return nil
	}
	if msg.From == nil || msg.From.IsBot || strings.TrimSpace(msg.Text) == "" {
		return nil
	}

	scope := requestScope(ctx)
	matches, err := a.services.AutoReply.MatchAll(scope.Context, msg.Chat.Id, msg.Text)
	if err != nil {
		log.Printf("auto reply match error: %v", err)
		return nil
	}
	for _, match := range matches {
		if strings.TrimSpace(match.ReplyText) == "" {
			continue
		}
		deleteTrigger := strings.HasPrefix(match.Keyword, "~")
		if deleteTrigger {
			_, _ = b.DeleteMessageWithContext(scope.Context, msg.Chat.Id, msg.MessageId, nil)
		}
		var sendOpts *gotgbot.SendMessageOpts
		if !deleteTrigger {
			sendOpts = &gotgbot.SendMessageOpts{
				ReplyParameters: &gotgbot.ReplyParameters{MessageId: msg.MessageId},
			}
		}
		if _, sendErr := b.SendMessageWithContext(scope.Context, msg.Chat.Id, match.ReplyText, sendOpts); sendErr != nil {
			log.Printf("auto reply send error: %v", sendErr)
		}
	}
	return nil
}

func formatAutoReplyPanel(replies []AutoReplyRecord) string {
	var builder strings.Builder
	builder.WriteString("自动回复面板\n\n")
	builder.WriteString("适合做问候、关键词触发、常见问题快捷答复。\n")
	if len(replies) == 0 {
		builder.WriteString("当前还没有配置自动回复。\n")
	} else {
		builder.WriteString("当前规则：")
		limit := len(replies)
		if limit > 8 {
			limit = 8
		}
		for i := 0; i < limit; i++ {
			item := replies[i]
			status := "关闭"
			if item.Enabled {
				status = "启用"
			}
			builder.WriteString(fmt.Sprintf("\n%d. [%s] %s -> %s", i+1, status, item.Keyword, truncateRunes(item.ReplyText, 36)))
		}
		if len(replies) > limit {
			builder.WriteString(fmt.Sprintf("\n... 其余 %d 条请在后台查看", len(replies)-limit))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("\n快捷命令\n/add_reply 关键词 | 回复内容\n/del_reply 关键词")
	return builder.String()
}

func autoReplyMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "刷新列表", CallbackData: CallbackData("admin", "autoreplies")},
			{Text: "消息模板", CallbackData: CallbackData("admin", "templates")},
		},
		{
			{Text: "关键词过滤", CallbackData: CallbackData("admin", "keywords")},
			{Text: "返回群管", CallbackData: CallbackData("admin", "moderation")},
		},
		{
			{Text: "返回群组", CallbackData: CallbackData("menu", "groups")},
		},
	}}}
}

func (a *App) showAutoReplyPanel(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.AutoReply == nil {
		return respondText(b, ctx, "自动回复服务尚未接入。", moderationMenuMarkup())
	}
	replies, err := a.services.AutoReply.ListForChat(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	return respondText(b, ctx, formatAutoReplyPanel(replies), autoReplyMarkup())
}

func (a *App) handleAddReply(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.AutoReply == nil {
		return sendText(b, ctx, "自动回复服务尚未接入。", nil)
	}
	raw := strings.TrimSpace(strings.Join(commandArgs(ctx), " "))
	keyword, reply, ok := parseAutoReplyCommand(raw)
	if !ok {
		return sendText(b, ctx, "用法：/add_reply 关键词 | 回复内容", nil)
	}
	_, err := a.services.AutoReply.CreateForBot(scope.Context, AutoReplyCreate{
		ChatID:    scope.Chat.ID,
		Keyword:   keyword,
		MatchType: "contains",
		ReplyText: reply,
		Enabled:   true,
		CreatedBy: scope.Actor.ID,
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") ||
			strings.Contains(strings.ToLower(err.Error()), "unique") {
			return sendText(b, ctx, fmt.Sprintf("自动回复「%s」已存在，请先删除再添加。", keyword), nil)
		}
		return err
	}
	return respondText(b, ctx, fmt.Sprintf("已添加自动回复：%s\n\n你可以继续发送 /add_reply 批量补充，或点按钮返回面板。", keyword), autoReplyMarkup())
}

func (a *App) handleDelReply(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.AutoReply == nil {
		return sendText(b, ctx, "自动回复服务尚未接入。", nil)
	}
	keyword := strings.TrimSpace(strings.Join(commandArgs(ctx), " "))
	if keyword == "" {
		return sendText(b, ctx, "用法：/del_reply 关键词", nil)
	}
	if err := a.services.AutoReply.DeleteByKeyword(scope.Context, scope.Chat.ID, keyword); err != nil {
		return err
	}
	return respondText(b, ctx, fmt.Sprintf("已删除自动回复：%s", keyword), autoReplyMarkup())
}

func (a *App) handleListReplies(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.AutoReply == nil {
		return sendText(b, ctx, "自动回复服务尚未接入。", nil)
	}
	replies, err := a.services.AutoReply.ListForChat(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	return respondText(b, ctx, formatAutoReplyPanel(replies), autoReplyMarkup())
}

func parseAutoReplyCommand(raw string) (string, string, bool) {
	parts := strings.SplitN(raw, "|", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	keyword := strings.TrimSpace(parts[0])
	reply := strings.TrimSpace(parts[1])
	return keyword, reply, keyword != "" && reply != ""
}
