package bot

import (
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func New(services Services, options Options) *App {
	if options.DefaultLocale == "" {
		options.DefaultLocale = "zh-CN"
	}

	app := &App{
		services:   services,
		options:    options,
		router:     NewCallbackRouter(),
		state:      newMemoryStateStore(),
		miniAppURL: options.MiniAppURL,
	}
	app.registerCallbackRoutes()
	return app
}

func (a *App) Register(dispatcher *ext.Dispatcher) {
	a.registerCoreHandlers(dispatcher)

	if a.options.Features.Enabled("admin") {
		a.registerAdminHandlers(dispatcher)
	}
	if a.options.Features.Enabled("moderation") {
		a.registerModerationHandlers(dispatcher)
	}
	if a.options.Features.Enabled("verify") {
		a.registerVerifyHandlers(dispatcher)
	}
	if a.options.Features.Enabled("points") {
		a.registerPointsHandlers(dispatcher)
	}
	if a.options.Features.Enabled("lottery") {
		a.registerLotteryHandlers(dispatcher)
	}
	if a.options.Features.Enabled("publish") {
		a.registerPublishHandlers(dispatcher)
	}
	if a.options.Features.Enabled("auto_reply") {
		a.registerAutoReplyHandlers(dispatcher)
	}
	a.registerKeywordHandlers(dispatcher)
	a.registerTemplateHandlers(dispatcher)
	a.registerInviteHandlers(dispatcher)
}
