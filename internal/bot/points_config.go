package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func (a *App) handlePointsConfig(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if a.services.Points == nil {
		return sendText(b, ctx, "积分配置服务尚未接入。", nil)
	}
	cfg, err := a.services.Points.GetConfig(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	return respondText(b, ctx, formatPointConfig(cfg), pointsConfigMarkup(cfg))
}

func (a *App) handleSetPoints(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.updatePointConfigField(b, ctx, "point")
}

func (a *App) handleSetCooldown(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if a.services.Points == nil {
		return sendText(b, ctx, "积分配置服务尚未接入。", nil)
	}
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}

	args := commandArgs(ctx)
	if len(args) < 1 {
		return sendText(b, ctx, "用法：/set_cooldown 30", nil)
	}
	seconds, err := strconv.Atoi(args[0])
	if err != nil || seconds < 0 {
		return sendText(b, ctx, "冷却时间必须是 >= 0 的整数秒。", nil)
	}

	cfg, err := a.services.Points.UpdateConfig(scope.Context, scope.Chat.ID, ChatPointConfigPatch{CooldownSeconds: &seconds})
	if err != nil {
		return err
	}
	return respondText(b, ctx, fmt.Sprintf("已更新防刷间隔：%d 秒\n%s", seconds, formatPointConfig(cfg)), pointsConfigMarkup(cfg))
}

func (a *App) handlePointsToggle(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if a.services.Points == nil {
		return sendText(b, ctx, "积分配置服务尚未接入。", nil)
	}
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}

	cfg, err := a.services.Points.ToggleConfig(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	state := "已关闭"
	if cfg.Enabled {
		state = "已开启"
	}
	return respondText(b, ctx, fmt.Sprintf("积分系统%s。\n%s", state, formatPointConfig(cfg)), pointsConfigMarkup(cfg))
}

func (a *App) updatePointConfigField(b *gotgbot.Bot, ctx *ext.Context, prefix string) error {
	scope := requestScope(ctx)
	if a.services.Points == nil {
		return sendText(b, ctx, "积分配置服务尚未接入。", nil)
	}
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}

	args := commandArgs(ctx)
	if len(args) < 2 {
		return sendText(b, ctx, "用法：/set_points text 2", nil)
	}

	field := strings.ToLower(args[0])
	value, err := strconv.Atoi(args[1])
	if err != nil || value < 0 {
		return sendText(b, ctx, "分值必须是 >= 0 的整数。", nil)
	}

	cfg, err := a.updatePointField(scope.Context, scope.Chat.ID, field, value)
	if err != nil {
		return sendText(b, ctx, "支持字段：text photo sticker video file voice", nil)
	}
	return respondText(b, ctx, fmt.Sprintf("已更新 %s 分值为 %d\n%s", field, value, formatPointConfig(cfg)), pointsConfigMarkup(cfg))
}

func (a *App) updatePointField(ctx context.Context, chatID int64, field string, value int) (ChatPointConfig, error) {
	patch := ChatPointConfigPatch{}
	switch strings.ToLower(field) {
	case "text":
		patch.PointText = &value
	case "photo":
		patch.PointPhoto = &value
	case "sticker":
		patch.PointSticker = &value
	case "video":
		patch.PointVideo = &value
	case "file":
		patch.PointFile = &value
	case "voice":
		patch.PointVoice = &value
	default:
		return ChatPointConfig{}, fmt.Errorf("unsupported point field")
	}
	return a.services.Points.UpdateConfig(ctx, chatID, patch)
}

func formatPointConfig(cfg ChatPointConfig) string {
	return fmt.Sprintf(
		"积分配置\nChat ID：%d\n状态：%s\n文字：%d\n图片：%d\n表情贴纸：%d\n视频：%d\n文件：%d\n语音：%d\n冷却：%d 秒",
		cfg.ChatID,
		boolLabel(cfg.Enabled, "开启", "关闭"),
		cfg.PointText,
		cfg.PointPhoto,
		cfg.PointSticker,
		cfg.PointVideo,
		cfg.PointFile,
		cfg.PointVoice,
		cfg.CooldownSeconds,
	)
}

func commandArgs(ctx *ext.Context) []string {
	if ctx == nil || ctx.Message == nil {
		return nil
	}
	text := strings.TrimSpace(ctx.Message.Text)
	if text == "" {
		return nil
	}
	parts := strings.Fields(text)
	if len(parts) <= 1 {
		return nil
	}
	return parts[1:]
}

func boolLabel(v bool, yes string, no string) string {
	if v {
		return yes
	}
	return no
}

func (a *App) requireTelegramManager(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if scope.Chat.Type == "" || scope.Chat.Type == "private" {
		return sendText(b, ctx, "请在群组或频道里操作。", nil)
	}
	if a.services.TelegramAccess == nil {
		return sendText(b, ctx, "Telegram 权限检查服务尚未接入。", nil)
	}
	if scope.Actor.ID == 0 {
		return sendText(b, ctx, "无法识别当前操作者。", nil)
	}
	ok, err := a.isTelegramManager(b, ctx)
	if err != nil {
		return err
	}
	if !ok {
		return sendText(b, ctx, "需要群主或管理员权限。", nil)
	}
	return nil
}

func (a *App) isTelegramManager(b *gotgbot.Bot, ctx *ext.Context) (bool, error) {
	scope := requestScope(ctx)
	if scope.Chat.Type == "" || scope.Chat.Type == "private" || scope.Actor.ID == 0 || a.services.TelegramAccess == nil {
		return false, nil
	}
	status, err := a.services.TelegramAccess.CheckUserAdmin(scope.Context, b, scope.Chat.ID, scope.Actor.ID)
	if err != nil {
		return false, err
	}
	return status.IsAdmin || status.Status == "creator", nil
}
