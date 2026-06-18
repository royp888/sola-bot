package bot

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
)

func (a *App) registerSedHandlers(d *ext.Dispatcher) {
	d.AddHandler(handlers.NewMessage(message.Text, a.handleSed))
}

func isSedSeparator(c byte) bool {
	return c == '/' || c == '|' || c == '_' || c == ':'
}

// parseSedCommand parses text of the form s/pattern/replacement/[flags].
// Returns ok=false when text doesn't match the sed format.
func parseSedCommand(text string) (pattern, replacement, flags string, ok bool) {
	if len(text) < 3 || text[0] != 's' {
		return "", "", "", false
	}
	sep := text[1]
	if !isSedSeparator(sep) {
		return "", "", "", false
	}
	parts := strings.Split(text[2:], string(sep))
	if len(parts) < 2 {
		return "", "", "", false
	}
	pattern = parts[0]
	if pattern == "" {
		return "", "", "", false
	}
	replacement = parts[1]
	if len(parts) >= 3 {
		flags = parts[2]
	}
	return pattern, replacement, flags, true
}

func (a *App) handleSed(b *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.Message == nil || ctx.Message.ReplyToMessage == nil {
		return ext.ContinueGroups
	}
	original := strings.TrimSpace(ctx.Message.ReplyToMessage.Text)
	if original == "" {
		return ext.ContinueGroups
	}
	text := strings.TrimSpace(ctx.Message.Text)
	pattern, replacement, flags, ok := parseSedCommand(text)
	if !ok {
		return ext.ContinueGroups
	}

	globalReplace := strings.Contains(flags, "g")
	regexPrefix := ""
	if strings.Contains(flags, "i") {
		regexPrefix = "(?i)"
	}
	fullPattern := regexPrefix + regexp.QuoteMeta(pattern)

	type result struct {
		out string
		err error
	}
	ch := make(chan result, 1)
	go func() {
		re, err := regexp.Compile(fullPattern)
		if err != nil {
			ch <- result{err: err}
			return
		}
		var out string
		if globalReplace {
			out = re.ReplaceAllLiteralString(original, replacement)
		} else {
			count := 0
			out = re.ReplaceAllStringFunc(original, func(m string) string {
				if count == 0 {
					count++
					return replacement
				}
				return m
			})
		}
		ch <- result{out: out}
	}()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-timeoutCtx.Done():
		return ext.ContinueGroups
	case res := <-ch:
		if res.err != nil || res.out == original {
			return ext.ContinueGroups
		}
		scope := requestScope(ctx)
		out := res.out
		if len([]rune(out)) > 4000 {
			out = string([]rune(out)[:4000]) + "…"
		}
		fromName := ""
		if ctx.Message.ReplyToMessage.From != nil {
			fromName = strings.TrimSpace(ctx.Message.ReplyToMessage.From.FirstName)
		}
		msg := fmt.Sprintf("修正：%s", out)
		if fromName != "" {
			msg = fmt.Sprintf("%s 说的应该是：\n%s", fromName, out)
		}
		_, _ = b.DeleteMessageWithContext(scope.Context, scope.Chat.ID, ctx.Message.MessageId, nil)
		return sendText(b, ctx, msg, nil)
	}
}
