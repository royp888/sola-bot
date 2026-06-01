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

func (a *App) handleLottery(b *gotgbot.Bot, ctx *ext.Context) error {
	args := commandArgs(ctx)
	if len(args) == 0 || strings.EqualFold(args[0], "list") {
		return a.showActiveLottery(b, ctx)
	}

	switch strings.ToLower(args[0]) {
	case "create":
		return a.createLotteryByCommand(b, ctx)
	case "join":
		return a.joinLotteryByCommand(b, ctx, args[1:])
	case "info":
		return a.showLotteryInfo(b, ctx, args[1:])
	case "cancel":
		return a.cancelLotteryByCommand(b, ctx, args[1:])
	default:
		return sendText(b, ctx, lotteryUsage(), nil)
	}
}

func (a *App) routeLotteryCallback(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	switch payload.Action {
	case "active":
		return a.showActiveLottery(b, ctx)
	case "join":
		return a.joinLotteryFromCallback(b, ctx, payload)
	case "info":
		return a.showLotteryInfoFromCallback(b, ctx, payload)
	case "cancel":
		return a.cancelLotteryFromCallback(b, ctx, payload)
	case "create_help":
		return a.showLotteryCreateHelp(b, ctx)
	case "create":
		joinType := ""
		if payload.Resource != "" {
			joinType = payload.Resource
		}
		if joinType == "" {
			return a.showLotteryCreateEntry(b, ctx)
		}
		scope := requestScope(ctx)
		if scope.Chat.Type == "private" {
			return a.showPrivateHome(b, ctx)
		}
		if err := a.requireTelegramManager(b, ctx); err != nil {
			return err
		}
		return a.startCreateLotteryWizardWithJoinType(b, ctx, scope.Chat.ID, joinType)
	case "draw":
		return respondText(b, ctx, "开奖由 Worker 自动执行。后台创建抽奖时设置开奖时间，到点后会随机开奖并在群里公布。", lotteryMenuMarkup(nil, isGroupChatType(requestScope(ctx).Chat.Type)))
	default:
		return a.showActiveLottery(b, ctx)
	}
}

func (a *App) showActiveLottery(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	return a.showActiveLotteryForChat(b, ctx, scope.Chat.ID, lotteryMenuMarkup(nil, true))
}

func (a *App) showActiveLotteryForChat(b *gotgbot.Bot, ctx *ext.Context, chatID int64, fallbackMarkup *gotgbot.SendMessageOpts) error {
	if a.services.Lottery == nil {
		return respondText(b, ctx, "抽奖服务尚未接入。", fallbackMarkup)
	}
	items, err := a.services.Lottery.ListActiveItems(requestScope(ctx).Context, chatID, 8)
	if err != nil {
		return err
	}
	active := formatActiveLotteryItems(items)
	if len(items) == 0 {
		recent, err := a.services.Lottery.ListItems(requestScope(ctx).Context, chatID, 5)
		if err != nil {
			return err
		}
		active = formatRecentLotteryItems(recent)
	}
	if requestScope(ctx).Chat.Type == "private" {
		return respondText(b, ctx, active, fallbackMarkup)
	}
	return respondText(b, ctx, active, lotteryMenuMarkup(items, true))
}

func (a *App) createLotteryByCommand(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if a.services.Lottery == nil {
		return respondText(b, ctx, "抽奖服务尚未接入。", lotteryMenuMarkup(nil, isGroupChatType(scope.Chat.Type)))
	}
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}

	req, err := parseLotteryCreateRequest(ctx, scope.Chat.ID, scope.Actor.ID)
	if err != nil {
		return sendText(b, ctx, err.Error()+"\n"+lotteryCreateUsage(), nil)
	}
	lottery, err := a.services.Lottery.Create(scope.Context, req)
	if err != nil {
		return err
	}
	return sendLotteryAnnouncement(b, ctx, *lottery)
}

func (a *App) showLotteryCreateEntry(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if scope.Chat.Type == "private" {
		return a.showPrivateHome(b, ctx)
	}
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	text := strings.Join([]string{
		"创建抽奖",
		"",
		"请选择参与方式：",
		"按钮抽奖：成员点击按钮参与",
		"口令抽奖：成员发送口令参与",
		"双模式：按钮和口令都支持",
	}, "\n")
	return respondText(b, ctx, text, lotteryCreateTypeGroupMarkup())
}

func lotteryCreateTypeGroupMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "按钮抽奖", CallbackData: CallbackData("lottery", "create", "button")},
			{Text: "口令抽奖", CallbackData: CallbackData("lottery", "create", "keyword")},
		},
		{
			{Text: "双模式", CallbackData: CallbackData("lottery", "create", "both")},
			{Text: "返回列表", CallbackData: CallbackData("lottery", "active")},
		},
	}}}
}
func (a *App) joinLotteryFromCallback(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	if payload.Resource == "" {
		return answerCallback(b, ctx, "抽奖 ID 缺失")
	}
	message, err := a.joinLotteryMessage(b, ctx, payload.Resource)
	if err != nil {
		if answerErr := answerCallback(b, ctx, callbackAnswerText(err.Error())); answerErr != nil {
			return answerErr
		}
		return nil
	}
	return answerCallback(b, ctx, callbackAnswerText(message))
}

func (a *App) showLotteryInfoFromCallback(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	if payload.Resource == "" {
		return answerCallback(b, ctx, "抽奖 ID 缺失")
	}
	scope := requestScope(ctx)
	lotteryID, err := parseLotteryID(payload.Resource)
	if err != nil {
		return answerCallback(b, ctx, "抽奖 ID 无效")
	}
	info, err := a.services.Lottery.Info(scope.Context, scope.Chat.ID, lotteryID)
	if err != nil {
		return err
	}
	item, err := a.services.Lottery.GetItem(scope.Context, scope.Chat.ID, lotteryID)
	if err != nil {
		return err
	}
	return respondText(b, ctx, info, lotteryDetailMarkup(item, isGroupChatType(scope.Chat.Type)))
}

func (a *App) cancelLotteryFromCallback(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error {
	if payload.Resource == "" {
		return answerCallback(b, ctx, "抽奖 ID 缺失")
	}
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	scope := requestScope(ctx)
	lotteryID, err := parseLotteryID(payload.Resource)
	if err != nil {
		return answerCallback(b, ctx, "抽奖 ID 无效")
	}
	message, err := a.services.Lottery.CancelForChat(scope.Context, scope.Chat.ID, lotteryID, scope.Actor.ID)
	if err != nil {
		return err
	}
	return respondText(b, ctx, message, lotteryMenuMarkup(nil, isGroupChatType(requestScope(ctx).Chat.Type)))
}

func (a *App) showLotteryCreateHelp(b *gotgbot.Bot, ctx *ext.Context) error {
	text := strings.Join([]string{
		"抽奖创建说明",
		"",
		"支持三种参与方式：",
		"1. 按钮抽奖：成员点按钮直接参与",
		"2. 口令抽奖：成员发送指定口令参与",
		"3. 双模式：按钮和口令都可参与",
		"",
		"群聊里：管理员点“创建抽奖”即可进入向导。",
		"私聊里：先选目标群，再到工作台创建。",
		"",
		"命令创建：",
		"/lottery create 标题 | 奖品 | 0 | 100 | 1 | 2026-06-01 21:30 | button",
		"/lottery create 标题 | 奖品 | 10 | 0 | 1 | 2026-06-01 21:30 | keyword | 口令",
	}, "\n")
	return respondText(b, ctx, text, lotteryMenuMarkup(nil, isGroupChatType(requestScope(ctx).Chat.Type)))
}

func (a *App) joinLotteryByCommand(b *gotgbot.Bot, ctx *ext.Context, args []string) error {
	if len(args) < 1 {
		return sendText(b, ctx, "用法：/lottery join <id>", nil)
	}
	return a.joinLotteryByID(b, ctx, args[0])
}

func (a *App) joinLotteryByID(b *gotgbot.Bot, ctx *ext.Context, rawID string) error {
	message, err := a.joinLotteryMessage(b, ctx, rawID)
	if err != nil {
		return err
	}
	return respondText(b, ctx, message, lotteryMenuMarkup(nil, isGroupChatType(requestScope(ctx).Chat.Type)))
}

func (a *App) joinLotteryMessage(b *gotgbot.Bot, ctx *ext.Context, rawID string) (string, error) {
	_ = b
	scope := requestScope(ctx)
	if a.services.Lottery == nil {
		return "抽奖服务尚未接入。", nil
	}
	lotteryID, err := parseLotteryID(rawID)
	if err != nil {
		return "", fmt.Errorf("抽奖 ID 必须是整数")
	}
	message, err := a.services.Lottery.Join(scope.Context, scope.Chat.ID, lotteryID, scope.Actor.ID, scope.Actor.Username)
	if err != nil {
		return "", err
	}
	if lotteryWasAutoDrawn(message) {
		if announceErr := a.announceLotteryResult(b, ctx, lotteryID); announceErr != nil {
			return message, nil
		}
	}
	return message, nil
}

func (a *App) showLotteryInfo(b *gotgbot.Bot, ctx *ext.Context, args []string) error {
	scope := requestScope(ctx)
	if a.services.Lottery == nil {
		return respondText(b, ctx, "抽奖服务尚未接入。", lotteryMenuMarkup(nil, isGroupChatType(scope.Chat.Type)))
	}
	if len(args) < 1 {
		return respondText(b, ctx, "用法：/lottery info <id>", lotteryMenuMarkup(nil, isGroupChatType(scope.Chat.Type)))
	}
	lotteryID, err := parseLotteryID(args[0])
	if err != nil {
		return respondText(b, ctx, "抽奖 ID 必须是整数。", lotteryMenuMarkup(nil, isGroupChatType(scope.Chat.Type)))
	}
	info, err := a.services.Lottery.Info(scope.Context, scope.Chat.ID, lotteryID)
	if err != nil {
		return err
	}
	item, itemErr := a.services.Lottery.GetItem(scope.Context, scope.Chat.ID, lotteryID)
	if itemErr == nil {
		return respondText(b, ctx, info, lotteryDetailMarkup(item, isGroupChatType(scope.Chat.Type)))
	}
	return respondText(b, ctx, info, lotteryMenuMarkup(nil, isGroupChatType(scope.Chat.Type)))
}

func (a *App) cancelLotteryByCommand(b *gotgbot.Bot, ctx *ext.Context, args []string) error {
	scope := requestScope(ctx)
	if a.services.Lottery == nil {
		return respondText(b, ctx, "抽奖服务尚未接入。", lotteryMenuMarkup(nil, isGroupChatType(scope.Chat.Type)))
	}
	if len(args) < 1 {
		return respondText(b, ctx, "用法：/lottery cancel <id>", lotteryMenuMarkup(nil, isGroupChatType(scope.Chat.Type)))
	}
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	lotteryID, err := parseLotteryID(args[0])
	if err != nil {
		return respondText(b, ctx, "抽奖 ID 必须是整数。", lotteryMenuMarkup(nil, isGroupChatType(scope.Chat.Type)))
	}
	message, err := a.services.Lottery.CancelForChat(scope.Context, scope.Chat.ID, lotteryID, scope.Actor.ID)
	if err != nil {
		return err
	}
	return respondText(b, ctx, message, lotteryMenuMarkup(nil, isGroupChatType(requestScope(ctx).Chat.Type)))
}

func parseLotteryID(raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid lottery id")
	}
	return value, nil
}

func parseLotteryCreateRequest(ctx *ext.Context, chatID int64, createdBy int64) (api.LotteryCreateRequest, error) {
	tail := lotteryCommandTail(ctx)
	if tail == "" {
		return api.LotteryCreateRequest{}, fmt.Errorf("缺少创建参数")
	}
	lowerTail := strings.ToLower(tail)
	if lowerTail != "create" && !strings.HasPrefix(lowerTail, "create ") {
		return api.LotteryCreateRequest{}, fmt.Errorf("缺少 create 子命令")
	}
	raw := strings.TrimSpace(tail[len("create"):])
	parts := strings.Split(raw, "|")
	if len(parts) != 6 && len(parts) != 8 {
		return api.LotteryCreateRequest{}, fmt.Errorf("创建参数需要 6 段或 8 段，用 | 分隔")
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	title := parts[0]
	if title == "" {
		return api.LotteryCreateRequest{}, fmt.Errorf("标题不能为空")
	}
	cost, err := parseNonNegativeInt(parts[2], "参与消耗")
	if err != nil {
		return api.LotteryCreateRequest{}, err
	}
	maxParticipants, err := parseNonNegativeInt(parts[3], "人数上限")
	if err != nil {
		return api.LotteryCreateRequest{}, err
	}
	winners, err := strconv.Atoi(parts[4])
	if err != nil || winners <= 0 {
		return api.LotteryCreateRequest{}, fmt.Errorf("中奖人数必须是正整数")
	}
	endAt, err := time.ParseInLocation("2006-01-02 15:04", parts[5], time.Local)
	if err != nil {
		return api.LotteryCreateRequest{}, fmt.Errorf("结束时间格式必须是 YYYY-MM-DD HH:mm")
	}
	if !endAt.After(time.Now()) {
		return api.LotteryCreateRequest{}, fmt.Errorf("结束时间必须晚于当前时间")
	}
	joinType := "button"
	joinKeyword := ""
	if len(parts) == 8 {
		joinType = strings.TrimSpace(parts[6])
		joinKeyword = strings.TrimSpace(parts[7])
	}
	return api.LotteryCreateRequest{
		ChatID:          chatID,
		Title:           title,
		Prize:           parts[1],
		CostPoints:      cost,
		MaxParticipants: maxParticipants,
		WinnerCount:     winners,
		EndAt:           &endAt,
		CreatedBy:       createdBy,
		JoinType:        joinType,
		JoinKeyword:     joinKeyword,
	}, nil
}

func parseNonNegativeInt(raw string, label string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s必须是非负整数", label)
	}
	return value, nil
}

func lotteryCommandTail(ctx *ext.Context) string {
	if ctx == nil || ctx.Message == nil {
		return ""
	}
	text := strings.TrimSpace(ctx.Message.Text)
	if text == "" {
		return ""
	}
	parts := strings.SplitN(text, " ", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func sendLotteryAnnouncement(b *gotgbot.Bot, ctx *ext.Context, lottery api.Lottery) error {
	text := lotteryAnnouncementText(lottery)
	opts := &gotgbot.SendMessageOpts{}
	if lotteryHasJoinButton(lottery.JoinType) {
		buttons := [][]gotgbot.InlineKeyboardButton{{
			{Text: "立即参与", CallbackData: CallbackData("lottery", "join", strconv.FormatInt(lottery.ID, 10))},
			{Text: "查看详情", CallbackData: CallbackData("lottery", "info", strconv.FormatInt(lottery.ID, 10))},
		}}
		opts.ReplyMarkup = gotgbot.InlineKeyboardMarkup{InlineKeyboard: buttons}
	}
	_, err := b.SendMessageWithContext(requestScope(ctx).Context, lottery.ChatID, text, opts)
	return err
}

func lotteryAnnouncementText(lottery api.Lottery) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("🎁 %s #%d\n", lotteryJoinTypeLabel(lottery.JoinType), lottery.ID))
	builder.WriteString("━━━━━━━━━━\n")
	builder.WriteString(fmt.Sprintf("活动：%s\n", lotteryTextFallback(lottery.Title, "未命名抽奖")))
	builder.WriteString(fmt.Sprintf("奖品：%s\n", lotteryTextFallback(lottery.Prize, "未填写")))
	builder.WriteString(fmt.Sprintf("参与方式：%s\n", lotteryJoinModeHint(lottery.JoinType, lottery.JoinKeyword)))
	builder.WriteString(fmt.Sprintf("所需积分：%d\n", maxInt(lottery.CostPoints, 0)))
	builder.WriteString(fmt.Sprintf("中奖人数：%d\n", lottery.WinnerCount))
	if lottery.MaxParticipants > 0 {
		builder.WriteString(fmt.Sprintf("开奖条件：满 %d 人自动开奖\n", lottery.MaxParticipants))
	} else {
		builder.WriteString("开奖条件：按设定时间开奖\n")
	}
	if lottery.EndAt != nil {
		builder.WriteString(fmt.Sprintf("开奖时间：%s\n", lottery.EndAt.Format("2006-01-02 15:04")))
	}
	builder.WriteString("━━━━━━━━━━\n")
	switch strings.ToLower(strings.TrimSpace(lottery.JoinType)) {
	case "keyword":
		kw := lotteryTextFallback(strings.TrimSpace(lottery.JoinKeyword), "未设置")
		builder.WriteString(fmt.Sprintf("口令：%s\n", kw))
		builder.WriteString(fmt.Sprintf("发送口令「%s」参与抽奖。", kw))
	case "both":
		kw := lotteryTextFallback(strings.TrimSpace(lottery.JoinKeyword), "未设置")
		builder.WriteString(fmt.Sprintf("口令：%s\n", kw))
		builder.WriteString(fmt.Sprintf("发送口令「%s」或点击下方按钮参与抽奖。", kw))
	default:
		builder.WriteString("点击下方按钮参与抽奖。")
	}
	return builder.String()
}

func lotteryJoinModeHint(joinType string, keyword string) string {
	switch strings.ToLower(strings.TrimSpace(joinType)) {
	case "keyword":
		return "口令参与"
	case "both":
		if strings.TrimSpace(keyword) == "" {
			return "按钮 + 口令"
		}
		return fmt.Sprintf("按钮 + 口令（%s）", strings.TrimSpace(keyword))
	default:
		return "按钮参与"
	}
}

func maxInt(value int, fallback int) int {
	if value < fallback {
		return fallback
	}
	return value
}

func lotteryUsage() string {
	return "用法：/lottery list | /lottery create <title> | <prize> | <cost> | <max> | <winners> | <YYYY-MM-DD HH:mm> [| <join_type> | <join_keyword>] | /lottery join <id> | /lottery info <id> | /lottery cancel <id>"
}

func lotteryCreateUsage() string {
	return "用法：/lottery create <title> | <prize> | <cost> | <max> | <winners> | <YYYY-MM-DD HH:mm> [| button|keyword|both | 口令]"
}

func lotteryTextFallback(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func lotteryJoinTypeLabel(joinType string) string {
	switch strings.ToLower(strings.TrimSpace(joinType)) {
	case "keyword":
		return "口令抽奖活动"
	case "both":
		return "按钮 + 口令抽奖活动"
	default:
		return "按钮抽奖活动"
	}
}

func callbackAnswerText(text string) string {
	text = strings.TrimSpace(text)
	if len([]rune(text)) <= 180 {
		return text
	}
	runes := []rune(text)
	return string(runes[:177]) + "..."
}

func formatActiveLotteryItems(items []api.Lottery) string {
	if len(items) == 0 {
		return "🎁 抽奖大厅\n\n当前没有进行中的抽奖。\n管理员可以直接点下方“创建抽奖”，成员后续在群里用按钮或口令参与。"
	}
	var builder strings.Builder
	builder.WriteString("🎁 抽奖大厅\n")
	builder.WriteString("━━━━━━━━━━\n")
	builder.WriteString(fmt.Sprintf("进行中：%d 场\n\n", len(items)))
	for _, item := range items {
		builder.WriteString(fmt.Sprintf("#%d %s\n", item.ID, lotteryTextFallback(item.Title, "未命名抽奖")))
		builder.WriteString(fmt.Sprintf("奖品：%s\n", lotteryTextFallback(item.Prize, "未填写")))
		builder.WriteString(fmt.Sprintf("参与方式：%s\n", lotteryJoinModeHint(item.JoinType, item.JoinKeyword)))
		builder.WriteString(fmt.Sprintf("已参与：%d", item.EntryCount))
		if item.MaxParticipants > 0 {
			builder.WriteString(fmt.Sprintf("/%d", item.MaxParticipants))
		}
		if item.CostPoints > 0 {
			builder.WriteString(fmt.Sprintf(" · %d 积分", item.CostPoints))
		}
		if item.EndAt != nil {
			builder.WriteString(fmt.Sprintf("\n开奖时间：%s", item.EndAt.Format("2006-01-02 15:04")))
		} else if item.MaxParticipants > 0 {
			builder.WriteString(fmt.Sprintf("\n开奖条件：满 %d 人", item.MaxParticipants))
		}
		builder.WriteString("\n\n")
	}
	builder.WriteString("成员可点按钮参与；口令型活动按公告发送口令即可。")
	return strings.TrimSpace(builder.String())
}

func formatRecentLotteryItems(items []api.Lottery) string {
	if len(items) == 0 {
		return "🎁 抽奖大厅\n\n当前没有进行中的抽奖，也没有历史抽奖记录。\n管理员可以直接点下方“创建抽奖”开始第一场活动。"
	}
	var builder strings.Builder
	builder.WriteString("🎁 抽奖大厅\n")
	builder.WriteString("━━━━━━━━━━\n")
	builder.WriteString("当前没有进行中的抽奖\n\n最近记录\n")
	for _, item := range items {
		builder.WriteString(fmt.Sprintf("#%d %s · %s\n", item.ID, lotteryTextFallback(item.Title, "未命名抽奖"), lotteryStatusLabel(item.Status)))
		builder.WriteString(fmt.Sprintf("奖品：%s\n", lotteryTextFallback(item.Prize, "未填写")))
		builder.WriteString(fmt.Sprintf("参与：%d", item.EntryCount))
		if item.WinnerCountDone > 0 {
			builder.WriteString(fmt.Sprintf(" · 已中奖：%d", item.WinnerCountDone))
		}
		if item.EndAt != nil {
			builder.WriteString(fmt.Sprintf("\n开奖时间：%s", item.EndAt.Format("2006-01-02 15:04")))
		}
		builder.WriteString("\n\n")
	}
	builder.WriteString("可从下方重新创建新活动。")
	return strings.TrimSpace(builder.String())
}

func lotteryStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		return "进行中"
	case "ended":
		return "已开奖"
	case "cancelled":
		return "已取消"
	default:
		return lotteryTextFallback(status, "未知")
	}
}

func lotteryMenuMarkup(items []api.Lottery, canCreate bool) *gotgbot.SendMessageOpts {
	rows := make([][]gotgbot.InlineKeyboardButton, 0, len(items)+4)
	for _, item := range items {
		id := strconv.FormatInt(item.ID, 10)
		title := lotteryTextFallback(item.Title, fmt.Sprintf("#%d", item.ID))
		if len([]rune(title)) > 10 {
			title = string([]rune(title)[:10]) + "..."
		}
		row := []gotgbot.InlineKeyboardButton{{Text: "详情 " + title, CallbackData: CallbackData("lottery", "info", id)}}
		if lotteryHasJoinButton(item.JoinType) {
			row = append(row, gotgbot.InlineKeyboardButton{Text: "立即参与", CallbackData: CallbackData("lottery", "join", id)})
		}
		rows = append(rows, row)
	}
	rows = append(rows,
		[]gotgbot.InlineKeyboardButton{
			{Text: "进行中列表", CallbackData: CallbackData("lottery", "active")},
			{Text: "创建说明", CallbackData: CallbackData("lottery", "create_help")},
		},
	)
	if canCreate {
		rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "创建抽奖", CallbackData: CallbackData("lottery", "create")}})
	} else {
		rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "私聊创建", CallbackData: CallbackData("private", "lottery_create")}})
	}
	rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "返回群组", CallbackData: CallbackData("menu", "groups")}})
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}}
}

func lotteryDetailMarkup(lottery api.Lottery, groupView bool) *gotgbot.SendMessageOpts {
	id := strconv.FormatInt(lottery.ID, 10)
	rows := [][]gotgbot.InlineKeyboardButton{}
	first := []gotgbot.InlineKeyboardButton{}
	if lottery.Status == "active" && lotteryHasJoinButton(lottery.JoinType) {
		first = append(first, gotgbot.InlineKeyboardButton{Text: "立即参与", CallbackData: CallbackData("lottery", "join", id)})
	}
	if lottery.Status == "active" {
		first = append(first, gotgbot.InlineKeyboardButton{Text: "取消抽奖", CallbackData: CallbackData("lottery", "cancel", id)})
	}
	if len(first) > 0 {
		rows = append(rows, first)
	}
	if groupView {
		rows = append(rows,
			[]gotgbot.InlineKeyboardButton{{Text: "返回抽奖大厅", CallbackData: CallbackData("lottery", "active")}},
			[]gotgbot.InlineKeyboardButton{{Text: "创建新抽奖", CallbackData: CallbackData("lottery", "create")}},
		)
	} else {
		rows = append(rows,
			[]gotgbot.InlineKeyboardButton{{Text: "返回抽奖中心", CallbackData: CallbackData("private", "lottery_active")}},
			[]gotgbot.InlineKeyboardButton{{Text: "创建新抽奖", CallbackData: CallbackData("private", "lottery_create")}},
		)
	}
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}}
}

func lotteryHasJoinButton(joinType string) bool {
	joinType = strings.ToLower(strings.TrimSpace(joinType))
	return joinType == "" || joinType == "button" || joinType == "both"
}

func lotteryWasAutoDrawn(message string) bool {
	return strings.Contains(message, "已自动开奖")
}

func (a *App) announceLotteryResult(b *gotgbot.Bot, ctx *ext.Context, lotteryID int64) error {
	scope := requestScope(ctx)
	if a.services.Lottery == nil {
		return nil
	}
	info, err := a.services.Lottery.Info(scope.Context, scope.Chat.ID, lotteryID)
	if err != nil {
		return err
	}
	_, err = b.SendMessageWithContext(scope.Context, scope.Chat.ID, "抽奖已开奖\n\n"+info, nil)
	return err
}
