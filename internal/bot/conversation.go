package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/redis/go-redis/v9"
)

const (
	selectedChatTTL = 30 * time.Minute
	conversationTTL = 10 * time.Minute
)

type ConversationState struct {
	Type   string         `json:"type"`
	Step   int            `json:"step"`
	ChatID int64          `json:"chat_id"`
	Data   map[string]any `json:"data"`
}

type memoryStateStore struct {
	mu    sync.Mutex
	items map[string]memoryStateItem
}

type memoryStateItem struct {
	value    string
	expireAt time.Time
}

func newMemoryStateStore() *memoryStateStore {
	return &memoryStateStore{items: make(map[string]memoryStateItem)}
}

func selectedChatKey(userID int64) string {
	return fmt.Sprintf("selected_chat:%d", userID)
}

func conversationKey(userID int64) string {
	return fmt.Sprintf("conversation:%d", userID)
}

func (a *App) getStateValue(ctx context.Context, key string) (string, bool) {
	if a.services.Redis != nil {
		value, err := a.services.Redis.Get(ctx, key).Result()
		if err == nil {
			return value, true
		}
		if !errors.Is(err, redis.Nil) {
			return "", false
		}
		return "", false
	}
	if a.state == nil {
		return "", false
	}
	a.state.mu.Lock()
	defer a.state.mu.Unlock()
	item, ok := a.state.items[key]
	if !ok {
		return "", false
	}
	if !item.expireAt.IsZero() && time.Now().After(item.expireAt) {
		delete(a.state.items, key)
		return "", false
	}
	return item.value, true
}

func (a *App) setStateValue(ctx context.Context, key string, value string, ttl time.Duration) error {
	if a.services.Redis != nil {
		return a.services.Redis.Set(ctx, key, value, ttl).Err()
	}
	if a.state == nil {
		return nil
	}
	a.state.mu.Lock()
	defer a.state.mu.Unlock()
	a.state.items[key] = memoryStateItem{value: value, expireAt: time.Now().Add(ttl)}
	return nil
}

func (a *App) clearStateValue(ctx context.Context, key string) {
	if a.services.Redis != nil {
		_ = a.services.Redis.Del(ctx, key).Err()
		return
	}
	if a.state == nil {
		return
	}
	a.state.mu.Lock()
	defer a.state.mu.Unlock()
	delete(a.state.items, key)
}

func (a *App) getSelectedChatID(ctx context.Context, userID int64) (int64, bool) {
	value, ok := a.getStateValue(ctx, selectedChatKey(userID))
	if !ok {
		return 0, false
	}
	chatID, err := strconv.ParseInt(value, 10, 64)
	if err != nil || chatID == 0 {
		return 0, false
	}
	return chatID, true
}

func (a *App) setSelectedChatID(ctx context.Context, userID int64, chatID int64) error {
	return a.setStateValue(ctx, selectedChatKey(userID), strconv.FormatInt(chatID, 10), selectedChatTTL)
}

func (a *App) getConversation(ctx context.Context, userID int64) (*ConversationState, error) {
	value, ok := a.getStateValue(ctx, conversationKey(userID))
	if !ok {
		return nil, nil
	}
	var state ConversationState
	if err := json.Unmarshal([]byte(value), &state); err != nil {
		a.clearConversation(ctx, userID)
		return nil, nil
	}
	if state.Data == nil {
		state.Data = map[string]any{}
	}
	return &state, nil
}

func (a *App) setConversation(ctx context.Context, userID int64, state *ConversationState) error {
	if state == nil {
		a.clearConversation(ctx, userID)
		return nil
	}
	if state.Data == nil {
		state.Data = map[string]any{}
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return a.setStateValue(ctx, conversationKey(userID), string(raw), conversationTTL)
}

func (a *App) clearConversation(ctx context.Context, userID int64) {
	a.clearStateValue(ctx, conversationKey(userID))
}

func (a *App) handleCancel(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	a.clearConversation(scope.Context, scope.Actor.ID)
	return sendText(b, ctx, "已取消当前操作。", nil)
}

func (a *App) handleConversationMessage(b *gotgbot.Bot, ctx *ext.Context) (bool, error) {
	if ctx == nil || ctx.Message == nil {
		return false, nil
	}
	scope := requestScope(ctx)
	if scope.Actor.ID == 0 {
		return false, nil
	}
	state, err := a.getConversation(scope.Context, scope.Actor.ID)
	if err != nil || state == nil {
		return false, err
	}
	if scope.Chat.Type != "private" && state.ChatID != 0 && state.ChatID != scope.Chat.ID {
		return false, nil
	}
	switch state.Type {
	case "create_lottery":
		return true, a.handleLotteryWizardStep(b, ctx, state)
	case "create_post":
		return true, a.handlePostWizardStep(b, ctx, state)
	default:
		a.clearConversation(scope.Context, scope.Actor.ID)
		return true, sendText(b, ctx, "当前操作已失效，请重新开始。", nil)
	}
}
