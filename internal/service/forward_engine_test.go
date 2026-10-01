package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/dabowin/sola/internal/api"
	"github.com/dabowin/sola/internal/model"
)

func baseForwardRule() model.AutoForward {
	return model.AutoForward{
		ID:               1,
		Name:             "A频道 -> 带货频道",
		SourceChatID:     -1001111111111,
		SourceTitle:      "A 频道",
		TargetChatID:     -1002222222222,
		Mode:             model.ForwardModeCopy,
		SkipDuplicates:   true,
		KeepOriginalText: true,
		BuyText:          "🛒 立即购买",
		BuyButtonOnMedia: true,
		BuyButtonOnText:  true,
	}
}

func TestRenderForwardKeepsOriginalByDefault(t *testing.T) {
	rule := baseForwardRule()
	got := renderForward(rule, "  原价 199  ", 10, rule.SourceChatID)
	if got.caption != "原价 199" {
		t.Fatalf("caption = %q, want trimmed original", got.caption)
	}
	if got.changed {
		t.Fatalf("changed should be false when no rewrite option is set")
	}
	if got.buyURL != "" {
		t.Fatalf("buyURL should be empty when buy button is disabled, got %q", got.buyURL)
	}
}

func TestRenderForwardTemplateAndBuyButton(t *testing.T) {
	rule := baseForwardRule()
	rule.BuyEnabled = true
	rule.BuyURLTemplate = "https://shop.example.com/product/{product_id}?ch={chat_id}&m={msg_id}"
	rule.CaptionTemplate = "🔥 {title}\n{text}\n\n{link}"
	rule.RemoveLinks = true
	rule.RemoveMentions = true
	rule.DropSourceLine = true

	source := "春夏新款连衣裙 #123\n来源：@some_channel https://t.me/some_channel\n联系 @seller 下单"
	got := renderForward(rule, source, 88, rule.SourceChatID)

	if got.buyURL != "https://shop.example.com/product/123?ch=-1001111111111&m=88" {
		t.Fatalf("buyURL = %q", got.buyURL)
	}
	if !strings.Contains(got.caption, "🔥 春夏新款连衣裙 #123") {
		t.Fatalf("caption should start with rewritten title, got %q", got.caption)
	}
	if strings.Contains(got.caption, "@seller") || strings.Contains(got.caption, "t.me/some_channel") {
		t.Fatalf("links/mentions/source line should be stripped, got %q", got.caption)
	}
	if !strings.Contains(got.caption, "https://t.me/c/1111111111/88") {
		t.Fatalf("caption should contain the source message link, got %q", got.caption)
	}

	keyboard, err := buildForwardKeyboard(rule, got.buyURL, false)
	if err != nil {
		t.Fatalf("buildForwardKeyboard: %v", err)
	}
	if keyboard == nil || len(keyboard.InlineKeyboard) != 1 {
		t.Fatalf("expected exactly one buy button row, got %#v", keyboard)
	}
	button := keyboard.InlineKeyboard[0][0]
	if button.Text != "🛒 立即购买" || button.Url != got.buyURL {
		t.Fatalf("unexpected buy button %#v", button)
	}
}

func TestRenderForwardSkipsBuyButtonWhenNoProductID(t *testing.T) {
	rule := baseForwardRule()
	rule.BuyEnabled = true
	rule.BuyURLTemplate = "https://shop.example.com/product/{product_id}"

	got := renderForward(rule, "没有任何编号的文案", 5, rule.SourceChatID)
	if got.buyURL != "" {
		t.Fatalf("buyURL should be empty without a product id, got %q", got.buyURL)
	}
	if len(got.warnings) == 0 {
		t.Fatalf("expected a warning explaining the missing product id")
	}
	keyboard, err := buildForwardKeyboard(rule, got.buyURL, false)
	if err != nil {
		t.Fatalf("buildForwardKeyboard: %v", err)
	}
	if keyboard != nil {
		t.Fatalf("no keyboard should be built, got %#v", keyboard)
	}
}

func TestRenderForwardExtraButtonsAndBuyRow(t *testing.T) {
	rule := baseForwardRule()
	rule.BuyEnabled = true
	rule.BuyURLTemplate = "https://shop.example.com/p/{product_id}"
	rule.ExtraButtonsJSON = `[{"buttons":[{"text":"📦 物流","url":"https://t.me/logistics"},{"text":"💬 客服","url":"https://t.me/support"}]}]`

	got := renderForward(rule, "商品 #777", 9, rule.SourceChatID)
	keyboard, err := buildForwardKeyboard(rule, got.buyURL, true)
	if err != nil {
		t.Fatalf("buildForwardKeyboard: %v", err)
	}
	if keyboard == nil || len(keyboard.InlineKeyboard) != 2 {
		t.Fatalf("expected extra row + buy row, got %#v", keyboard)
	}
	if len(keyboard.InlineKeyboard[0]) != 2 || keyboard.InlineKeyboard[0][0].Text != "📦 物流" {
		t.Fatalf("extra buttons row wrong: %#v", keyboard.InlineKeyboard[0])
	}
	if keyboard.InlineKeyboard[1][0].Text != "🛒 立即购买" {
		t.Fatalf("buy button should be the last row, got %#v", keyboard.InlineKeyboard[1])
	}
}

func TestRenderForwardEscapesHTMLText(t *testing.T) {
	rule := baseForwardRule()
	rule.ParseMode = "HTML"
	rule.CaptionTemplate = "<b>{title}</b>\n{text}"

	got := renderForward(rule, "5 < 10 & 好东西", 1, rule.SourceChatID)
	if !strings.Contains(got.caption, "5 &lt; 10 &amp; 好东西") {
		t.Fatalf("original text should be HTML escaped, got %q", got.caption)
	}
	if !strings.Contains(got.caption, "<b>5 &lt; 10 &amp; 好东西</b>") {
		t.Fatalf("template tags must survive escaping, got %q", got.caption)
	}
	if strings.Contains(got.caption, "<b>5 < 10") {
		t.Fatalf("title placeholder must be escaped too, got %q", got.caption)
	}
}

func TestRenderForwardReplacementsAndPrefixSuffix(t *testing.T) {
	rule := baseForwardRule()
	rule.CaptionPrefix = "🔥 今日特价"
	rule.CaptionSuffix = "👉 点下方按钮下单"
	rule.ReplacementsJSON = `[{"from":"某品牌","to":"我们家"}]`

	got := renderForward(rule, "某品牌新款", 1, rule.SourceChatID)
	want := "🔥 今日特价\n\n我们家新款\n\n👉 点下方按钮下单"
	if got.caption != want {
		t.Fatalf("caption = %q, want %q", got.caption, want)
	}
}

func TestShouldSkipForward(t *testing.T) {
	textMsg := func(text string) *gotgbot.Message {
		return &gotgbot.Message{Text: text, Chat: gotgbot.Chat{Id: -1001111111111, Type: "channel"}}
	}
	photoMsg := func(text string) *gotgbot.Message {
		return &gotgbot.Message{
			Caption: text,
			Chat:    gotgbot.Chat{Id: -1001111111111, Type: "channel"},
			Photo:   []gotgbot.PhotoSize{{FileSize: 2_000_000}},
		}
	}

	t.Run("keyword", func(t *testing.T) {
		rule := baseForwardRule()
		rule.SkipKeywords = "广告, 加微信"
		if reason := shouldSkipForward(rule, nil, textMsg("正品代购 加微信 询价")); reason == "" {
			t.Fatalf("expected keyword skip")
		}
		if reason := shouldSkipForward(rule, nil, textMsg("正常商品文案")); reason != "" {
			t.Fatalf("unexpected skip reason %q", reason)
		}
	})

	t.Run("duplicate filters", func(t *testing.T) {
		rule := baseForwardRule()
		rule.SkipNoMedia = true
		if reason := shouldSkipForward(rule, nil, textMsg("纯文字消息")); reason != "no_media" {
			t.Fatalf("reason = %q, want no_media", reason)
		}
		rule = baseForwardRule()
		rule.SkipNoText = true
		if reason := shouldSkipForward(rule, nil, textMsg("   ")); reason != "no_text" {
			t.Fatalf("reason = %q, want no_text", reason)
		}
		rule = baseForwardRule()
		rule.MinTextLength = 10
		if reason := shouldSkipForward(rule, nil, textMsg("太短")); reason != "text_too_short" {
			t.Fatalf("reason = %q, want text_too_short", reason)
		}
		rule = baseForwardRule()
		rule.MaxFileSizeMB = 1
		if reason := shouldSkipForward(rule, nil, photoMsg("大图")); reason != "file_too_large" {
			t.Fatalf("reason = %q, want file_too_large", reason)
		}
		if mediaType := forwardMediaType(photoMsg("图")); mediaType != "photo" {
			t.Fatalf("media type = %q, want photo", mediaType)
		}
	})
}

func TestRenderForwardPreviewPayload(t *testing.T) {
	rule := baseForwardRule()
	rule.BuyEnabled = true
	rule.BuyURLTemplate = "https://shop.example.com/p/{product_id}"

	resp := RenderForwardPreview(rule, api.AutoForwardPreviewRequest{
		Text:      "羽绒服 #55",
		MediaType: "photo",
		MessageID: 12,
	})
	if resp.Skipped {
		t.Fatalf("preview should not be skipped: %s", resp.SkipReason)
	}
	if len(resp.Buttons) != 1 || resp.Buttons[0].Buttons[0].URL != "https://shop.example.com/p/55" {
		t.Fatalf("unexpected preview buttons %#v", resp.Buttons)
	}
	if !strings.Contains(resp.Caption, "羽绒服 #55") {
		t.Fatalf("unexpected preview caption %q", resp.Caption)
	}
}

func TestIsMessageNotFound(t *testing.T) {
	if !isMessageNotFound(errors.New("Bad Request: message to copy not found")) {
		t.Fatalf("expected not-found detection")
	}
	if !isMessageNotFound(errors.New("Bad Request: MESSAGE_ID_INVALID")) {
		t.Fatalf("expected MessageIdInvalid detection")
	}
	if isMessageNotFound(nil) {
		t.Fatalf("nil error must not be not-found")
	}
}
