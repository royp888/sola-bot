package bot

import (
	"context"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
)

// registerForwardHandlers 注册实时自动转发的入口。
// 必须放在“万能消息处理器”之前，但返回 nil 让后续处理器继续执行。
func (a *App) registerForwardHandlers(d *ext.Dispatcher) {
	if a.services.Forwards == nil {
		return
	}
	d.AddHandler(handlers.NewMessage(message.All, a.handleForwardRealtime))
}

func (a *App) handleForwardRealtime(b *gotgbot.Bot, ctx *ext.Context) error {
	engine := a.services.Forwards
	if engine == nil {
		return nil
	}
	msg := ctx.EffectiveMessage
	if msg == nil || msg.Chat.Id == 0 {
		return nil
	}
	// 转发可能触发 FloodWait，绝不能阻塞更新循环
	go func() {
		timeout := 3 * time.Minute
		runCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := engine.HandleRealtime(runCtx, b, msg); err != nil {
			_ = err
		}
	}()
	return nil
}
