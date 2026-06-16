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
		MiniAppURL:    cfg.Bot.MiniAppURL,
		Features:      botapp.NewFeatures(cfg.Bot.DisabledFeatures),
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
				"poll_answer",
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
	groupMemberCommands := []gotgbot.BotCommand{
		{Command: "start", Description: "打开群组控制面板"},
		{Command: "help", Description: "查看命令说明"},
		{Command: "points", Description: "查看我的积分"},
		{Command: "rank", Description: "查看积分榜单"},
		{Command: "sign", Description: "每日签到"},
		{Command: "lottery", Description: "查看进行中的抽奖"},
		{Command: "info", Description: "查看当前会话信息"},
	}
	groupAdminCommands := append(append([]gotgbot.BotCommand{}, groupMemberCommands...), []gotgbot.BotCommand{
		{Command: "ban", Description: "封禁成员（回复消息或接用户ID）"},
		{Command: "unban", Description: "解封 /unban 用户ID"},
		{Command: "mute", Description: "禁言 /mute 30m"},
		{Command: "unmute", Description: "解除禁言"},
		{Command: "kick", Description: "踢出成员"},
		{Command: "warn", Description: "警告成员"},
		{Command: "manage", Description: "打开成员管理面板"},
		{Command: "purge", Description: "批量删消息 /purge 或回复+/purge"},
		{Command: "del", Description: "删除回复的消息"},
		{Command: "promote", Description: "提升为管理员"},
		{Command: "demote", Description: "撤销管理员权限"},
		{Command: "set_title", Description: "设置管理员头衔"},
		{Command: "report", Description: "举报消息通知管理员"},
		{Command: "ban_ghosts", Description: "清理注销账号"},
		{Command: "setrules", Description: "设置群规"},
		{Command: "clearrules", Description: "清除群规"},
		{Command: "rules", Description: "查看群规"},
		{Command: "publish", Description: "立即发布内容"},
		{Command: "posts", Description: "查看定时任务"},
		{Command: "adminconfig", Description: "查看群组配置"},
		{Command: "verify_toggle", Description: "开关入群验证"},
		{Command: "keywords", Description: "查看关键词规则"},
		{Command: "invites", Description: "管理邀请链接"},
		{Command: "bans", Description: "查看封禁记录"},
	}...)
	privateCommands := []gotgbot.BotCommand{
		{Command: "start", Description: "打开运营工作台"},
		{Command: "help", Description: "查看私聊功能说明"},
		{Command: "bind", Description: "绑定群组或频道"},
		{Command: "publish", Description: "快捷发布"},
		{Command: "posts", Description: "查看定时任务"},
		{Command: "info", Description: "查看会话信息"},
		{Command: "html", Description: "HTML 格式参考"},
		{Command: "cancel", Description: "取消当前操作"},
	}

	var errs []error
	if _, err := tgBot.SetMyCommandsWithContext(ctx, groupMemberCommands, &gotgbot.SetMyCommandsOpts{Scope: gotgbot.BotCommandScopeAllGroupChats{}}); err != nil {
		errs = append(errs, err)
	}
	if _, err := tgBot.SetMyCommandsWithContext(ctx, privateCommands, &gotgbot.SetMyCommandsOpts{Scope: gotgbot.BotCommandScopeAllPrivateChats{}}); err != nil {
		errs = append(errs, err)
	}
	if _, err := tgBot.SetMyCommandsWithContext(ctx, groupAdminCommands, &gotgbot.SetMyCommandsOpts{Scope: gotgbot.BotCommandScopeAllChatAdministrators{}}); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
