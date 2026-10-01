package service

import (
	"encoding/json"
	"testing"

	"github.com/dabowin/sola/internal/api"
	"github.com/dabowin/sola/internal/model"
)

func TestNormalizeBroadcastTargetType(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"users", model.BroadcastTargetUsers},
		{"USERS", model.BroadcastTargetUsers},
		{" users ", model.BroadcastTargetUsers},
		{"groups", model.BroadcastTargetGroups},
		{"group", model.BroadcastTargetGroups},
		{"chats", model.BroadcastTargetGroups},
		{"chat", model.BroadcastTargetGroups},
		{"both", model.BroadcastTargetBoth},
		{"all", model.BroadcastTargetBoth},
		{"", ""},
		{"nonsense", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := normalizeBroadcastTargetType(tc.in); got != tc.want {
				t.Errorf("normalizeBroadcastTargetType(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizeChatIDSlice(t *testing.T) {
	if got := normalizeChatIDSlice(nil); len(got) != 0 {
		t.Errorf("nil should normalise to an empty slice, got %v", got)
	}
	if got := normalizeChatIDSlice([]int64{}); got == nil || len(got) != 0 {
		t.Errorf("empty should stay an empty slice, got %v", got)
	}

	got := normalizeChatIDSlice([]int64{-1001, -1002, -1001, 0, -1003})
	want := []int64{-1001, -1002, -1003}
	if len(got) != len(want) {
		t.Fatalf("dedupe/drop-zero failed: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestMarshalAndParseChatIDs(t *testing.T) {
	if got := marshalChatIDs(nil); got != "[]" {
		t.Errorf("marshalChatIDs(nil) = %q, want %q", got, "[]")
	}
	encoded := marshalChatIDs([]int64{-1001, -1002})
	decoded := parseChatIDsJSON(encoded)
	if len(decoded) != 2 || decoded[0] != -1001 || decoded[1] != -1002 {
		t.Errorf("round trip failed: %q -> %v", encoded, decoded)
	}

	for _, raw := range []string{"", "[]", "null", "not json"} {
		if got := parseChatIDsJSON(raw); got != nil {
			t.Errorf("parseChatIDsJSON(%q) = %v, want nil", raw, got)
		}
	}
}

func TestMarshalBroadcastKeyboard(t *testing.T) {
	t.Run("no buttons yields an empty array", func(t *testing.T) {
		got, err := marshalBroadcastKeyboard(nil, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "[]" {
			t.Errorf("got %q, want %q", got, "[]")
		}
	})

	t.Run("structured rows are serialised", func(t *testing.T) {
		rows := []api.InlineKeyboardRow{
			{Buttons: []api.InlineKeyboardButton{
				{Text: "官网", URL: "https://example.com"},
				{Text: "帮助", URL: "https://example.com/help"},
			}},
		}
		got, err := marshalBroadcastKeyboard(rows, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var parsed []struct {
			Buttons []struct {
				Text string `json:"text"`
				URL  string `json:"url"`
			} `json:"buttons"`
		}
		if err := json.Unmarshal([]byte(got), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v (%s)", err, got)
		}
		if len(parsed) != 1 || len(parsed[0].Buttons) != 2 {
			t.Fatalf("unexpected shape: %s", got)
		}
		if parsed[0].Buttons[1].Text != "帮助" {
			t.Errorf("second button lost: %s", got)
		}
	})

	t.Run("raw json string is accepted", func(t *testing.T) {
		raw := `[{"buttons":[{"text":"打开","url":"https://example.com"}]}]`
		got, err := marshalBroadcastKeyboard(nil, raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == "[]" {
			t.Errorf("raw keyboard was dropped: %q", got)
		}
	})

	t.Run("invalid raw json errors", func(t *testing.T) {
		if _, err := marshalBroadcastKeyboard(nil, "{not json}"); err == nil {
			t.Error("expected an error for malformed inline_keyboard_json")
		}
	})

	t.Run("rows with no usable buttons collapse", func(t *testing.T) {
		rows := []api.InlineKeyboardRow{
			{Buttons: []api.InlineKeyboardButton{{Text: "   "}}},
			{Buttons: []api.InlineKeyboardButton{}},
		}
		got, err := marshalBroadcastKeyboard(rows, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "[]" {
			t.Errorf("got %q, want an empty keyboard", got)
		}
	})
}

func TestParseUUIDPtr(t *testing.T) {
	if got := parseUUIDPtr(""); got != nil {
		t.Errorf("empty string should be nil, got %v", got)
	}
	if got := parseUUIDPtr("not-a-uuid"); got != nil {
		t.Errorf("invalid uuid should be nil, got %v", got)
	}
	valid := "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	got := parseUUIDPtr(valid)
	if got == nil || got.String() != valid {
		t.Errorf("parseUUIDPtr(%q) = %v", valid, got)
	}
}

func TestToAPIBroadcastNeverReturnsNilChatIDs(t *testing.T) {
	out := toAPIBroadcast(model.Broadcast{
		ID:          7,
		TargetType:  model.BroadcastTargetGroups,
		ChatIDsJSON: "[]",
	})
	if out.ChatIDs == nil {
		t.Fatal("ChatIDs must serialise as [] rather than null")
	}
	if len(out.ChatIDs) != 0 {
		t.Errorf("expected an empty slice, got %v", out.ChatIDs)
	}

	encoded, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if raw["chat_ids"] == nil {
		t.Errorf("chat_ids encoded as null: %s", encoded)
	}
}

func TestToAPIBroadcastHasMedia(t *testing.T) {
	withURL := toAPIBroadcast(model.Broadcast{MediaURL: "https://example.com/a.jpg", MediaType: "photo"})
	if !withURL.HasMedia {
		t.Error("expected HasMedia=true when a media url is present")
	}
	withInline := toAPIBroadcast(model.Broadcast{MediaData: []byte{1, 2, 3}, MediaType: "photo"})
	if !withInline.HasMedia {
		t.Error("expected HasMedia=true when inline media bytes are present")
	}
	plain := toAPIBroadcast(model.Broadcast{Content: "hello"})
	if plain.HasMedia {
		t.Error("expected HasMedia=false for a text-only broadcast")
	}
}
