package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

const adminCacheTTL = time.Hour

type cachedAdmin struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	Status    string `json:"status"` // "creator" or "administrator"
	IsBot     bool   `json:"is_bot"`
}

func adminCacheKey(chatID int64) string {
	return fmt.Sprintf("admin:cache:%d", chatID)
}

func (a *App) getCachedAdmins(ctx context.Context, b *gotgbot.Bot, chatID int64) ([]cachedAdmin, error) {
	if a.services.Redis != nil {
		if raw, err := a.services.Redis.Get(ctx, adminCacheKey(chatID)).Result(); err == nil {
			var cached []cachedAdmin
			if json.Unmarshal([]byte(raw), &cached) == nil {
				return cached, nil
			}
		}
	}

	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	members, err := b.GetChatAdministratorsWithContext(reqCtx, chatID, nil)
	if err != nil {
		return nil, err
	}

	out := make([]cachedAdmin, 0, len(members))
	for _, m := range members {
		user := m.GetUser()
		out = append(out, cachedAdmin{
			UserID:    user.Id,
			Username:  user.Username,
			FirstName: user.FirstName,
			Status:    m.GetStatus(),
			IsBot:     user.IsBot,
		})
	}

	if a.services.Redis != nil {
		if payload, err := json.Marshal(out); err == nil {
			_ = a.services.Redis.Set(ctx, adminCacheKey(chatID), string(payload), adminCacheTTL)
		}
	}
	return out, nil
}

func (a *App) invalidateAdminCache(chatID int64) {
	if a.services.Redis == nil {
		return
	}
	_ = a.services.Redis.Del(context.Background(), adminCacheKey(chatID))
}

func (a *App) isAdminInCache(admins []cachedAdmin, userID int64) bool {
	for _, admin := range admins {
		if admin.UserID == userID {
			return true
		}
	}
	return false
}
