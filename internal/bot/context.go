package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

const telegramRequestTimeout = 15 * time.Second

func telegramRequestContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, telegramRequestTimeout)
}

func requestScope(ctx *ext.Context) RequestScope {
	scope := RequestScope{Context: context.Background()}
	if ctx == nil {
		return scope
	}
	if ctx.EffectiveUser != nil {
		scope.Actor = Actor{
			ID:           ctx.EffectiveUser.Id,
			Username:     ctx.EffectiveUser.Username,
			FirstName:    ctx.EffectiveUser.FirstName,
			IsBot:        ctx.EffectiveUser.IsBot,
			LanguageCode: ctx.EffectiveUser.LanguageCode,
		}
	}
	if ctx.EffectiveChat != nil {
		scope.Chat = ChatRef{
			ID:       ctx.EffectiveChat.Id,
			Type:     ctx.EffectiveChat.Type,
			Title:    ctx.EffectiveChat.Title,
			Username: ctx.EffectiveChat.Username,
		}
	}
	if ctx.CallbackQuery != nil {
		scope.CallbackData = ctx.CallbackQuery.Data
	}
	return scope
}

func sendText(b *gotgbot.Bot, ctx *ext.Context, text string, opts *gotgbot.SendMessageOpts) error {
	if ctx == nil {
		return fmt.Errorf("telegram context is nil")
	}
	if ctx.EffectiveChat != nil {
		reqCtx, cancel := telegramRequestContext(requestScope(ctx).Context)
		defer cancel()

		start := time.Now()
		log.Printf("telegram send message start: chat=%d chars=%d", ctx.EffectiveChat.Id, len([]rune(text)))
		_, err := b.SendMessageWithContext(reqCtx, ctx.EffectiveChat.Id, text, opts)
		if err != nil {
			log.Printf("telegram send message failed: chat=%d duration=%s error=%v", ctx.EffectiveChat.Id, time.Since(start).Round(time.Millisecond), err)
			return err
		}
		log.Printf("telegram send message done: chat=%d duration=%s", ctx.EffectiveChat.Id, time.Since(start).Round(time.Millisecond))
		return err
	}
	return fmt.Errorf("no effective message or chat available")
}

func respondText(b *gotgbot.Bot, ctx *ext.Context, text string, opts *gotgbot.SendMessageOpts) error {
	if ctx == nil || ctx.CallbackQuery == nil || ctx.EffectiveChat == nil || ctx.EffectiveMessage == nil {
		return sendText(b, ctx, text, opts)
	}
	reqCtx, cancel := telegramRequestContext(requestScope(ctx).Context)
	defer cancel()

	editOpts := &gotgbot.EditMessageTextOpts{
		ChatId:    ctx.EffectiveChat.Id,
		MessageId: ctx.EffectiveMessage.MessageId,
	}
	if opts != nil {
		editOpts.ParseMode = opts.ParseMode
		if markup, ok := opts.ReplyMarkup.(gotgbot.InlineKeyboardMarkup); ok {
			editOpts.ReplyMarkup = markup
		}
	}
	if _, _, err := b.EditMessageTextWithContext(reqCtx, text, editOpts); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "message is not modified") {
			return nil
		}
		log.Printf("telegram edit message failed: chat=%d message=%d error=%v", ctx.EffectiveChat.Id, ctx.EffectiveMessage.MessageId, err)
		return sendText(b, ctx, text, opts)
	}
	return nil
}

func answerCallback(b *gotgbot.Bot, ctx *ext.Context, text string) error {
	if ctx == nil || ctx.CallbackQuery == nil {
		return nil
	}
	reqCtx, cancel := telegramRequestContext(requestScope(ctx).Context)
	defer cancel()
	_, err := b.AnswerCallbackQueryWithContext(reqCtx, ctx.CallbackQuery.Id, &gotgbot.AnswerCallbackQueryOpts{Text: text})
	return err
}
