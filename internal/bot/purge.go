package bot

import (
	"fmt"
	"strconv"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

const purgeMaxMessages = 500

func (a *App) handlePurge(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if ctx.Message == nil {
		return nil
	}

	var fromMsgID int64
	if ctx.Message.ReplyToMessage != nil {
		fromMsgID = ctx.Message.ReplyToMessage.MessageId
	} else {
		args := commandArgs(ctx)
		if len(args) == 0 {
			return sendText(b, ctx, "用法：回复某条消息 /purge，或 /purge <数量>（最多500）。", nil)
		}
		n, err := strconv.Atoi(args[0])
		if err != nil || n <= 0 {
			return sendText(b, ctx, "数量必须是大于 0 的整数。", nil)
		}
		if n > purgeMaxMessages {
			n = purgeMaxMessages
		}
		fromMsgID = ctx.Message.MessageId - int64(n)
	}

	endMsgID := ctx.Message.MessageId
	if endMsgID-fromMsgID > purgeMaxMessages {
		fromMsgID = endMsgID - purgeMaxMessages
	}
	if fromMsgID >= endMsgID {
		return sendText(b, ctx, "没有需要删除的消息。", nil)
	}

	ids := make([]int64, 0, endMsgID-fromMsgID)
	for id := fromMsgID + 1; id <= endMsgID; id++ {
		ids = append(ids, id)
	}
	const batchSize = 100
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		_, _ = b.DeleteMessagesWithContext(scope.Context, scope.Chat.ID, ids[i:end], nil)
	}

	notice, _ := b.SendMessageWithContext(scope.Context, scope.Chat.ID,
		fmt.Sprintf("已清除 %d 条消息。", len(ids)), nil)
	if notice != nil {
		time.AfterFunc(5*time.Second, func() {
			_, _ = b.DeleteMessageWithContext(scope.Context, scope.Chat.ID, notice.MessageId, nil)
		})
	}
	return nil
}

func (a *App) handleDel(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if ctx.Message == nil {
		return nil
	}
	if ctx.Message.ReplyToMessage == nil {
		return sendText(b, ctx, "用法：回复目标消息 /del。", nil)
	}
	targetID := ctx.Message.ReplyToMessage.MessageId
	cmdID := ctx.Message.MessageId
	_, _ = b.DeleteMessageWithContext(scope.Context, scope.Chat.ID, targetID, nil)
	_, _ = b.DeleteMessageWithContext(scope.Context, scope.Chat.ID, cmdID, nil)
	return nil
}
