package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

const (
	postWizardStepSchedule = 1
	postWizardStepContent  = 2
	postWizardStepDelete   = 3
	postWizardStepConfirm  = 99
)

func (a *App) startCreatePostWizard(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.startCreatePostWizardWithTarget(b, ctx, "", requestScope(ctx).Chat.ID)
}

func (a *App) startCreatePostWizardWithMode(b *gotgbot.Bot, ctx *ext.Context, mode string) error {
	return a.startCreatePostWizardWithTarget(b, ctx, mode, requestScope(ctx).Chat.ID)
}

func (a *App) startCreatePostWizardWithTarget(b *gotgbot.Bot, ctx *ext.Context, mode string, chatID int64) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Publish == nil {
		return respondText(b, ctx, "发布服务尚未接入。", nil)
	}
	state := &ConversationState{
		Type:   "create_post",
		Step:   postWizardStepSchedule,
		ChatID: chatID,
		Data:   map[string]any{},
	}
	if mode == "once" || mode == "repeat" {
		state.Data["mode"] = mode
	}
	if err := a.setConversation(scope.Context, scope.Actor.ID, state); err != nil {
		return err
	}
	if mode == "once" {
		return respondText(b, ctx, "创建一次性提醒\n\n先选一个快捷时间，也可以稍后直接输入 2026-06-01 20:30 这种完整时间。", postOnceMarkup())
	}
	if mode == "repeat" {
		return respondText(b, ctx, "创建循环发布\n\n先选一个常用周期，也可以后面直接输入 Cron 表达式。", postRepeatMarkup())
	}
	return respondText(b, ctx, "创建定时发帖\n\n先选任务类型，再继续设置频率、文案和自动删除。", postModeMarkup())
}

func (a *App) handlePostWizardStep(b *gotgbot.Bot, ctx *ext.Context, state *ConversationState) error {
	scope := requestScope(ctx)
	input := ""
	if ctx != nil && ctx.Message != nil {
		input = strings.TrimSpace(ctx.Message.Text)
	}
	if strings.EqualFold(input, "/cancel") || strings.EqualFold(input, "cancel") || input == "取消" {
		a.clearConversation(scope.Context, scope.Actor.ID)
		return sendText(b, ctx, "已取消创建定时发帖。", nil)
	}

	switch state.Step {
	case postWizardStepSchedule:
		if err := applyPostScheduleInput(state, input); err != nil {
			return sendText(b, ctx, err.Error(), postScheduleValueMarkup(stringVal(state.Data, "mode"), stringVal(state.Data, "schedule_kind")))
		}
		state.Step = postWizardStepContent
		return a.saveWizardAndSend(b, ctx, state, postWizardTitle(state)+"\n\n请输入要发送的内容，支持 HTML：", cancelMarkup())
	case postWizardStepContent:
		if input == "" {
			return sendText(b, ctx, "内容不能为空，请重新输入：", cancelMarkup())
		}
		state.Data["content"] = input
		state.Step = postWizardStepDelete
		return a.saveWizardAndSend(b, ctx, state, postWizardTitle(state)+"\n\n自动删除秒数？输入 0 表示不自动删除。", postAutoDeleteMarkup())
	case postWizardStepDelete:
		seconds, err := parseWizardNonNegativeInt(input)
		if err != nil {
			return sendText(b, ctx, "请输入非负整数秒数：", postAutoDeleteMarkup())
		}
		state.Data["auto_delete_seconds"] = seconds
		return a.confirmPostWizard(b, ctx, state)
	case postWizardStepConfirm:
		return sendText(b, ctx, "请点击确认创建或取消。", confirmPostMarkup())
	default:
		a.clearConversation(scope.Context, scope.Actor.ID)
		return sendText(b, ctx, "定时发帖向导已失效，请重新开始。", nil)
	}
}

func (a *App) routePostWizardCallback(b *gotgbot.Bot, ctx *ext.Context, state *ConversationState, payload CallbackPayload) error {
	switch payload.Action {
	case "post_mode":
		mode := strings.TrimSpace(payload.Resource)
		if mode != "once" && mode != "repeat" {
			return respondText(b, ctx, "任务类型无效，请重新选择：", postModeMarkup())
		}
		state.Data["mode"] = mode
		state.Step = postWizardStepSchedule
		if mode == "once" {
			return a.saveWizardAndRespond(b, ctx, state, "创建一次性提醒\n\n先选一个快捷时间，也可以直接输入完整时间。", postOnceMarkup())
		}
		return a.saveWizardAndRespond(b, ctx, state, "创建循环发布\n\n先选一个常用周期，也可以后续输入更细的规则。", postRepeatMarkup())
	case "post_once":
		return a.applyPostScheduleFromCallback(b, ctx, state, "once", payload.Resource)
	case "post_repeat":
		if done, err := applyPresetRepeatSchedule(state, payload.Resource); done {
			if err != nil {
				return respondText(b, ctx, err.Error(), postRepeatMarkup())
			}
			state.Step = postWizardStepContent
			return a.saveWizardAndRespond(b, ctx, state, postWizardTitle(state)+"\n\n请输入要发送的内容，支持 HTML：", cancelMarkup())
		}
		return a.applyPostScheduleFromCallback(b, ctx, state, "repeat", payload.Resource)
	case "post_schedule_minute":
		minute, err := parseWizardNonNegativeInt(payload.Resource)
		if err != nil || minute > 59 {
			return respondText(b, ctx, "分钟必须是 0-59，请重新选择：", postScheduleValueMarkup("repeat", "hourly"))
		}
		state.Data["cron_expr"] = fmt.Sprintf("%d * * * *", minute)
		state.Data["run_once_at"] = ""
		state.Data["mode"] = "repeat"
		state.Data["schedule_kind"] = "hourly"
		state.Step = postWizardStepContent
		return a.saveWizardAndRespond(b, ctx, state, postWizardTitle(state)+"\n\n请输入要发送的内容，支持 HTML：", cancelMarkup())
	case "post_auto_delete":
		seconds, err := parseWizardNonNegativeInt(payload.Resource)
		if err != nil {
			return respondText(b, ctx, "自动删除秒数无效，请重新输入：", postAutoDeleteMarkup())
		}
		state.Data["auto_delete_seconds"] = seconds
		return a.confirmPostWizard(b, ctx, state)
	case "post_confirm":
		if payload.Resource == "no" {
			scope := requestScope(ctx)
			a.clearConversation(scope.Context, scope.Actor.ID)
			return respondText(b, ctx, "已取消创建定时发帖。", nil)
		}
		return a.confirmCreatePost(b, ctx, state)
	default:
		return answerCallback(b, ctx, "未知操作")
	}
}

func (a *App) applyPostScheduleFromCallback(b *gotgbot.Bot, ctx *ext.Context, state *ConversationState, mode string, value string) error {
	state.Data["mode"] = mode
	switch mode + ":" + value {
	case "once:2m", "once:10m", "once:1h":
		duration, _ := time.ParseDuration(value)
		runAt := time.Now().Add(duration)
		state.Data["run_once_at"] = runAt.Format(time.RFC3339)
		state.Data["cron_expr"] = ""
		state.Step = postWizardStepContent
		return a.saveWizardAndRespond(b, ctx, state, postWizardTitle(state)+"\n\n请输入要发送的内容，支持 HTML：", cancelMarkup())
	case "once:custom":
		state.Data["schedule_kind"] = "once_custom"
		state.Step = postWizardStepSchedule
		return a.saveWizardAndRespond(b, ctx, state, "请输入发送时间，格式 2026-06-01 20:00；也可以输入 2m / 10m / 1h：", cancelMarkup())
	case "repeat:30s":
		state.Data["cron_expr"] = "@every 30s"
	case "repeat:5m":
		state.Data["cron_expr"] = "@every 5m"
	case "repeat:hourly":
		state.Data["schedule_kind"] = "hourly"
		state.Step = postWizardStepSchedule
		return a.saveWizardAndRespond(b, ctx, state, "请输入每小时第几分钟发送（0-59）：", postScheduleValueMarkup("repeat", "hourly"))
	case "repeat:daily":
		state.Data["schedule_kind"] = "daily"
		state.Step = postWizardStepSchedule
		return a.saveWizardAndRespond(b, ctx, state, "请输入每天发送时间，格式 HH:mm，例如 09:30：", postScheduleValueMarkup("repeat", "daily"))
	case "repeat:cron":
		state.Data["schedule_kind"] = "cron"
		state.Step = postWizardStepSchedule
		return a.saveWizardAndRespond(b, ctx, state, "请输入 Cron 表达式，例如 0 9 * * *：", cancelMarkup())
	default:
		return respondText(b, ctx, "调度选项无效，请重新选择。", postModeMarkup())
	}
	state.Data["run_once_at"] = ""
	state.Step = postWizardStepContent
	return a.saveWizardAndRespond(b, ctx, state, postWizardTitle(state)+"\n\n请输入要发送的内容，支持 HTML：", cancelMarkup())
}

func applyPostScheduleInput(state *ConversationState, input string) error {
	kind := stringVal(state.Data, "schedule_kind")
	switch kind {
	case "once_custom":
		if runAt, err := parsePostRunAt(input); err == nil && runAt.After(time.Now()) {
			state.Data["run_once_at"] = runAt.Format(time.RFC3339)
			state.Data["cron_expr"] = ""
			return nil
		}
		return fmt.Errorf("时间无效，请输入未来时间，例如 2026-06-01 20:00，或 10m")
	case "hourly":
		minute, err := strconv.Atoi(input)
		if err != nil || minute < 0 || minute > 59 {
			return fmt.Errorf("分钟必须是 0-59")
		}
		state.Data["cron_expr"] = fmt.Sprintf("%d * * * *", minute)
		state.Data["run_once_at"] = ""
		return nil
	case "daily":
		parts := strings.Split(input, ":")
		if len(parts) != 2 {
			return fmt.Errorf("时间格式必须是 HH:mm，例如 09:30")
		}
		hour, err1 := strconv.Atoi(parts[0])
		minute, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
			return fmt.Errorf("时间格式必须是 HH:mm，例如 09:30")
		}
		state.Data["cron_expr"] = fmt.Sprintf("%d %d * * *", minute, hour)
		state.Data["run_once_at"] = ""
		return nil
	case "cron":
		cron := strings.TrimSpace(input)
		if cron == "" {
			return fmt.Errorf("Cron 表达式不能为空")
		}
		state.Data["cron_expr"] = cron
		state.Data["run_once_at"] = ""
		return nil
	default:
		return fmt.Errorf("请先通过按钮选择发送时间或循环周期。")
	}
}

func (a *App) confirmPostWizard(b *gotgbot.Bot, ctx *ext.Context, state *ConversationState) error {
	scope := requestScope(ctx)
	text := fmt.Sprintf(
		"请确认定时发帖：\n\n类型：%s\n计划：%s\n自动删除：%d 秒\n内容：%s",
		postModeLabel(stringVal(state.Data, "mode")),
		postScheduleText(state),
		intVal(state.Data, "auto_delete_seconds"),
		truncateRunes(stringVal(state.Data, "content"), 120),
	)
	state.Step = postWizardStepConfirm
	if err := a.setConversation(scope.Context, scope.Actor.ID, state); err != nil {
		return err
	}
	return respondText(b, ctx, text, confirmPostMarkup())
}

func (a *App) confirmCreatePost(b *gotgbot.Bot, ctx *ext.Context, state *ConversationState) error {
	scope := requestScope(ctx)
	a.clearConversation(scope.Context, scope.Actor.ID)
	var runAt *time.Time
	if raw := stringVal(state.Data, "run_once_at"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return respondText(b, ctx, "创建失败：发送时间无效。", nil)
		}
		runAt = &parsed
	}
	post, err := a.services.Publish.CreateScheduledPost(scope.Context, ScheduledPostCreate{
		ChatID:            state.ChatID,
		Title:             "Bot 创建",
		Content:           stringVal(state.Data, "content"),
		MediaType:         "text",
		CronExpr:          stringVal(state.Data, "cron_expr"),
		RunOnceAt:         runAt,
		Enabled:           true,
		AutoDeleteSeconds: intVal(state.Data, "auto_delete_seconds"),
		CreatedBy:         scope.Actor.ID,
	})
	if err != nil {
		return respondText(b, ctx, "创建失败："+err.Error(), nil)
	}
	return respondText(b, ctx, "定时任务已创建：\n"+formatScheduledPostLine(post), postListMarkup([]ScheduledPostItem{post}))
}

func applyPresetRepeatSchedule(state *ConversationState, preset string) (bool, error) {
	preset = strings.TrimSpace(strings.ToLower(preset))
	now := time.Now()
	switch preset {
	case "daily_09":
		state.Data["mode"] = "repeat"
		state.Data["schedule_kind"] = "daily"
		state.Data["cron_expr"] = "0 9 * * *"
		state.Data["run_once_at"] = ""
		return true, nil
	case "daily_12":
		state.Data["mode"] = "repeat"
		state.Data["schedule_kind"] = "daily"
		state.Data["cron_expr"] = "0 12 * * *"
		state.Data["run_once_at"] = ""
		return true, nil
	case "daily_18":
		state.Data["mode"] = "repeat"
		state.Data["schedule_kind"] = "daily"
		state.Data["cron_expr"] = "0 18 * * *"
		state.Data["run_once_at"] = ""
		return true, nil
	case "daily_22":
		state.Data["mode"] = "repeat"
		state.Data["schedule_kind"] = "daily"
		state.Data["cron_expr"] = "0 22 * * *"
		state.Data["run_once_at"] = ""
		return true, nil
	case "weekly_mon":
		state.Data["mode"] = "repeat"
		state.Data["schedule_kind"] = "weekly"
		state.Data["cron_expr"] = "0 9 * * 1"
		state.Data["run_once_at"] = ""
		return true, nil
	case "weekly_wed":
		state.Data["mode"] = "repeat"
		state.Data["schedule_kind"] = "weekly"
		state.Data["cron_expr"] = "0 12 * * 3"
		state.Data["run_once_at"] = ""
		return true, nil
	case "weekly_fri":
		state.Data["mode"] = "repeat"
		state.Data["schedule_kind"] = "weekly"
		state.Data["cron_expr"] = "0 18 * * 5"
		state.Data["run_once_at"] = ""
		return true, nil
	case "monthly_1":
		state.Data["mode"] = "repeat"
		state.Data["schedule_kind"] = "monthly"
		state.Data["cron_expr"] = "0 9 1 * *"
		state.Data["run_once_at"] = ""
		return true, nil
	case "monthly_15":
		state.Data["mode"] = "repeat"
		state.Data["schedule_kind"] = "monthly"
		state.Data["cron_expr"] = "0 12 15 * *"
		state.Data["run_once_at"] = ""
		return true, nil
	case "monthly_last":
		state.Data["mode"] = "repeat"
		state.Data["schedule_kind"] = "monthly"
		state.Data["cron_expr"] = "0 20 L * *"
		state.Data["run_once_at"] = ""
		return true, nil
	case "today20":
		runAt := time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, now.Location())
		if !runAt.After(now) {
			runAt = runAt.Add(24 * time.Hour)
		}
		state.Data["mode"] = "once"
		state.Data["schedule_kind"] = "once"
		state.Data["run_once_at"] = runAt.Format(time.RFC3339)
		state.Data["cron_expr"] = ""
		return true, nil
	case "tomorrow9":
		runAt := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location()).Add(24 * time.Hour)
		state.Data["mode"] = "once"
		state.Data["schedule_kind"] = "once"
		state.Data["run_once_at"] = runAt.Format(time.RFC3339)
		state.Data["cron_expr"] = ""
		return true, nil
	default:
		return false, nil
	}
}

func parsePostRunAt(input string) (time.Time, error) {
	text := strings.TrimSpace(input)
	if duration, err := time.ParseDuration(text); err == nil && duration > 0 {
		return time.Now().Add(duration), nil
	}
	return time.ParseInLocation("2006-01-02 15:04", text, time.Local)
}

func postWizardTitle(state *ConversationState) string {
	return "创建" + postModeLabel(stringVal(state.Data, "mode"))
}

func postModeLabel(mode string) string {
	if mode == "repeat" {
		return "循环发布"
	}
	return "一次性提醒"
}

func postScheduleText(state *ConversationState) string {
	if raw := stringVal(state.Data, "run_once_at"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			return parsed.Format("2006-01-02 15:04")
		}
	}
	if cron := strings.TrimSpace(stringVal(state.Data, "cron_expr")); cron != "" {
		return cron
	}
	return "未设置"
}

func postModeMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "一次提醒", CallbackData: CallbackData("wizard", "post_mode", "once")},
			{Text: "循环发布", CallbackData: CallbackData("wizard", "post_mode", "repeat")},
		},
		cancelRow(),
	}}}
}

func postOnceMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "30秒后", CallbackData: CallbackData("wizard", "post_once", "30s")},
			{Text: "2分钟后", CallbackData: CallbackData("wizard", "post_once", "2m")},
		},
		{
			{Text: "10分钟后", CallbackData: CallbackData("wizard", "post_once", "10m")},
			{Text: "1小时后", CallbackData: CallbackData("wizard", "post_once", "1h")},
		},
		{
			{Text: "今晚 20:00", CallbackData: CallbackData("wizard", "post_once", "today20")},
			{Text: "明早 09:00", CallbackData: CallbackData("wizard", "post_once", "tomorrow9")},
		},
		{
			{Text: "自定义时间", CallbackData: CallbackData("wizard", "post_once", "custom")},
		},
		cancelRow(),
	}}}
}

func postRepeatMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "每30秒", CallbackData: CallbackData("wizard", "post_repeat", "30s")},
			{Text: "每5分钟", CallbackData: CallbackData("wizard", "post_repeat", "5m")},
		},
		{
			{Text: "每小时", CallbackData: CallbackData("wizard", "post_repeat", "hourly")},
			{Text: "每天", CallbackData: CallbackData("wizard", "post_repeat", "daily")},
		},
		{
			{Text: "每周", CallbackData: CallbackData("wizard", "post_repeat", "weekly")},
			{Text: "每月", CallbackData: CallbackData("wizard", "post_repeat", "monthly")},
		},
		{
			{Text: "Cron 高级", CallbackData: CallbackData("wizard", "post_repeat", "cron")},
		},
		cancelRow(),
	}}}
}

func postScheduleValueMarkup(mode string, kind string) *gotgbot.SendMessageOpts {
	if mode == "repeat" && kind == "hourly" {
		return quickIntMarkup([]int{0, 15, 30, 45}, "wizard", "post_schedule_minute")
	}
	return postSchedulePickerMarkup(mode, kind)
}

func postAutoDeleteMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "不删除", CallbackData: CallbackData("wizard", "post_auto_delete", "0")},
			{Text: "30秒", CallbackData: CallbackData("wizard", "post_auto_delete", "30")},
		},
		{
			{Text: "1分钟", CallbackData: CallbackData("wizard", "post_auto_delete", "60")},
			{Text: "5分钟", CallbackData: CallbackData("wizard", "post_auto_delete", "300")},
		},
		cancelRow(),
	}}}
}

func confirmPostMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "确认创建", CallbackData: CallbackData("wizard", "post_confirm", "yes")},
			{Text: "取消", CallbackData: CallbackData("wizard", "post_confirm", "no")},
		},
	}}}
}

func postSchedulePickerMarkup(mode string, kind string) *gotgbot.SendMessageOpts {
	if mode == "repeat" {
		switch kind {
		case "hourly":
			return quickIntMarkup([]int{0, 15, 30, 45}, "wizard", "post_schedule_minute")
		case "daily":
			return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{{Text: "09:00", CallbackData: CallbackData("wizard", "post_repeat", "daily_09")}, {Text: "12:00", CallbackData: CallbackData("wizard", "post_repeat", "daily_12")}},
				{{Text: "18:00", CallbackData: CallbackData("wizard", "post_repeat", "daily_18")}, {Text: "22:00", CallbackData: CallbackData("wizard", "post_repeat", "daily_22")}},
				cancelRow(),
			}}}
		case "weekly":
			return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{{Text: "周一 09:00", CallbackData: CallbackData("wizard", "post_repeat", "weekly_mon")}},
				{{Text: "周三 12:00", CallbackData: CallbackData("wizard", "post_repeat", "weekly_wed")}},
				{{Text: "周五 18:00", CallbackData: CallbackData("wizard", "post_repeat", "weekly_fri")}},
				cancelRow(),
			}}}
		case "monthly":
			return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{{Text: "每月 1 号 09:00", CallbackData: CallbackData("wizard", "post_repeat", "monthly_1")}},
				{{Text: "每月 15 号 12:00", CallbackData: CallbackData("wizard", "post_repeat", "monthly_15")}},
				{{Text: "每月最后一天 20:00", CallbackData: CallbackData("wizard", "post_repeat", "monthly_last")}},
				cancelRow(),
			}}}
		}
	}
	return cancelMarkup()
}
