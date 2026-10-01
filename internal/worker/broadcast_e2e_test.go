package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"

	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

// fakeTelegram is a stand-in for api.telegram.org. It records every
// sendMessage call and lets a test force specific chats to fail, which is
// exactly what the broadcast fan-out has to cope with in production.
type fakeTelegram struct {
	mu      sync.Mutex
	sends   []int64
	texts   map[int64]string
	methods []string
	// failFor maps a chat id to the Telegram error description to return.
	failFor map[int64]string
}

func newFakeTelegram() *fakeTelegram {
	return &fakeTelegram{
		texts:   map[int64]string{},
		failFor: map[int64]string{},
	}
}

func (f *fakeTelegram) recorded() []int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]int64, len(f.sends))
	copy(out, f.sends)
	return out
}

func (f *fakeTelegram) calledMethods() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.methods))
	copy(out, f.methods)
	return out
}

func (f *fakeTelegram) start(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		// Paths look like /bot<token>/<method>.
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		method := parts[len(parts)-1]

		f.mu.Lock()
		f.methods = append(f.methods, method)
		f.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		if method == "getMe" {
			writeJSON(w, map[string]any{
				"ok": true,
				"result": map[string]any{
					"id": 42, "is_bot": true, "first_name": "SolaTest",
					"username": "solatest_bot", "can_join_groups": true,
				},
			})
			return
		}

		if method != "sendMessage" {
			writeJSON(w, map[string]any{"ok": true, "result": true})
			return
		}

		chatID, text := parseSendMessageBody(r, body)

		f.mu.Lock()
		description, shouldFail := f.failFor[chatID]
		f.mu.Unlock()

		if shouldFail {
			w.WriteHeader(http.StatusForbidden)
			writeJSON(w, map[string]any{
				"ok": false, "error_code": 403, "description": description,
			})
			return
		}

		f.mu.Lock()
		f.sends = append(f.sends, chatID)
		f.texts[chatID] = text
		messageID := int64(len(f.sends))
		f.mu.Unlock()

		writeJSON(w, map[string]any{
			"ok": true,
			"result": map[string]any{
				"message_id": messageID,
				"date":       time.Now().Unix(),
				"chat":       map[string]any{"id": chatID, "type": "supergroup"},
				"text":       text,
			},
		})
	}))
	t.Cleanup(server.Close)
	return server
}

// parseSendMessageBody copes with whichever encoding gotgbot picks
// (JSON, urlencoded, or multipart).
func parseSendMessageBody(r *http.Request, body []byte) (int64, string) {
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "application/json") {
		var payload struct {
			ChatID int64  `json:"chat_id"`
			Text   string `json:"text"`
		}
		if err := json.Unmarshal(body, &payload); err == nil {
			return payload.ChatID, payload.Text
		}
	}

	r.Body = io.NopCloser(bytes.NewReader(body))
	if strings.HasPrefix(contentType, "multipart/form-data") {
		_ = r.ParseMultipartForm(10 << 20)
	} else {
		_ = r.ParseForm()
	}

	var chatID int64
	fmt.Sscanf(r.FormValue("chat_id"), "%d", &chatID)
	return chatID, r.FormValue("text")
}

func writeJSON(w http.ResponseWriter, payload any) {
	encoded, _ := json.Marshal(payload)
	_, _ = w.Write(encoded)
}

// newBroadcastRunner wires a Runner against the fake Telegram API.
func newBroadcastRunner(t *testing.T, apiURL string) (*Runner, *store.Store) {
	t.Helper()

	runner, st := newTestRunner(t)

	bot, err := gotgbot.NewBot("123456:TESTTOKEN", &gotgbot.BotOpts{
		BotClient: &gotgbot.BaseBotClient{
			Client: http.Client{Timeout: 10 * time.Second},
			DefaultRequestOpts: &gotgbot.RequestOpts{
				APIURL: apiURL,
			},
		},
	})
	if err != nil {
		t.Fatalf("build fake bot: %v", err)
	}
	runner.tgBot = bot
	return runner, st
}

// seededChatFilter limits a broadcast to the two chats seedTargets created, so
// the assertions do not depend on whatever else lives in the database.
const seededChatFilter = "[-1009000000101, -1009000000102]"

func insertQueuedBroadcast(t *testing.T, st *store.Store, content string, chatIDs string) uint64 {
	t.Helper()

	record := model.Broadcast{
		Title:              "E2E",
		Content:            content,
		ParseMode:          "HTML",
		InlineKeyboardJSON: "[]",
		TargetType:         model.BroadcastTargetGroups,
		ChatIDsJSON:        chatIDs,
		Status:             model.BroadcastStatusQueued,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	if err := st.DB.Create(&record).Error; err != nil {
		t.Fatalf("insert broadcast: %v", err)
	}
	return record.ID
}

func TestRunBroadcastDeliversToEveryChat(t *testing.T) {
	fake := newFakeTelegram()
	server := fake.start(t)

	runner, st := newBroadcastRunner(t, server.URL)
	cleanupSeed(t, st)
	seedTargets(t, st)
	t.Cleanup(func() { cleanupSeed(t, st) })

	id := insertQueuedBroadcast(t, st, "hello broadcast", seededChatFilter)
	t.Cleanup(func() { st.DB.Exec("DELETE FROM broadcasts WHERE id = ?", id) })

	runner.runBroadcast(context.Background(), id)

	var record model.Broadcast
	if err := st.DB.First(&record, id).Error; err != nil {
		t.Fatalf("reload broadcast: %v", err)
	}

	if record.Status != model.BroadcastStatusCompleted {
		t.Errorf("status = %q, want %q (last_error=%q)", record.Status, model.BroadcastStatusCompleted, record.LastError)
	}
	if record.TotalTargets != 2 {
		t.Errorf("total_targets = %d, want 2", record.TotalTargets)
	}
	if record.SentCount != 2 || record.FailedCount != 0 {
		t.Errorf("sent/failed = %d/%d, want 2/0", record.SentCount, record.FailedCount)
	}
	if record.StartedAt == nil || record.FinishedAt == nil {
		t.Error("started_at and finished_at must both be recorded")
	}

	var deliveries []model.BroadcastDelivery
	if err := st.DB.Where("broadcast_id = ?", id).Order("id asc").Find(&deliveries).Error; err != nil {
		t.Fatalf("load deliveries: %v", err)
	}
	if len(deliveries) != 2 {
		t.Fatalf("expected 2 delivery rows, got %d", len(deliveries))
	}
	for _, d := range deliveries {
		if d.Status != model.BroadcastDeliverySent {
			t.Errorf("delivery to %d = %q (%s), want sent", d.TargetID, d.Status, d.Error)
		}
		if d.MessageID == 0 {
			t.Errorf("delivery to %d has no message id", d.TargetID)
		}
	}

	sent := fake.recorded()
	if len(sent) != 2 {
		t.Fatalf("fake telegram received %d sends, want 2 (%v)", len(sent), sent)
	}
	for _, chatID := range []int64{-1009000000101, -1009000000102} {
		found := false
		for _, got := range sent {
			if got == chatID {
				found = true
			}
		}
		if !found {
			t.Errorf("chat %d never received the broadcast (got %v)", chatID, sent)
		}
	}
}

func TestRunBroadcastNarrowsToRequestedChats(t *testing.T) {
	fake := newFakeTelegram()
	server := fake.start(t)

	runner, st := newBroadcastRunner(t, server.URL)
	cleanupSeed(t, st)
	seedTargets(t, st)
	t.Cleanup(func() { cleanupSeed(t, st) })

	id := insertQueuedBroadcast(t, st, "narrow", "[-1009000000102]")
	t.Cleanup(func() { st.DB.Exec("DELETE FROM broadcasts WHERE id = ?", id) })

	runner.runBroadcast(context.Background(), id)

	sent := fake.recorded()
	if len(sent) != 1 || sent[0] != -1009000000102 {
		t.Fatalf("expected only chat -1009000000102 to be messaged, got %v", sent)
	}
}

func TestRunBroadcastRecordsPerTargetFailures(t *testing.T) {
	fake := newFakeTelegram()
	fake.failFor[-1009000000102] = "Forbidden: bot was blocked by the user"
	server := fake.start(t)

	runner, st := newBroadcastRunner(t, server.URL)
	cleanupSeed(t, st)
	seedTargets(t, st)
	t.Cleanup(func() { cleanupSeed(t, st) })

	id := insertQueuedBroadcast(t, st, "partial", seededChatFilter)
	t.Cleanup(func() { st.DB.Exec("DELETE FROM broadcasts WHERE id = ?", id) })

	runner.runBroadcast(context.Background(), id)

	var record model.Broadcast
	if err := st.DB.First(&record, id).Error; err != nil {
		t.Fatalf("reload broadcast: %v", err)
	}

	if record.SentCount != 1 || record.FailedCount != 1 {
		t.Errorf("sent/failed = %d/%d, want 1/1", record.SentCount, record.FailedCount)
	}
	// One good delivery is enough to keep the job out of the "failed" bucket.
	if record.Status != model.BroadcastStatusCompleted {
		t.Errorf("status = %q, want %q", record.Status, model.BroadcastStatusCompleted)
	}
	if !strings.Contains(record.LastError, "-1009000000102") {
		t.Errorf("last_error should name the failing chat, got %q", record.LastError)
	}

	var failed model.BroadcastDelivery
	if err := st.DB.Where("broadcast_id = ? AND status = ?", id, model.BroadcastDeliveryFailed).
		First(&failed).Error; err != nil {
		t.Fatalf("expected a failed delivery row: %v", err)
	}
	if !strings.Contains(failed.Error, "blocked") {
		t.Errorf("delivery error = %q, want it to mention the Telegram error", failed.Error)
	}
}

func TestRunBroadcastFailsWhenEveryTargetFails(t *testing.T) {
	fake := newFakeTelegram()
	fake.failFor[-1009000000101] = "Forbidden: bot was kicked"
	fake.failFor[-1009000000102] = "Forbidden: bot was kicked"
	server := fake.start(t)

	runner, st := newBroadcastRunner(t, server.URL)
	cleanupSeed(t, st)
	seedTargets(t, st)
	t.Cleanup(func() { cleanupSeed(t, st) })

	id := insertQueuedBroadcast(t, st, "doomed", seededChatFilter)
	t.Cleanup(func() { st.DB.Exec("DELETE FROM broadcasts WHERE id = ?", id) })

	runner.runBroadcast(context.Background(), id)

	var record model.Broadcast
	if err := st.DB.First(&record, id).Error; err != nil {
		t.Fatalf("reload broadcast: %v", err)
	}
	if record.Status != model.BroadcastStatusFailed {
		t.Errorf("status = %q, want %q", record.Status, model.BroadcastStatusFailed)
	}
	if record.SentCount != 0 || record.FailedCount != 2 {
		t.Errorf("sent/failed = %d/%d, want 0/2", record.SentCount, record.FailedCount)
	}
}

func TestRunBroadcastCannotDoubleClaim(t *testing.T) {
	fake := newFakeTelegram()
	server := fake.start(t)

	runner, st := newBroadcastRunner(t, server.URL)
	cleanupSeed(t, st)
	seedTargets(t, st)
	t.Cleanup(func() { cleanupSeed(t, st) })

	id := insertQueuedBroadcast(t, st, "once only", "[-1009000000101]")
	t.Cleanup(func() { st.DB.Exec("DELETE FROM broadcasts WHERE id = ?", id) })

	runner.runBroadcast(context.Background(), id)
	before := len(fake.recorded())

	// A second run must be a no-op because the row is no longer "queued".
	runner.runBroadcast(context.Background(), id)

	if after := len(fake.recorded()); after != before {
		t.Errorf("broadcast was delivered twice: %d -> %d sends", before, after)
	}
}
