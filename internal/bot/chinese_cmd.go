package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// handleChineseCommand intercepts natural-language Chinese commands before
// they reach moderation/points handlers. Matched commands receive the same
// rate-limit and permission treatment as their slash-command equivalents via
// the registered middleware chain.
func (a *App) handleChineseCommand(b *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.Message
	if msg == nil || msg.Text == "" || msg.From == nil || msg.From.IsBot {
		return ext.ContinueGroups
	}
	if msg.Chat.Type != "group" && msg.Chat.Type != "supergroup" {
		return ext.ContinueGroups
	}

	text := strings.TrimSpace(msg.Text)

	// --- exact matches ---
	switch text {
	case "签到", "打卡":
		return a.handleSign(b, ctx)
	case "积分", "我的积分", "查询积分":
		return a.handlePoints(b, ctx)
	case "排行榜", "积分榜", "排名":
		// default to all-time rank
		return a.showChineseRank(b, ctx, "all")
	case "今日榜":
		return a.showChineseRank(b, ctx, "day")
	case "本周榜":
		return a.showChineseRank(b, ctx, "week")
	case "抽奖", "抽奖列表", "进行中的抽奖":
		return a.handleLottery(b, ctx)
	}

	// --- prefix matches with arguments ---
	if strings.HasPrefix(text, "加积分") {
		return a.handleChinesePointsAdjust(b, ctx, text, 1)
	}
	if strings.HasPrefix(text, "扣积分") || strings.HasPrefix(text, "减积分") {
		return a.handleChinesePointsAdjust(b, ctx, text, -1)
	}

	return ext.ContinueGroups
}

func (a *App) showChineseRank(b *gotgbot.Bot, ctx *ext.Context, period string) error {
	scope := requestScope(ctx)
	if a.services.Points == nil {
		return sendText(b, ctx, "积分排行服务尚未接入。", nil)
	}
	rank, err := a.services.Points.GetRank(scope.Context, scope.Chat.ID, period, 10)
	if err != nil {
		return err
	}
	return sendText(b, ctx, rank, nil)
}

func (a *App) handleChinesePointsAdjust(b *gotgbot.Bot, ctx *ext.Context, raw string, sign int) error {
	scope := requestScope(ctx)
	if a.services.Points == nil {
		return sendText(b, ctx, "积分服务尚未接入。", nil)
	}

	// Strip the prefix: "加积分 " or "扣积分 " or "减积分 "
	args := strings.Fields(raw)
	if len(args) < 2 {
		return sendText(b, ctx, "用法示例：加积分 10 或 加积分 @用户名 10", nil)
	}

	args = args[1:] // drop prefix

	// Try to find a user mention or user ID at the front
	targetID := scope.Actor.ID // default: self
	if len(args) >= 2 {
		// Check if first arg looks like an @mention or numeric ID
		first := args[0]
		if strings.HasPrefix(first, "@") {
			// username lookup not available in bot scope, skip
			return sendText(b, ctx, "暂不支持 @用户名 方式，请使用用户 ID 或回复目标消息。", nil)
		}
		if id, err := strconv.ParseInt(first, 10, 64); err == nil && id > 0 {
			targetID = id
			args = args[1:] // consume the user ID
		}
	}

	if len(args) < 1 {
		return sendText(b, ctx, "请输入积分数值，如：加积分 10", nil)
	}

	delta, err := strconv.Atoi(args[0])
	if err != nil || delta == 0 {
		return sendText(b, ctx, "积分数值无效，请输入整数。", nil)
	}
	delta *= sign

	if err := a.services.Points.Adjust(scope.Context, scope.Chat.ID, targetID, delta, "manual_chinese_cmd"); err != nil {
		return err
	}

	verb := "增加"
	if delta < 0 {
		verb = "扣除"
	}
	summary, err := a.services.Points.GetSummary(scope.Context, scope.Chat.ID, targetID)
	if err != nil {
		return sendText(b, ctx, fmt.Sprintf("已为用户 %d %s %d 积分。", targetID, verb, absInt(delta)), nil)
	}
	return sendText(b, ctx, fmt.Sprintf("已为用户 %d %s %d 积分。\n%s", targetID, verb, absInt(delta), summary), nil)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
