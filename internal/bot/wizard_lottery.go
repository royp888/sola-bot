package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/dabowin/sola/internal/api"
)

const (
	lotteryWizardStepDrawMode = 1
	lotteryWizardStepDrawRule = 2
	lotteryWizardStepPrize    = 3
	lotteryWizardStepQuantity = 4
	lotteryWizardStepTitle    = 5
	lotteryWizardStepKeyword  = 6
	lotteryWizardStepCost     = 7
	lotteryWizardStepConfirm  = 99
)

func (a *App) startCreateLotteryWizard(b *gotgbot.Bot, ctx *ext.Context, chatID int64) error {
	return a.startCreateLotteryWizardWithJoinType(b, ctx, chatID, "button")
}

func (a *App) startCreateLotteryWizardWithJoinType(b *gotgbot.Bot, ctx *ext.Context, chatID int64, joinType string) error {
	scope := requestScope(ctx)
	if a.services.Lottery == nil {
		return respondText(b, ctx, "抽奖服务尚未接入。", nil)
	}
	joinType, ok := parseJoinTypeInput(joinType)
	if !ok {
		joinType = "button"
	}
	state := &ConversationState{
		Type:   "create_lottery",
		Step:   lotteryWizardStepDrawMode,
		ChatID: chatID,
		Data: map[string]any{
			"join_type": joinType,
		},
	}
	if err := a.setConversation(scope.Context, scope.Actor.ID, state); err != nil {
		return err
	}
	return respondText(b, ctx, lotteryWizardTitle(state)+"\n\n请选择开奖方式：", drawModeMarkup())
}

func (a *App) handleLotteryWizardStep(b *gotgbot.Bot, ctx *ext.Context, state *ConversationState) error {
	scope := requestScope(ctx)
	if ctx == nil || ctx.Message == nil {
		return nil
	}
	input := strings.TrimSpace(ctx.Message.Text)
	if strings.EqualFold(input, "/cancel") || strings.EqualFold(input, "cancel") || input == "取消" {
		a.clearConversation(scope.Context, scope.Actor.ID)
		return sendText(b, ctx, "已取消创建抽奖。", nil)
	}

	switch state.Step {
	case lotteryWizardStepDrawMode:
		mode, ok := parseDrawModeInput(input)
		if !ok {
			return sendText(b, ctx, "请选择按钮上的开奖方式，或输入“满人开奖 / 定时开奖”。", drawModeMarkup())
		}
		return a.applyLotteryDrawMode(b, ctx, state, mode)
	case lotteryWizardStepDrawRule:
		if stringVal(state.Data, "draw_mode") == "full" {
			value, err := parseWizardPositiveInt(input)
			if err != nil {
				return sendText(b, ctx, "请输入正整数，例如 10。", fullCountMarkup())
			}
			state.Data["max_participants"] = value
			state.Data["end_at"] = ""
			state.Step = lotteryWizardStepPrize
			return a.saveWizardAndSend(b, ctx, state, lotteryWizardTitle(state)+"\n\n请输入奖品名称：", cancelMarkup())
		}
		endAt, err := time.ParseInLocation("2006-01-02 15:04", input, time.Local)
		if err != nil || !endAt.After(time.Now()) {
			return sendText(b, ctx, "请点击快捷时间，或输入未来时间（格式：2026-06-01 20:00）：", endAtMarkup())
		}
		state.Data["end_at"] = endAt.Format(time.RFC3339)
		state.Data["max_participants"] = 0
		state.Step = lotteryWizardStepPrize
		return a.saveWizardAndSend(b, ctx, state, lotteryWizardTitle(state)+"\n\n请输入奖品名称：", cancelMarkup())
	case lotteryWizardStepPrize:
		if input == "" {
			return sendText(b, ctx, "奖品名称不能为空，请重新输入：", cancelMarkup())
		}
		state.Data["prize_name"] = input
		state.Step = lotteryWizardStepQuantity
		return a.saveWizardAndSend(b, ctx, state, lotteryWizardTitle(state)+"\n\n请输入奖品数量：", quickIntMarkup([]int{1, 2, 3, 5}, "wizard", "lottery_prize_count"))
	case lotteryWizardStepQuantity:
		count, err := parseWizardPositiveInt(input)
		if err != nil {
			return sendText(b, ctx, "请输入正整数：", quickIntMarkup([]int{1, 2, 3, 5}, "wizard", "lottery_prize_count"))
		}
		return a.applyLotteryPrizeCount(b, ctx, state, count, true)
	case lotteryWizardStepTitle:
		if input == "" {
			return sendText(b, ctx, "活动名称不能为空，请重新输入：", cancelMarkup())
		}
		state.Data["title"] = input
		if lotteryNeedsKeyword(stringVal(state.Data, "join_type")) {
			state.Step = lotteryWizardStepKeyword
			return a.saveWizardAndSend(b, ctx, state, lotteryWizardTitle(state)+"\n\n请输入参与口令：", cancelMarkup())
		}
		state.Step = lotteryWizardStepCost
		return a.saveWizardAndSend(b, ctx, state, lotteryWizardTitle(state)+"\n\n请输入参与所需积分（0 = 免费）：", quickIntMarkup([]int{0, 10, 50, 100}, "wizard", "lottery_cost"))
	case lotteryWizardStepKeyword:
		if input == "" {
			return sendText(b, ctx, "口令不能为空，请重新输入：", cancelMarkup())
		}
		state.Data["join_keyword"] = input
		state.Step = lotteryWizardStepCost
		return a.saveWizardAndSend(b, ctx, state, lotteryWizardTitle(state)+"\n\n请输入参与所需积分（0 = 免费）：", quickIntMarkup([]int{0, 10, 50, 100}, "wizard", "lottery_cost"))
	case lotteryWizardStepCost:
		cost, err := parseWizardNonNegativeInt(input)
		if err != nil {
			return sendText(b, ctx, "请输入非负整数：", quickIntMarkup([]int{0, 10, 50, 100}, "wizard", "lottery_cost"))
		}
		state.Data["cost_points"] = cost
		return a.confirmLotteryWizard(b, ctx, state)
	case lotteryWizardStepConfirm:
		return sendText(b, ctx, "请点击确认创建或取消。", confirmLotteryMarkup())
	default:
		a.clearConversation(scope.Context, scope.Actor.ID)
		return sendText(b, ctx, "向导状态已失效，请重新开始。", nil)
	}
}

func (a *App) routeWizardCallback(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	scope := requestScope(ctx)
	if payload.Action == "cancel" {
		a.clearConversation(scope.Context, scope.Actor.ID)
		_ = answerCallback(b, ctx, "已取消")
		return respondText(b, ctx, "已取消当前操作。", nil)
	}
	state, err := a.getConversation(scope.Context, scope.Actor.ID)
	if err != nil {
		return err
	}
	if state == nil {
		return answerCallback(b, ctx, "操作已超时，请重新开始")
	}
	if state.Type == "create_post" {
		if err := answerCallback(b, ctx, "处理中"); err != nil {
			return err
		}
		return a.routePostWizardCallback(b, ctx, state, payload)
	}
	if state.Type != "create_lottery" {
		a.clearConversation(scope.Context, scope.Actor.ID)
		return answerCallback(b, ctx, "向导状态无效")
	}
	if err := answerCallback(b, ctx, "处理中"); err != nil {
		return err
	}

	switch payload.Action {
	case "lottery_join_type":
		joinType, ok := parseJoinTypeInput(payload.Resource)
		if !ok {
			return respondText(b, ctx, "参与方式无效，请重新选择：", joinTypeMarkup())
		}
		state.Data["join_type"] = joinType
		state.Step = lotteryWizardStepDrawMode
		return a.saveWizardAndRespond(b, ctx, state, lotteryWizardTitle(state)+"\n\n请选择开奖方式：", drawModeMarkup())
	case "lottery_draw_mode":
		return a.applyLotteryDrawMode(b, ctx, state, payload.Resource)
	case "lottery_full_count":
		value, err := parseWizardPositiveInt(payload.Resource)
		if err != nil {
			return respondText(b, ctx, "满人开奖人数无效，请重新输入：", fullCountMarkup())
		}
		state.Data["max_participants"] = value
		state.Data["end_at"] = ""
		state.Step = lotteryWizardStepPrize
		return a.saveWizardAndRespond(b, ctx, state, lotteryWizardTitle(state)+"\n\n请输入奖品名称：", cancelMarkup())
	case "lottery_end_at":
		if payload.Resource == "custom" {
			state.Step = lotteryWizardStepDrawRule
			state.Data["draw_mode"] = "time"
			return a.saveWizardAndRespond(b, ctx, state, "请输入开奖时间（格式：2026-06-01 20:00）：", cancelMarkup())
		}
		duration, err := time.ParseDuration(payload.Resource)
		if err != nil || duration <= 0 {
			return respondText(b, ctx, "开奖时间无效，请重新选择：", endAtMarkup())
		}
		state.Data["end_at"] = time.Now().Add(duration).Format(time.RFC3339)
		state.Data["max_participants"] = 0
		state.Step = lotteryWizardStepPrize
		return a.saveWizardAndRespond(b, ctx, state, lotteryWizardTitle(state)+"\n\n请输入奖品名称：", cancelMarkup())
	case "lottery_prize_count":
		count, err := parseWizardPositiveInt(payload.Resource)
		if err != nil {
			return respondText(b, ctx, "奖品数量无效，请重新输入：", quickIntMarkup([]int{1, 2, 3, 5}, "wizard", "lottery_prize_count"))
		}
		return a.applyLotteryPrizeCount(b, ctx, state, count, false)
	case "lottery_cost":
		cost, err := parseWizardNonNegativeInt(payload.Resource)
		if err != nil {
			return respondText(b, ctx, "参与所需积分无效，请重新输入：", quickIntMarkup([]int{0, 10, 50, 100}, "wizard", "lottery_cost"))
		}
		state.Data["cost_points"] = cost
		return a.confirmLotteryWizard(b, ctx, state)
	case "lottery_confirm":
		if payload.Resource == "no" {
			a.clearConversation(scope.Context, scope.Actor.ID)
			return respondText(b, ctx, "已取消创建抽奖。", nil)
		}
		return a.confirmCreateLottery(b, ctx, state)
	default:
		return answerCallback(b, ctx, "未知操作")
	}
}

func (a *App) applyLotteryDrawMode(b *gotgbot.Bot, ctx *ext.Context, state *ConversationState, mode string) error {
	mode, ok := parseDrawModeInput(mode)
	if !ok {
		return respondText(b, ctx, "开奖方式无效，请重新选择：", drawModeMarkup())
	}
	state.Data["draw_mode"] = mode
	state.Step = lotteryWizardStepDrawRule
	if mode == "full" {
		return a.saveWizardAndRespond(b, ctx, state, lotteryWizardTitle(state)+"\n\n请输入多少人参与后开奖：", fullCountMarkup())
	}
	return a.saveWizardAndRespond(b, ctx, state, lotteryWizardTitle(state)+"\n\n请选择开奖时间：", endAtMarkup())
}

func (a *App) applyLotteryPrizeCount(b *gotgbot.Bot, ctx *ext.Context, state *ConversationState, count int, send bool) error {
	state.Data["prize_count"] = count
	state.Data["winner_count"] = count
	state.Step = lotteryWizardStepTitle
	text := lotteryWizardTitle(state) + "\n\n请输入活动名称："
	if send {
		return a.saveWizardAndSend(b, ctx, state, text, cancelMarkup())
	}
	return a.saveWizardAndRespond(b, ctx, state, text, cancelMarkup())
}

func (a *App) confirmLotteryWizard(b *gotgbot.Bot, ctx *ext.Context, state *ConversationState) error {
	scope := requestScope(ctx)
	drawMode := stringVal(state.Data, "draw_mode")
	text := fmt.Sprintf(
		"请确认抽奖信息：\n\n类型：%s\n活动：%s\n奖品：%s\n开奖方式：%s\n%s%s参与费用：%d 积分",
		lotteryJoinTypeLabel(stringVal(state.Data, "join_type")),
		stringVal(state.Data, "title"),
		lotteryPrizeText(state.Data),
		lotteryDrawModeLabel(drawMode),
		lotteryDrawRuleLine(state.Data),
		keywordLine(stringVal(state.Data, "join_type"), stringVal(state.Data, "join_keyword")),
		intVal(state.Data, "cost_points"),
	)
	state.Step = lotteryWizardStepConfirm
	if err := a.setConversation(scope.Context, scope.Actor.ID, state); err != nil {
		return err
	}
	return respondText(b, ctx, text, confirmLotteryMarkup())
}

func (a *App) confirmCreateLottery(b *gotgbot.Bot, ctx *ext.Context, state *ConversationState) error {
	scope := requestScope(ctx)
	a.clearConversation(scope.Context, scope.Actor.ID)
	var endAt *time.Time
	if raw := stringVal(state.Data, "end_at"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return respondText(b, ctx, "创建失败：开奖时间无效，请重新创建。", nil)
		}
		endAt = &parsed
	}
	req := api.LotteryCreateRequest{
		ChatID:          state.ChatID,
		Title:           stringVal(state.Data, "title"),
		Prize:           lotteryPrizeText(state.Data),
		CostPoints:      intVal(state.Data, "cost_points"),
		MaxParticipants: intVal(state.Data, "max_participants"),
		WinnerCount:     intVal(state.Data, "winner_count"),
		EndAt:           endAt,
		CreatedBy:       scope.Actor.ID,
		JoinType:        stringVal(state.Data, "join_type"),
		JoinKeyword:     stringVal(state.Data, "join_keyword"),
	}
	lottery, err := a.services.Lottery.Create(scope.Context, req)
	if err != nil {
		return respondText(b, ctx, "创建失败："+err.Error(), nil)
	}
	if lottery.JoinType == "" {
		lottery.JoinType = req.JoinType
	}
	if lottery.JoinKeyword == "" {
		lottery.JoinKeyword = req.JoinKeyword
	}
	if err := sendLotteryAnnouncement(b, ctx, *lottery); err != nil {
		return respondText(b, ctx, fmt.Sprintf("抽奖已创建（#%d），但发送公告失败：%s", lottery.ID, err.Error()), nil)
	}
	return respondText(b, ctx, fmt.Sprintf("%s #%d 已创建，公告已发送到目标群组。", lotteryJoinTypeLabel(lottery.JoinType), lottery.ID), nil)
}

func (a *App) saveWizardAndSend(b *gotgbot.Bot, ctx *ext.Context, state *ConversationState, text string, opts *gotgbot.SendMessageOpts) error {
	scope := requestScope(ctx)
	if err := a.setConversation(scope.Context, scope.Actor.ID, state); err != nil {
		return err
	}
	return sendText(b, ctx, text, opts)
}

func (a *App) saveWizardAndRespond(b *gotgbot.Bot, ctx *ext.Context, state *ConversationState, text string, opts *gotgbot.SendMessageOpts) error {
	scope := requestScope(ctx)
	if err := a.setConversation(scope.Context, scope.Actor.ID, state); err != nil {
		return err
	}
	return respondText(b, ctx, text, opts)
}

func cancelMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{cancelRow()}}}
}

func cancelRow() []gotgbot.InlineKeyboardButton {
	return []gotgbot.InlineKeyboardButton{{Text: "取消", CallbackData: CallbackData("wizard", "cancel")}}
}

func drawModeMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "满人开奖", CallbackData: CallbackData("wizard", "lottery_draw_mode", "full")},
			{Text: "定时开奖", CallbackData: CallbackData("wizard", "lottery_draw_mode", "time")},
		},
		cancelRow(),
	}}}
}

func fullCountMarkup() *gotgbot.SendMessageOpts {
	return quickIntMarkup([]int{10, 50, 100, 200}, "wizard", "lottery_full_count")
}

func joinTypeMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "按钮参与", CallbackData: CallbackData("wizard", "lottery_join_type", "button")},
			{Text: "口令参与", CallbackData: CallbackData("wizard", "lottery_join_type", "keyword")},
		},
		{
			{Text: "两种都支持", CallbackData: CallbackData("wizard", "lottery_join_type", "both")},
		},
		cancelRow(),
	}}}
}

func endAtMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "10分钟后", CallbackData: CallbackData("wizard", "lottery_end_at", "10m")},
			{Text: "1小时后", CallbackData: CallbackData("wizard", "lottery_end_at", "1h")},
		},
		{
			{Text: "3小时后", CallbackData: CallbackData("wizard", "lottery_end_at", "3h")},
			{Text: "明天此时", CallbackData: CallbackData("wizard", "lottery_end_at", "24h")},
		},
		{
			{Text: "自定义时间", CallbackData: CallbackData("wizard", "lottery_end_at", "custom")},
		},
		cancelRow(),
	}}}
}

func quickIntMarkup(values []int, domain string, action string) *gotgbot.SendMessageOpts {
	buttons := make([]gotgbot.InlineKeyboardButton, 0, len(values))
	for _, value := range values {
		raw := strconv.Itoa(value)
		buttons = append(buttons, gotgbot.InlineKeyboardButton{Text: raw, CallbackData: CallbackData(domain, action, raw)})
	}
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{buttons, cancelRow()}}}
}

func confirmLotteryMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "确认创建", CallbackData: CallbackData("wizard", "lottery_confirm", "yes")},
			{Text: "取消", CallbackData: CallbackData("wizard", "lottery_confirm", "no")},
		},
	}}}
}

func parseJoinTypeInput(input string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "", "button", "按钮", "按钮参与", "按钮抽奖":
		return "button", true
	case "keyword", "口令", "口令参与", "口令抽奖":
		return "keyword", true
	case "both", "all", "两种", "两种都支持", "按钮+口令", "按钮 + 口令":
		return "both", true
	default:
		return "", false
	}
}

func parseDrawModeInput(input string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "full", "满人", "满人开奖", "人数", "人数开奖":
		return "full", true
	case "time", "定时", "定时开奖", "时间", "时间开奖":
		return "time", true
	default:
		return "", false
	}
}

func parseWizardNonNegativeInt(input string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid non-negative integer")
	}
	return value, nil
}

func parseWizardPositiveInt(input string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid positive integer")
	}
	return value, nil
}

func lotteryWizardTitle(state *ConversationState) string {
	joinType := "button"
	if state != nil && state.Data != nil {
		joinType = stringVal(state.Data, "join_type")
	}
	return "创建" + lotteryJoinTypeLabel(joinType)
}

func lotteryNeedsKeyword(joinType string) bool {
	joinType = strings.ToLower(strings.TrimSpace(joinType))
	return joinType == "keyword" || joinType == "both"
}

func lotteryPrizeText(data map[string]any) string {
	name := strings.TrimSpace(stringVal(data, "prize_name"))
	if name == "" {
		name = strings.TrimSpace(stringVal(data, "prize"))
	}
	count := intVal(data, "prize_count")
	if count > 0 {
		return fmt.Sprintf("%s x%d", lotteryTextFallback(name, "未填写"), count)
	}
	return lotteryTextFallback(name, "未填写")
}

func lotteryDrawModeLabel(mode string) string {
	if mode == "full" {
		return "满人开奖"
	}
	return "定时开奖"
}

func lotteryDrawRuleLine(data map[string]any) string {
	if stringVal(data, "draw_mode") == "full" {
		return fmt.Sprintf("开奖人数：%d 人\n", intVal(data, "max_participants"))
	}
	endAt, _ := time.Parse(time.RFC3339, stringVal(data, "end_at"))
	if endAt.IsZero() {
		return "开奖时间：未设置\n"
	}
	return fmt.Sprintf("开奖时间：%s\n", endAt.Format("2006-01-02 15:04"))
}

func keywordLine(joinType string, keyword string) string {
	if lotteryNeedsKeyword(joinType) && strings.TrimSpace(keyword) != "" {
		return fmt.Sprintf("口令：%s\n", strings.TrimSpace(keyword))
	}
	return ""
}

func stringVal(data map[string]any, key string) string {
	if value, ok := data[key]; ok {
		if text, ok := value.(string); ok {
			return text
		}
	}
	return ""
}

func intVal(data map[string]any, key string) int {
	if value, ok := data[key]; ok {
		switch typed := value.(type) {
		case int:
			return typed
		case int64:
			return int(typed)
		case float64:
			return int(typed)
		case string:
			parsed, _ := strconv.Atoi(typed)
			return parsed
		}
	}
	return 0
}
