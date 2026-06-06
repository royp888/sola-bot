package bot

import (
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/chatmember"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/pollanswer"
)

func New(services Services, options Options) *App {
	if options.DefaultLocale == "" {
		options.DefaultLocale = "zh-CN"
	}

	app := &App{
		services: services,
		options:  options,
		router:   NewCallbackRouter(),
		state:    newMemoryStateStore(),
		miniAppURL: options.MiniAppURL,
	}
	app.registerCallbackRoutes()
	return app
}

func (a *App) Register(dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(handlers.NewCommand("start", a.wrap(a.handleStart, a.RateLimit("cmd:start", 1))))
	dispatcher.AddHandler(handlers.NewCommand("menu", a.wrap(a.handleStart, a.RateLimit("cmd:menu", 1))))
	dispatcher.AddHandler(handlers.NewCommand("settings", a.wrap(a.handleStart, a.RateLimit("cmd:settings", 1))))
	dispatcher.AddHandler(handlers.NewCommand("help", a.wrap(a.handleHelp, a.RateLimit("cmd:help", 1))))
	dispatcher.AddHandler(handlers.NewCommand("info", a.wrap(a.handleInfo, a.RateLimit("cmd:info", 1))))
	dispatcher.AddHandler(handlers.NewCommand("html", a.wrap(a.handleHTMLHelp, a.RateLimit("cmd:html", 1))))
	dispatcher.AddHandler(handlers.NewCommand("bind", a.wrap(a.handleBind, a.RateLimit("cmd:bind", 1))))
	dispatcher.AddHandler(handlers.NewCommand("check_admin", a.wrap(a.handleCheckAdmin, a.RateLimit("cmd:check_admin", 1))))
	dispatcher.AddHandler(handlers.NewCommand("cancel", a.wrap(a.handleCancel, a.RateLimit("cmd:cancel", 1))))

	dispatcher.AddHandler(handlers.NewCommand("rank", a.wrap(a.handlePointsRank, a.RateLimit("cmd:rank", 1))))
	dispatcher.AddHandler(handlers.NewCommand("points_rank", a.wrap(a.handlePointsRank, a.RateLimit("cmd:points_rank", 1))))
	dispatcher.AddHandler(handlers.NewCommand("points", a.wrap(a.handlePoints, a.RateLimit("cmd:points", 1))))
	dispatcher.AddHandler(handlers.NewCommand("sign", a.wrap(a.handleSign, a.RateLimit("cmd:sign", 1))))
	dispatcher.AddHandler(handlers.NewCommand("points_config", a.wrap(a.handlePointsConfig, a.RateLimit("cmd:points_config", 1))))
	dispatcher.AddHandler(handlers.NewCommand("set_points", a.wrap(a.handleSetPoints, a.RateLimit("cmd:set_points", 1))))
	dispatcher.AddHandler(handlers.NewCommand("set_cooldown", a.wrap(a.handleSetCooldown, a.RateLimit("cmd:set_cooldown", 1))))
	dispatcher.AddHandler(handlers.NewCommand("points_toggle", a.wrap(a.handlePointsToggle, a.RateLimit("cmd:points_toggle", 1))))
	dispatcher.AddHandler(handlers.NewCommand("stat", a.wrap(a.handleTodayStats, a.RequirePermission(PermissionStats), a.RateLimit("cmd:stat", 1))))
	dispatcher.AddHandler(handlers.NewCommand("stat_week", a.wrap(a.handleWeekStats, a.RequirePermission(PermissionStats), a.RateLimit("cmd:stat_week", 1))))
	dispatcher.AddHandler(handlers.NewCommand("stats", a.wrap(a.handleCustomStats, a.RequirePermission(PermissionStats), a.RateLimit("cmd:stats", 1))))

	dispatcher.AddHandler(handlers.NewCommand("lottery", a.wrap(a.handleLottery, a.RateLimit("cmd:lottery", 1))))
	dispatcher.AddHandler(handlers.NewCommand("posts", a.wrap(a.handlePosts, a.RequirePermission(PermissionPublish), a.RateLimit("cmd:posts", 1))))
	dispatcher.AddHandler(handlers.NewCommand("publish", a.wrap(a.handlePublish, a.RequirePermission(PermissionPublish), a.RateLimit("cmd:publish", 1))))
	dispatcher.AddHandler(handlers.NewCommand("post_create", a.wrap(a.handlePostCreate, a.RequirePermission(PermissionPublish), a.RateLimit("cmd:post_create", 1))))
	dispatcher.AddHandler(handlers.NewCommand("post_toggle", a.wrap(a.handlePostToggle, a.RequirePermission(PermissionPublish), a.RateLimit("cmd:post_toggle", 1))))
	dispatcher.AddHandler(handlers.NewCommand("post_delete", a.wrap(a.handlePostDelete, a.RequirePermission(PermissionPublish), a.RateLimit("cmd:post_delete", 1))))

	dispatcher.AddHandler(handlers.NewCommand("ban", a.wrap(a.handleBan, a.RateLimit("cmd:ban", 1))))
	dispatcher.AddHandler(handlers.NewCommand("bans", a.wrap(a.handleBans, a.RateLimit("cmd:bans", 1))))
	dispatcher.AddHandler(handlers.NewCommand("violations", a.wrap(a.handleViolations, a.RateLimit("cmd:violations", 1))))
	dispatcher.AddHandler(handlers.NewCommand("resolve_violation", a.wrap(a.handleResolveViolation, a.RateLimit("cmd:resolve_violation", 1))))
	dispatcher.AddHandler(handlers.NewCommand("ignore_violation", a.wrap(a.handleIgnoreViolation, a.RateLimit("cmd:ignore_violation", 1))))
	dispatcher.AddHandler(handlers.NewCommand("manage", a.wrap(a.handleManageMember, a.RateLimit("cmd:manage", 1))))
	dispatcher.AddHandler(handlers.NewCommand("mod", a.wrap(a.handleManageMember, a.RateLimit("cmd:mod", 1))))
	dispatcher.AddHandler(handlers.NewCommand("unban", a.wrap(a.handleUnban, a.RateLimit("cmd:unban", 1))))
	dispatcher.AddHandler(handlers.NewCommand("mute", a.wrap(a.handleMute, a.RateLimit("cmd:mute", 1))))
	dispatcher.AddHandler(handlers.NewCommand("unmute", a.wrap(a.handleUnmute, a.RateLimit("cmd:unmute", 1))))
	dispatcher.AddHandler(handlers.NewCommand("kick", a.wrap(a.handleKick, a.RateLimit("cmd:kick", 1))))
	dispatcher.AddHandler(handlers.NewCommand("warn", a.wrap(a.handleWarn, a.RateLimit("cmd:warn", 1))))
	dispatcher.AddHandler(handlers.NewCommand("unwarn", a.wrap(a.handleUnwarn, a.RateLimit("cmd:unwarn", 1))))
	dispatcher.AddHandler(handlers.NewCommand("warns", a.wrap(a.handleWarns, a.RateLimit("cmd:warns", 1))))
	dispatcher.AddHandler(handlers.NewCommand("adminconfig", a.wrap(a.handleAdminConfig, a.RateLimit("cmd:adminconfig", 1))))
	dispatcher.AddHandler(handlers.NewCommand("set_welcome", a.wrap(a.handleSetWelcome, a.RateLimit("cmd:set_welcome", 1))))
	dispatcher.AddHandler(handlers.NewCommand("set_warn_limit", a.wrap(a.handleSetWarnLimit, a.RateLimit("cmd:set_warn_limit", 1))))
	dispatcher.AddHandler(handlers.NewCommand("set_level", a.wrap(a.handleSetLevel, a.RateLimit("cmd:set_level", 1))))
	dispatcher.AddHandler(handlers.NewCommand("levels", a.wrap(a.handleLevels, a.RateLimit("cmd:levels", 1))))
	dispatcher.AddHandler(handlers.NewCommand("add_level", a.wrap(a.handleAddLevel, a.RateLimit("cmd:add_level", 1))))
	dispatcher.AddHandler(handlers.NewCommand("del_level", a.wrap(a.handleDelLevel, a.RateLimit("cmd:del_level", 1))))
	dispatcher.AddHandler(handlers.NewCommand("verify_toggle", a.wrap(a.handleVerifyToggle, a.RateLimit("cmd:verify_toggle", 1))))
	dispatcher.AddHandler(handlers.NewCommand("set_verify", a.wrap(a.handleSetVerify, a.RateLimit("cmd:set_verify", 1))))
	dispatcher.AddHandler(handlers.NewCommand("verify_stats", a.wrap(a.handleVerifyStats, a.RateLimit("cmd:verify_stats", 1))))
	dispatcher.AddHandler(handlers.NewCommand("allowuser", a.wrap(a.handleAllowUser, a.RateLimit("cmd:allowuser", 1))))
	dispatcher.AddHandler(handlers.NewCommand("delallowuser", a.wrap(a.handleDelAllowUser, a.RateLimit("cmd:delallowuser", 1))))
	dispatcher.AddHandler(handlers.NewCommand("add_keyword", a.wrap(a.handleAddKeyword, a.RateLimit("cmd:add_keyword", 1))))
	dispatcher.AddHandler(handlers.NewCommand("del_keyword", a.wrap(a.handleDelKeyword, a.RateLimit("cmd:del_keyword", 1))))
	dispatcher.AddHandler(handlers.NewCommand("keywords", a.wrap(a.handleKeywords, a.RateLimit("cmd:keywords", 1))))
	dispatcher.AddHandler(handlers.NewCommand("add_reply", a.wrap(a.handleAddReply, a.RateLimit("cmd:add_reply", 1))))
	dispatcher.AddHandler(handlers.NewCommand("del_reply", a.wrap(a.handleDelReply, a.RateLimit("cmd:del_reply", 1))))
	dispatcher.AddHandler(handlers.NewCommand("replies", a.wrap(a.handleListReplies, a.RateLimit("cmd:replies", 1))))
	dispatcher.AddHandler(handlers.NewCommand("templates", a.wrap(a.handleTemplates, a.RateLimit("cmd:templates", 1))))
	dispatcher.AddHandler(handlers.NewCommand("add_template", a.wrap(a.handleAddTemplate, a.RateLimit("cmd:add_template", 1))))
	dispatcher.AddHandler(handlers.NewCommand("del_template", a.wrap(a.handleDelTemplate, a.RateLimit("cmd:del_template", 1))))
	dispatcher.AddHandler(handlers.NewCommand("invites", a.wrap(a.handleInvites, a.RateLimit("cmd:invites", 1))))
	dispatcher.AddHandler(handlers.NewCommand("invite_create", a.wrap(a.handleInviteCreate, a.RateLimit("cmd:invite_create", 1))))
	dispatcher.AddHandler(handlers.NewCommand("invite_delete", a.wrap(a.handleInviteDelete, a.RateLimit("cmd:invite_delete", 1))))

	dispatcher.AddHandler(handlers.NewChatMember(chatmember.InviteLink, a.handleChatMemberInviteLink))
	dispatcher.AddHandler(handlers.NewMessage(message.NewChatMembers, a.handleNewChatMembers))
	dispatcher.AddHandler(handlers.NewMessage(message.All, a.handleChineseCommand))
	dispatcher.AddHandler(handlers.NewMessage(message.All, a.handleMessageModeration))
	dispatcher.AddHandler(handlers.NewMessage(message.All, a.handleMessagePoints))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix(CallbackPrefix+":"), a.wrap(a.router.Handle, a.RateLimit("callback", 1))))
	dispatcher.AddHandler(handlers.NewPollAnswer(pollanswer.All, a.handlePollAnswer))
}
