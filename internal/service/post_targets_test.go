package service

import (
	"strings"
	"testing"
)

func TestNormalizePostInlineKeyboardAcceptsRows(t *testing.T) {
	raw := `[[{"text":"🛒 立即购买","url":"https://shop.example.com/p/1"},{"text":"💬 客服","url":"https://t.me/support"}]]`
	got, err := normalizePostInlineKeyboard(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "🛒 立即购买") || !strings.Contains(got, "shop.example.com/p/1") {
		t.Fatalf("normalized keyboard lost data: %s", got)
	}
}

func TestNormalizePostInlineKeyboardAcceptsWrapper(t *testing.T) {
	raw := `[{"buttons":[{"text":"📦 物流","url":"https://t.me/logistics"}]}]`
	got, err := normalizePostInlineKeyboard(raw)
	if err != nil {
		t.Fatalf("wrapper format should be accepted, got %v", err)
	}
	if !strings.Contains(got, "📦 物流") {
		t.Fatalf("normalized keyboard lost data: %s", got)
	}
}

func TestNormalizePostInlineKeyboardRejectsBadInput(t *testing.T) {
	cases := map[string]string{
		"没有链接":    `[[{"text":"立即购买"}]]`,
		"链接不是网址":  `[[{"text":"立即购买","url":"shop.example.com"}]]`,
		"不是 JSON": `not-json`,
	}
	for name, raw := range cases {
		if _, err := normalizePostInlineKeyboard(raw); err == nil {
			t.Fatalf("%s: expected an error for %s", name, raw)
		}
	}
}

func TestNormalizePostInlineKeyboardEmpty(t *testing.T) {
	for _, raw := range []string{"", "[]", "  "} {
		got, err := normalizePostInlineKeyboard(raw)
		if err != nil {
			t.Fatalf("empty keyboard should be fine, got %v", err)
		}
		if got != "[]" {
			t.Fatalf("got %q, want []", got)
		}
	}
}

func TestPostTargetsEncodeDecode(t *testing.T) {
	encoded := encodePostTargets([]int64{-1001111111111, -1002222222222})
	if encoded != "[-1001111111111,-1002222222222]" {
		t.Fatalf("encoded = %s", encoded)
	}
	decoded := decodePostTargets(encoded)
	if len(decoded) != 2 || decoded[0] != -1001111111111 {
		t.Fatalf("decoded = %#v", decoded)
	}
	if decodePostTargets("") != nil || decodePostTargets("[]") != nil {
		t.Fatalf("empty payload should decode to nil")
	}
	if decodePostTargets("{bad json") != nil {
		t.Fatalf("invalid payload should decode to nil")
	}
}
