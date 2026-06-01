package bot

import (
	"fmt"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

type CallbackPayload struct {
	Raw       string
	Domain    string
	Action    string
	Resource  string
	Arguments []string
}

type CallbackRoute func(b *gotgbot.Bot, ctx *ext.Context, payload CallbackPayload) error

type CallbackRouter struct {
	routes map[string]CallbackRoute
}

func NewCallbackRouter() *CallbackRouter {
	return &CallbackRouter{routes: make(map[string]CallbackRoute)}
}

func (r *CallbackRouter) HandleDomain(domain string, handler CallbackRoute) {
	r.routes[domain] = handler
}

func (r *CallbackRouter) Handle(b *gotgbot.Bot, ctx *ext.Context) error {
	if ctx == nil || ctx.CallbackQuery == nil {
		return nil
	}
	payload, ok := ParseCallbackData(ctx.CallbackQuery.Data)
	if !ok {
		return answerCallback(b, ctx, "未知操作")
	}
	handler, ok := r.routes[payload.Domain]
	if !ok {
		if err := answerCallback(b, ctx, "当前入口还在完善，先回主菜单"); err != nil {
			return err
		}
		return respondText(b, ctx, fmt.Sprintf("入口 %q 还没完成接入，先回到主菜单继续操作。", payload.Domain), backHomeMarkup())
	}
	if shouldPreAnswerCallback(payload) {
		if err := answerCallback(b, ctx, "处理中"); err != nil {
			return err
		}
	}
	return handler(b, ctx, payload)
}

func CallbackData(parts ...string) string {
	all := append([]string{CallbackPrefix}, parts...)
	return strings.Join(all, ":")
}

func ParseCallbackData(data string) (CallbackPayload, bool) {
	parts := strings.Split(data, ":")
	if len(parts) < 3 || parts[0] != CallbackPrefix {
		return CallbackPayload{}, false
	}
	payload := CallbackPayload{
		Raw:    data,
		Domain: parts[1],
		Action: parts[2],
	}
	if len(parts) > 3 {
		payload.Resource = parts[3]
	}
	if len(parts) > 4 {
		payload.Arguments = parts[4:]
	}
	return payload, true
}

func shouldPreAnswerCallback(payload CallbackPayload) bool {
	if payload.Domain == "admin" && payload.Action == "verify" {
		return false
	}
	if payload.Domain == "lottery" && payload.Action == "join" {
		return false
	}
	if payload.Domain == "wizard" {
		return false
	}
	return true
}

func (a *App) registerCallbackRoutes() {
	a.router.HandleDomain("menu", a.routeMenuCallback)
	a.router.HandleDomain("points", a.routePointsCallback)
	a.router.HandleDomain("lottery", a.routeLotteryCallback)
	a.router.HandleDomain("publish", a.routePublishCallback)
	a.router.HandleDomain("admin", a.routeAdminCallback)
	a.router.HandleDomain("private", a.routePrivateCallback)
	a.router.HandleDomain("wizard", a.routeWizardCallback)
}
