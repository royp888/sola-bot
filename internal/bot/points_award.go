package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func (a *App) handleSign(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.awardSign(b, ctx)
}

func (a *App) handleMessagePoints(b *gotgbot.Bot, ctx *ext.Context) error {
	if handled, err := a.handleConversationMessage(b, ctx); handled || err != nil {
		return err
	}
	if ctx == nil || ctx.Message == nil {
		return nil
	}
	if moderationBlockedPoints(ctx) {
		return nil
	}

	msg := ctx.Message
	if msg.Chat.Type != "group" && msg.Chat.Type != "supergroup" {
		return nil
	}
	if msg.From == nil {
		return nil
	}
	if msg.From.IsBot {
		return nil
	}

	if handled, err := a.handleKeywordLottery(b, ctx); err != nil {
		log.Printf("keyword lottery error: %v", err)
	} else if handled {
		return nil
	}

	if err := a.handleAutoReply(b, ctx); err != nil {
		log.Printf("auto reply error: %v", err)
	}

	if a.services.Points == nil {
		return nil
	}

	switch normalizedMessageText(msg) {
	case "积分", "我的积分", "查询积分":
		return a.handlePoints(b, ctx)
	case "排行", "排行榜", "积分排行":
		return a.showPointsRankPeriod(b, ctx, "all")
	case "今日排行", "日排行":
		return a.showPointsRankPeriod(b, ctx, "day")
	case "本周排行", "周排行":
		return a.showPointsRankPeriod(b, ctx, "week")
	case "本月排行", "月排行":
		return a.showPointsRankPeriod(b, ctx, "month")
	case "签到":
		return a.awardSign(b, ctx)
	}

	messageType := pointMessageType(msg)
	if messageType == "" {
		return nil
	}
	log.Printf("telegram group message received: chat=%d user=%d type=%s command=%t", msg.Chat.Id, msg.From.Id, messageType, isCommandMessage(msg))

	_, err := a.services.Points.AwardMessage(requestScope(ctx).Context, PointAwardRequest{
		ChatID:      msg.Chat.Id,
		UserID:      msg.From.Id,
		MessageID:   msg.MessageId,
		MessageType: messageType,
		Username:    msg.From.Username,
		DisplayName: displayName(msg.From),
		ChatType:    msg.Chat.Type,
		ChatTitle:   msg.Chat.Title,
		IsForwarded: msg.ForwardOrigin != nil || msg.IsAutomaticForward,
		IsCommand:   isCommandMessage(msg),
		FromBot:     msg.From.IsBot,
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *App) handleKeywordLottery(b *gotgbot.Bot, ctx *ext.Context) (bool, error) {
	if a.services.Lottery == nil || ctx == nil || ctx.Message == nil {
		return false, nil
	}
	msg := ctx.Message
	if msg.Text == "" || msg.From == nil || msg.From.IsBot {
		return false, nil
	}
	scope := requestScope(ctx)
	handled, message, lotteryID, err := a.services.Lottery.JoinByKeyword(scope.Context, scope.Chat.ID, msg.Text, scope.Actor.ID, msg.From.Username)
	if !handled {
		return false, err
	}

	_, _ = b.DeleteMessageWithContext(scope.Context, scope.Chat.ID, msg.MessageId, nil)

	reply := keywordLotteryReply(msg.From, message, err)
	sent, sendErr := b.SendMessageWithContext(scope.Context, scope.Chat.ID, reply, nil)
	if sendErr == nil && sent != nil {
		go deleteMessageLater(b, scope.Chat.ID, sent.MessageId, 5*time.Second)
	}
	if err != nil {
		return true, nil
	}
	log.Printf("keyword lottery joined: chat=%d user=%d lottery=%d", scope.Chat.ID, scope.Actor.ID, lotteryID)
	if lotteryWasAutoDrawn(message) {
		_ = a.announceLotteryResult(b, ctx, lotteryID)
	}
	return true, nil
}

func keywordLotteryReply(user *gotgbot.User, message string, err error) string {
	name := usernameOrID(user)
	if err == nil {
		if strings.TrimSpace(message) == "" {
			message = "已参与抽奖。"
		}
		return fmt.Sprintf("@%s %s", name, message)
	}
	errText := err.Error()
	switch {
	case strings.Contains(errText, "已经参与"):
		return fmt.Sprintf("@%s 你已参与本次抽奖。", name)
	case strings.Contains(errText, "积分不足"):
		return fmt.Sprintf("@%s 积分不足，无法参与。", name)
	case strings.Contains(errText, "名额已满"):
		return fmt.Sprintf("@%s 抽奖名额已满。", name)
	case strings.Contains(errText, "结束") || strings.Contains(errText, "取消"):
		return fmt.Sprintf("@%s 抽奖已结束或已取消。", name)
	default:
		return fmt.Sprintf("@%s 参与失败：%s", name, errText)
	}
}

func usernameOrID(user *gotgbot.User) string {
	if user == nil {
		return "用户"
	}
	if strings.TrimSpace(user.Username) != "" {
		return strings.TrimSpace(user.Username)
	}
	return strconv.FormatInt(user.Id, 10)
}

func (a *App) awardSign(b *gotgbot.Bot, ctx *ext.Context) error {
	if a.services.Points == nil || ctx == nil || ctx.Message == nil {
		return sendText(b, ctx, "积分服务尚未接入。", nil)
	}
	msg := ctx.Message
	if msg.Chat.Type != "group" && msg.Chat.Type != "supergroup" {
		return sendText(b, ctx, "请在群里签到。", nil)
	}
	if msg.From == nil {
		return nil
	}
	result, err := a.services.Points.AwardMessage(requestScope(ctx).Context, PointAwardRequest{
		ChatID:        msg.Chat.Id,
		UserID:        msg.From.Id,
		MessageID:     msg.MessageId,
		MessageType:   "text",
		CooldownScope: "sign",
		ReasonPrefix:  "sign",
		Username:      msg.From.Username,
		DisplayName:   displayName(msg.From),
		ChatType:      msg.Chat.Type,
		ChatTitle:     msg.Chat.Title,
		IsCommand:     false,
		FromBot:       msg.From.IsBot,
	})
	if err != nil {
		return err
	}
	return a.replySignResult(b, ctx, result)
}

func (a *App) replySignResult(b *gotgbot.Bot, ctx *ext.Context, result PointAwardResult) error {
	scope := requestScope(ctx)
	if result.Awarded {
		summary, err := a.services.Points.GetSummary(scope.Context, scope.Chat.ID, scope.Actor.ID)
		if err != nil {
			return err
		}
		return sendText(b, ctx, fmt.Sprintf("签到成功，+%d 积分。\n%s", result.Points, summary), nil)
	}
	if result.Reason == "cooldown" {
		return sendText(b, ctx, "签到太频繁了，稍后再试。", nil)
	}
	if result.Reason == "disabled" {
		return sendText(b, ctx, "本群积分系统已关闭。", nil)
	}
	return sendText(b, ctx, "签到未计分："+result.Reason, nil)
}

func pointMessageType(msg *gotgbot.Message) string {
	switch {
	case msg.Text != "":
		return "text"
	case len(msg.Photo) > 0:
		return "photo"
	case msg.Sticker != nil:
		return "sticker"
	case msg.Video != nil || msg.VideoNote != nil || msg.Animation != nil:
		return "video"
	case msg.Document != nil || msg.Audio != nil:
		return "file"
	case msg.Voice != nil:
		return "voice"
	default:
		return ""
	}
}

func isCommandMessage(msg *gotgbot.Message) bool {
	entities := msg.GetEntities()
	return len(entities) > 0 && entities[0].Type == "bot_command" && entities[0].Offset == 0
}

func normalizedMessageText(msg *gotgbot.Message) string {
	if msg == nil {
		return ""
	}
	text := strings.TrimSpace(msg.Text)
	text = strings.TrimPrefix(text, "/")
	return strings.TrimSpace(text)
}

func displayName(user *gotgbot.User) string {
	if user == nil {
		return ""
	}
	if user.FirstName != "" && user.LastName != "" {
		return user.FirstName + " " + user.LastName
	}
	if user.FirstName != "" {
		return user.FirstName
	}
	return user.Username
}
