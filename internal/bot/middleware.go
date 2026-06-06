package bot

import (
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

type Middleware func(next HandlerFunc) HandlerFunc

func (a *App) wrap(handler HandlerFunc, middleware ...Middleware) HandlerFunc {
	wrapped := handler
	for i := len(middleware) - 1; i >= 0; i-- {
		wrapped = middleware[i](wrapped)
	}
	return func(b *gotgbot.Bot, ctx *ext.Context) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in handler: %v\nstack trace:\n%s", r, debug.Stack())
				err = fmt.Errorf("panic: %v", r)
			}
		}()
		logInteraction(ctx)
		start := time.Now()
		err = wrapped(b, ctx)
		if err != nil {
			log.Printf("telegram handler failed: duration=%s error=%v", time.Since(start).Round(time.Millisecond), err)
			return err
		}
		log.Printf("telegram handler done: duration=%s", time.Since(start).Round(time.Millisecond))
		return nil
	}
}

func logInteraction(ctx *ext.Context) {
	if ctx == nil {
		return
	}
	chatID := int64(0)
	userID := int64(0)
	chatType := ""
	if ctx.EffectiveChat != nil {
		chatID = ctx.EffectiveChat.Id
		chatType = ctx.EffectiveChat.Type
	}
	if ctx.EffectiveUser != nil {
		userID = ctx.EffectiveUser.Id
	}
	if ctx.CallbackQuery != nil {
		log.Printf("telegram callback received: chat=%d type=%s user=%d data=%q", chatID, chatType, userID, ctx.CallbackQuery.Data)
		return
	}
	if ctx.Message != nil {
		log.Printf("telegram command received: chat=%d type=%s user=%d text=%q", chatID, chatType, userID, ctx.Message.Text)
	}
}

func (a *App) RequireAdmin() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(b *gotgbot.Bot, ctx *ext.Context) error {
			scope := requestScope(ctx)
			if a.services.Access == nil || scope.Chat.ID == 0 || scope.Actor.ID == 0 {
				return next(b, ctx)
			}
			ok, err := a.services.Access.IsAdmin(scope.Context, scope.Chat.ID, scope.Actor.ID)
			if err != nil {
				return err
			}
			if !ok {
				ok, err = a.isTelegramManager(b, ctx)
				if err != nil {
					return err
				}
			}
			if !ok {
				return sendText(b, ctx, "需要频道/群管理员权限。", nil)
			}
			return next(b, ctx)
		}
	}
}

func (a *App) RequirePermission(permission string) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(b *gotgbot.Bot, ctx *ext.Context) error {
			scope := requestScope(ctx)
			if a.services.Access == nil || scope.Chat.ID == 0 || scope.Actor.ID == 0 {
				return next(b, ctx)
			}
			ok, err := a.services.Access.HasPermission(scope.Context, scope.Chat.ID, scope.Actor.ID, permission)
			if err != nil {
				return err
			}
			if !ok {
				ok, err = a.isTelegramManager(b, ctx)
				if err != nil {
					return err
				}
			}
			if !ok {
				return sendText(b, ctx, fmt.Sprintf("缺少 %s 权限。", permission), nil)
			}
			return next(b, ctx)
		}
	}
}

func (a *App) RateLimit(bucket string, cost int) Middleware {
	if cost <= 0 {
		cost = 1
	}
	return func(next HandlerFunc) HandlerFunc {
		return func(b *gotgbot.Bot, ctx *ext.Context) error {
			scope := requestScope(ctx)
			if a.services.RateLimit == nil || scope.Actor.ID == 0 {
				return next(b, ctx)
			}
			key := fmt.Sprintf("%s:%d:%d", bucket, scope.Chat.ID, scope.Actor.ID)
			allowed, retryAfter, err := a.services.RateLimit.Allow(scope.Context, key, cost)
			if err != nil {
				return err
			}
			if !allowed {
				if retryAfter <= 0 {
					retryAfter = time.Second
				}
				return sendText(b, ctx, fmt.Sprintf("操作太频繁，请 %s 后再试。", retryAfter.Round(time.Second)), nil)
			}
			return next(b, ctx)
		}
	}
}
