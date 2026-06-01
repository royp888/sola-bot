package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"github.com/dabowin/sola/internal/bootstrap"
	botapp "github.com/dabowin/sola/internal/bot"
	"github.com/dabowin/sola/internal/config"
	"github.com/dabowin/sola/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	botServices := service.NewBundle(nil, nil).BotServices()

	resources, err := bootstrap.New(ctx, "")
	if err != nil {
		log.Printf("bootstrap resources unavailable; starting bot without database/redis: %v", sanitizeTokenError(err, cfg.Bot.Token))
	} else {
		defer resources.Close(context.Background())
		cfg = resources.Config
		botServices = resources.BotServices()
	}

	token := strings.TrimSpace(cfg.Bot.Token)
	if token == "" {
		return fmt.Errorf("bot token is required: set SOLA_BOT_TOKEN or bot.token")
	}

	requestOpts := &gotgbot.RequestOpts{Timeout: 30 * time.Second}
	tgBot, err := gotgbot.NewBot(token, &gotgbot.BotOpts{
		BotClient: &gotgbot.BaseBotClient{
			Client:             http.Client{Timeout: 35 * time.Second},
			DefaultRequestOpts: requestOpts,
		},
		RequestOpts: requestOpts,
	})
	if err != nil {
		return fmt.Errorf("create telegram bot: %w", sanitizeTokenError(err, token))
	}

	me, err := tgBot.GetMe(nil)
	if err != nil {
		return fmt.Errorf("getMe: %w", sanitizeTokenError(err, token))
	}
	log.Printf("telegram bot connected: id=%d username=@%s", me.Id, me.Username)
	if err := registerBotCommands(ctx, tgBot); err != nil {
		log.Printf("register bot commands failed: %v", sanitizeTokenError(err, token))
	} else {
		log.Printf("telegram bot commands registered")
	}

	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(_ *gotgbot.Bot, _ *ext.Context, err error) ext.DispatcherAction {
			log.Printf("telegram handler error: %v", err)
			return ext.DispatcherActionNoop
		},
		UnhandledErrFunc: func(err error) {
			log.Printf("telegram dispatcher error: %v", err)
		},
	})

	app := botapp.New(botServices, botapp.Options{
		DefaultLocale: cfg.Bot.DefaultLocale,
	})
	app.Register(dispatcher)

	updater := ext.NewUpdater(dispatcher, &ext.UpdaterOpts{
		UnhandledErrFunc: func(err error) {
			log.Printf("telegram polling error: %v", sanitizeTokenError(err, token))
		},
	})

	if mode := strings.TrimSpace(cfg.Bot.Mode); mode != "" && !strings.EqualFold(mode, "polling") {
		log.Printf("bot.mode=%q configured; cmd/bot is starting polling", mode)
	}

	if err := updater.StartPolling(tgBot, &ext.PollingOpts{
		EnableWebhookDeletion: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout: 30,
			RequestOpts: &gotgbot.RequestOpts{
				Timeout: 45 * time.Second,
			},
			AllowedUpdates: []string{
				"message",
				"callback_query",
				"chat_member",
				"chat_join_request",
				"my_chat_member",
			},
		},
	}); err != nil {
		return fmt.Errorf("start polling: %w", sanitizeTokenError(err, token))
	}
	log.Printf("telegram polling started")

	<-ctx.Done()
	log.Printf("shutdown signal received")

	stopCtx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- updater.Stop()
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("stop updater: %w", sanitizeTokenError(err, token))
		}
	case <-stopCtx.Done():
		return fmt.Errorf("stop updater: %w", stopCtx.Err())
	}

	return nil
}

func sanitizeTokenError(err error, token string) error {
	if err == nil || token == "" {
		return err
	}
	return fmt.Errorf("%s", strings.ReplaceAll(err.Error(), token, "<redacted>"))
}

func registerBotCommands(ctx context.Context, tgBot *gotgbot.Bot) error {
	defaultCommands := []gotgbot.BotCommand{
		{Command: "start", Description: "开始菜单"},
		{Command: "menu", Description: "打开功能菜单"},
		{Command: "help", Description: "帮助"},
		{Command: "info", Description: "查看当前聊天信息"},
		{Command: "html", Description: "HTML 发布说明"},
		{Command: "cancel", Description: "取消当前操作"},
	}
	privateCommands := append([]gotgbot.BotCommand{}, defaultCommands...)
	privateCommands = append(privateCommands,
		gotgbot.BotCommand{Command: "bind", Description: "绑定/检查群或频道"},
		gotgbot.BotCommand{Command: "check_admin", Description: "检查 Bot 管理员权限"},
		gotgbot.BotCommand{Command: "publish", Description: "快捷发布"},
		gotgbot.BotCommand{Command: "posts", Description: "查看定时发帖"},
	)
	groupCommands := append([]gotgbot.BotCommand{}, defaultCommands...)
	groupCommands = append(groupCommands,
		gotgbot.BotCommand{Command: "bind", Description: "绑定/检查群管理权限"},
		gotgbot.BotCommand{Command: "check_admin", Description: "检查 Bot 管理员权限"},
		gotgbot.BotCommand{Command: "points", Description: "查询我的积分"},
		gotgbot.BotCommand{Command: "sign", Description: "签到领积分"},
		gotgbot.BotCommand{Command: "rank", Description: "积分排行榜"},
		gotgbot.BotCommand{Command: "points_rank", Description: "积分排行榜"},
		gotgbot.BotCommand{Command: "stat", Description: "今日统计"},
		gotgbot.BotCommand{Command: "stat_week", Description: "本周统计"},
		gotgbot.BotCommand{Command: "stats", Description: "自定义统计"},
		gotgbot.BotCommand{Command: "publish", Description: "快捷发布"},
		gotgbot.BotCommand{Command: "posts", Description: "查看定时发帖"},
		gotgbot.BotCommand{Command: "post_create", Description: "创建定时发帖"},
		gotgbot.BotCommand{Command: "lottery", Description: "抽奖创建/参与"},
	)
	adminCommands := append([]gotgbot.BotCommand{}, groupCommands...)
	adminCommands = append(adminCommands,
		gotgbot.BotCommand{Command: "points_config", Description: "查看积分配置"},
		gotgbot.BotCommand{Command: "set_points", Description: "设置消息分值"},
		gotgbot.BotCommand{Command: "set_cooldown", Description: "设置防刷冷却"},
		gotgbot.BotCommand{Command: "points_toggle", Description: "开启/关闭积分"},
		gotgbot.BotCommand{Command: "manage", Description: "回复用户打开管理面板"},
		gotgbot.BotCommand{Command: "mod", Description: "回复用户打开管理面板"},
		gotgbot.BotCommand{Command: "ban", Description: "封禁用户"},
		gotgbot.BotCommand{Command: "unban", Description: "解封用户"},
		gotgbot.BotCommand{Command: "mute", Description: "禁言用户"},
		gotgbot.BotCommand{Command: "unmute", Description: "解除禁言"},
		gotgbot.BotCommand{Command: "kick", Description: "踢出用户"},
		gotgbot.BotCommand{Command: "warn", Description: "警告用户"},
		gotgbot.BotCommand{Command: "unwarn", Description: "清除警告"},
		gotgbot.BotCommand{Command: "warns", Description: "查看警告"},
		gotgbot.BotCommand{Command: "adminconfig", Description: "群组管理配置"},
		gotgbot.BotCommand{Command: "set_welcome", Description: "设置欢迎语"},
		gotgbot.BotCommand{Command: "set_warn_limit", Description: "设置警告上限"},
		gotgbot.BotCommand{Command: "set_level", Description: "设置用户等级"},
		gotgbot.BotCommand{Command: "levels", Description: "查看等级规则"},
		gotgbot.BotCommand{Command: "add_level", Description: "添加等级规则"},
		gotgbot.BotCommand{Command: "del_level", Description: "删除等级规则"},
		gotgbot.BotCommand{Command: "verify_toggle", Description: "开关入群验证"},
		gotgbot.BotCommand{Command: "keywords", Description: "查看过滤关键词"},
		gotgbot.BotCommand{Command: "add_keyword", Description: "添加过滤关键词"},
		gotgbot.BotCommand{Command: "del_keyword", Description: "删除过滤关键词"},
		gotgbot.BotCommand{Command: "replies", Description: "查看自动回复"},
		gotgbot.BotCommand{Command: "add_reply", Description: "添加自动回复"},
		gotgbot.BotCommand{Command: "del_reply", Description: "删除自动回复"},
		gotgbot.BotCommand{Command: "templates", Description: "查看消息模板"},
		gotgbot.BotCommand{Command: "add_template", Description: "添加消息模板"},
		gotgbot.BotCommand{Command: "del_template", Description: "删除消息模板"},
		gotgbot.BotCommand{Command: "post_toggle", Description: "开关定时任务"},
		gotgbot.BotCommand{Command: "post_delete", Description: "删除定时任务"},
		gotgbot.BotCommand{Command: "invites", Description: "查看邀请链接"},
		gotgbot.BotCommand{Command: "invite_create", Description: "创建邀请链接"},
		gotgbot.BotCommand{Command: "invite_delete", Description: "删除邀请链接"},
		gotgbot.BotCommand{Command: "bans", Description: "查看封禁记录"},
		gotgbot.BotCommand{Command: "violations", Description: "查看违规记录"},
		gotgbot.BotCommand{Command: "resolve_violation", Description: "处理违规记录"},
		gotgbot.BotCommand{Command: "ignore_violation", Description: "忽略违规记录"},
	)

	scopes := []struct {
		commands []gotgbot.BotCommand
		scope    gotgbot.BotCommandScope
	}{
		{commands: defaultCommands, scope: gotgbot.BotCommandScopeDefault{}},
		{commands: privateCommands, scope: gotgbot.BotCommandScopeAllPrivateChats{}},
		{commands: groupCommands, scope: gotgbot.BotCommandScopeAllGroupChats{}},
		{commands: adminCommands, scope: gotgbot.BotCommandScopeAllChatAdministrators{}},
	}

	var errs []error
	for _, item := range scopes {
		if _, err := tgBot.SetMyCommandsWithContext(ctx, item.commands, &gotgbot.SetMyCommandsOpts{Scope: item.scope}); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
