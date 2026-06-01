package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/dabowin/sola/internal/model"
	"gorm.io/gorm"
)

func (a *App) handleBans(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Admin == nil {
		return sendText(b, ctx, "封禁记录服务尚未接入。", nil)
	}
	bans, err := a.services.Admin.ListBans(scope.Context, scope.Chat.ID, 10)
	if err != nil {
		return err
	}
	if len(bans) == 0 {
		return sendText(b, ctx, "本群暂无封禁记录。", nil)
	}
	var builder strings.Builder
	builder.WriteString("本群封禁记录：")
	for _, ban := range bans {
		status := "封禁中"
		if ban.UnbannedAt != nil {
			status = "已解封"
		}
		reason := strings.TrimSpace(ban.Reason)
		if reason == "" {
			reason = "无"
		}
		builder.WriteString(fmt.Sprintf("\n#%d [%s] 用户 %d，原因：%s", ban.ID, status, ban.UserID, truncateRunes(reason, 36)))
	}
	builder.WriteString("\n\n解封：/unban 用户ID")
	return sendText(b, ctx, builder.String(), nil)
}

func (a *App) handleViolations(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Violations == nil {
		return sendText(b, ctx, "违规记录服务尚未接入。", nil)
	}
	userID := int64(0)
	if args := commandArgs(ctx); len(args) > 0 && !strings.EqualFold(args[0], "all") {
		parsed, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
		if err != nil || parsed == 0 {
			return sendText(b, ctx, "用法：/violations [用户ID]", nil)
		}
		userID = parsed
	}
	records, err := a.services.Violations.ListViolations(scope.Context, scope.Chat.ID, userID, 10, 0)
	if err != nil {
		return err
	}
	text := formatViolationRecords(records)
	text += "\n\n处理：/resolve_violation ID [备注]\n忽略：/ignore_violation ID [备注]"
	return respondText(b, ctx, strings.Join([]string{"违规记录面板", "", text}, "\n"), moderationAssetsMarkup())
}

func (a *App) handleResolveViolation(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.updateViolationFromCommand(b, ctx, "resolved", "resolved_by_bot_command", "已处理违规记录。")
}

func (a *App) handleIgnoreViolation(b *gotgbot.Bot, ctx *ext.Context) error {
	return a.updateViolationFromCommand(b, ctx, "resolved", "ignored_by_bot_command", "已忽略并归档违规记录。")
}

func (a *App) updateViolationFromCommand(b *gotgbot.Bot, ctx *ext.Context, statusValue string, defaultResolution string, doneText string) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Violations == nil {
		return sendText(b, ctx, "违规记录服务尚未接入。", nil)
	}
	args := commandArgs(ctx)
	if len(args) < 1 {
		return sendText(b, ctx, "用法：/resolve_violation ID [备注] 或 /ignore_violation ID [备注]", nil)
	}
	id, err := a.resolveViolationID(scope.Context, scope.Chat.ID, args[0])
	if err != nil {
		return sendText(b, ctx, "没有找到这个违规记录，请用 /violations 查看 ID 或序号。", nil)
	}
	resolution := strings.TrimSpace(strings.Join(args[1:], " "))
	if resolution == "" {
		resolution = defaultResolution
	}
	status := statusValue
	record, err := a.services.Violations.UpdateViolation(scope.Context, id, &status, &resolution)
	if err != nil {
		return err
	}
	return sendText(b, ctx, fmt.Sprintf("%s\nID：%s\n用户：%d\n结果：%s", doneText, record.ID.String(), record.UserID, resolution), nil)
}

func (a *App) handleLevels(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Level == nil {
		return sendText(b, ctx, "等级规则服务尚未接入。", nil)
	}
	text, err := a.services.Level.ListLevelRules(scope.Context, scope.Chat.ID)
	if err != nil {
		return err
	}
	text += "\n\n添加/修改：/add_level 2 100 活跃 Lv.2\n删除：/del_level 2"
	return respondText(b, ctx, strings.Join([]string{"违规记录面板", "", text}, "\n"), moderationAssetsMarkup())
}

func (a *App) handleAddLevel(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Level == nil {
		return sendText(b, ctx, "等级规则服务尚未接入。", nil)
	}
	args := commandArgs(ctx)
	if len(args) < 3 {
		return sendText(b, ctx, "用法：/add_level 等级 最低积分 名称 [徽章]\n示例：/add_level 2 100 活跃 Lv.2", nil)
	}
	level, err := strconv.Atoi(args[0])
	if err != nil || level <= 0 {
		return sendText(b, ctx, "等级必须是大于 0 的整数。", nil)
	}
	minPoints, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil || minPoints < 0 {
		return sendText(b, ctx, "最低积分必须是 >= 0 的整数。", nil)
	}
	name := strings.TrimSpace(args[2])
	badge := fmt.Sprintf("Lv.%d", level)
	if len(args) > 3 {
		badge = strings.TrimSpace(strings.Join(args[3:], " "))
	}
	text, err := a.services.Level.UpsertLevelRule(scope.Context, scope.Chat.ID, level, name, minPoints, badge)
	if err != nil {
		return err
	}
	return respondText(b, ctx, strings.Join([]string{"违规记录面板", "", text}, "\n"), moderationAssetsMarkup())
}

func (a *App) handleDelLevel(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Level == nil {
		return sendText(b, ctx, "等级规则服务尚未接入。", nil)
	}
	args := commandArgs(ctx)
	if len(args) < 1 {
		return sendText(b, ctx, "用法：/del_level 等级\n示例：/del_level 2", nil)
	}
	level, err := strconv.Atoi(args[0])
	if err != nil || level <= 0 {
		return sendText(b, ctx, "等级必须是大于 0 的整数。", nil)
	}
	text, err := a.services.Level.DeleteLevelRule(scope.Context, scope.Chat.ID, level)
	if err != nil {
		return err
	}
	return respondText(b, ctx, strings.Join([]string{"违规记录面板", "", text}, "\n"), moderationAssetsMarkup())
}

func (a *App) handleTemplates(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Templates == nil {
		return sendText(b, ctx, "消息模板服务尚未接入。", nil)
	}
	templates, err := a.services.Templates.ListForChat(scope.Context, scope.Chat.ID, 20)
	if err != nil {
		return err
	}
	if len(templates) == 0 {
		return sendText(b, ctx, "本群暂无消息模板。\n添加：/add_template 标题 | 内容", nil)
	}
	var builder strings.Builder
	builder.WriteString("本群消息模板：")
	for i, item := range templates {
		builder.WriteString(fmt.Sprintf("\n%d. %s [%s]\nID：%s\n%s", i+1, item.Name, templateMediaLabel(item.MediaType), item.ID, truncateRunes(item.Content, 64)))
	}
	builder.WriteString("\n\n添加：/add_template 标题 | 内容\n删除：/del_template ID 或序号")
	return sendText(b, ctx, builder.String(), nil)
}

func (a *App) handleAddTemplate(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Templates == nil {
		return sendText(b, ctx, "消息模板服务尚未接入。", nil)
	}
	raw := strings.TrimSpace(strings.Join(commandArgs(ctx), " "))
	name, content, ok := parseNameContent(raw)
	if !ok {
		return sendText(b, ctx, "用法：/add_template 标题 | 内容", nil)
	}
	record, err := a.services.Templates.CreateForBot(scope.Context, MessageTemplateCreate{
		ChatID:    scope.Chat.ID,
		Name:      name,
		Content:   content,
		MediaType: "text",
		ParseMode: "HTML",
		CreatedBy: scope.Actor.ID,
	})
	if err != nil {
		return err
	}
	return sendText(b, ctx, fmt.Sprintf("已添加消息模板：%s\nID：%s", record.Name, record.ID), nil)
}

func (a *App) handleDelTemplate(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Templates == nil {
		return sendText(b, ctx, "消息模板服务尚未接入。", nil)
	}
	rawID := strings.TrimSpace(strings.Join(commandArgs(ctx), " "))
	if rawID == "" {
		return sendText(b, ctx, "用法：/del_template ID 或序号", nil)
	}
	id, err := a.resolveTemplateID(scope.Context, scope.Chat.ID, rawID)
	if err != nil {
		return sendText(b, ctx, "没有找到这个模板，请用 /templates 查看 ID 或序号。", nil)
	}
	if err := a.services.Templates.DeleteForChat(scope.Context, scope.Chat.ID, id); err != nil {
		return err
	}
	return sendText(b, ctx, "已删除消息模板。", nil)
}

func (a *App) handleInvites(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.InviteLink == nil {
		return sendText(b, ctx, "邀请链接服务尚未接入。", nil)
	}
	text, err := a.services.InviteLink.ListForChat(scope.Context, scope.Chat.ID, 20)
	if err != nil {
		return err
	}
	text += "\n\n创建：/invite_create 名称\n删除：/invite_delete ID 或序号"
	return respondText(b, ctx, strings.Join([]string{"邀请链接面板", "", text}, "\n"), moderationAssetsMarkup())
}

func (a *App) handleInviteCreate(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.InviteLink == nil {
		return sendText(b, ctx, "邀请链接服务尚未接入。", nil)
	}
	name := strings.TrimSpace(strings.Join(commandArgs(ctx), " "))
	if name == "" {
		name = "Bot 创建"
	}
	record, err := a.services.InviteLink.CreateForBot(scope.Context, scope.Chat.ID, name, false, scope.Actor.ID)
	if err != nil {
		return err
	}
	return sendText(b, ctx, fmt.Sprintf("已创建邀请链接：%s\nID：%s\n%s", record.Name, record.ID, record.InviteLink), nil)
}

func (a *App) handleInviteDelete(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.InviteLink == nil {
		return sendText(b, ctx, "邀请链接服务尚未接入。", nil)
	}
	rawID := strings.TrimSpace(strings.Join(commandArgs(ctx), " "))
	if rawID == "" {
		return sendText(b, ctx, "用法：/invite_delete ID 或序号", nil)
	}
	if err := a.services.InviteLink.DeleteForChat(scope.Context, scope.Chat.ID, rawID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return sendText(b, ctx, "没有找到这个邀请链接，请用 /invites 查看 ID 或序号。", nil)
		}
		return err
	}
	return sendText(b, ctx, "已删除并撤销邀请链接。", nil)
}

func parseNameContent(raw string) (string, string, bool) {
	parts := strings.SplitN(raw, "|", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	name := strings.TrimSpace(parts[0])
	content := strings.TrimSpace(parts[1])
	return name, content, name != "" && content != ""
}

func (a *App) resolveTemplateID(ctx context.Context, chatID int64, raw string) (string, error) {
	templates, err := a.services.Templates.ListForChat(ctx, chatID, 100)
	if err != nil {
		return "", err
	}
	needle := strings.TrimSpace(raw)
	if index, err := strconv.Atoi(needle); err == nil && index > 0 && index <= len(templates) {
		return templates[index-1].ID, nil
	}
	for _, item := range templates {
		if item.ID == needle || strings.HasPrefix(item.ID, needle) || strings.EqualFold(item.Name, needle) {
			return item.ID, nil
		}
	}
	return "", gorm.ErrRecordNotFound
}

func (a *App) resolveViolationID(ctx context.Context, chatID int64, raw string) (string, error) {
	records, err := a.services.Violations.ListViolations(ctx, chatID, 0, 100, 0)
	if err != nil {
		return "", err
	}
	needle := strings.TrimSpace(raw)
	if index, err := strconv.Atoi(needle); err == nil && index > 0 && index <= len(records) {
		return records[index-1].ID.String(), nil
	}
	for _, item := range records {
		id := item.ID.String()
		if id == needle || strings.HasPrefix(id, needle) {
			return id, nil
		}
	}
	return "", gorm.ErrRecordNotFound
}

func formatViolationRecords(records []model.ViolationRecord) string {
	if len(records) == 0 {
		return "本群暂无违规记录。"
	}
	var builder strings.Builder
	builder.WriteString("本群违规记录：")
	for i, item := range records {
		status := "未处理"
		if item.Cleared {
			status = "已处理"
		}
		id := item.ID.String()
		shortID := id
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		reason := strings.TrimSpace(item.DetectedBy)
		if reason == "" {
			reason = "规则命中"
		}
		builder.WriteString(fmt.Sprintf(
			"\n%d. %s [%s] 用户 %d，类型：%s，动作：%s，来源：%s，时间：%s",
			i+1,
			shortID,
			status,
			item.UserID,
			item.ViolationType,
			item.ActionTaken,
			truncateRunes(reason, 24),
			item.CreatedAt.Format("01-02 15:04"),
		))
		if strings.TrimSpace(item.MessageText) != "" {
			builder.WriteString("\n   " + truncateRunes(item.MessageText, 48))
		}
	}
	return builder.String()
}

func templateMediaLabel(mediaType string) string {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "photo":
		return "图片"
	case "video":
		return "视频"
	default:
		return "文本"
	}
}
