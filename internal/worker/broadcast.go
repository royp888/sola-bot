package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"go.uber.org/zap"

	"github.com/dabowin/sola/internal/model"
)

// broadcastSendDelay throttles the fan-out so we stay well below Telegram's
// per-chat flood limits when a broadcast has thousands of targets.
const broadcastSendDelay = 50 * time.Millisecond

// broadcastBatchLimit caps how many queued broadcasts we pick up per tick.
const broadcastBatchLimit = 5

type broadcastTarget struct {
	Type string // model.BroadcastTargetUsers | model.BroadcastTargetGroups
	ID   int64
	Name string
}

// runDueBroadcasts is registered as its own scheduler job (rather than being
// folded into runDueJobs) because a broadcast can take minutes and must not
// hold the lock that the other jobs need.
func (r *Runner) runDueBroadcasts() {
	if r.store == nil || r.store.DB == nil || r.tgBot == nil {
		return
	}
	r.muBroadcast.Lock()
	defer r.muBroadcast.Unlock()

	ctx := context.Background()
	now := time.Now()

	var broadcasts []model.Broadcast
	if err := r.store.DB.WithContext(ctx).
		Where("status = ?", model.BroadcastStatusQueued).
		Where("scheduled_at IS NULL OR scheduled_at <= ?", now).
		Order("scheduled_at asc NULLS FIRST, id asc").
		Limit(broadcastBatchLimit).
		Find(&broadcasts).Error; err != nil {
		r.log.Error("load queued broadcasts", zap.Error(err))
		return
	}

	for _, b := range broadcasts {
		r.runBroadcast(ctx, b.ID)
	}
}

func (r *Runner) runBroadcast(ctx context.Context, broadcastID uint64) {
	// Claim the job atomically so two workers can never double-send.
	res := r.store.DB.WithContext(ctx).
		Model(&model.Broadcast{}).
		Where("id = ? AND status = ?", broadcastID, model.BroadcastStatusQueued).
		Updates(map[string]any{
			"status":     model.BroadcastStatusSending,
			"started_at": time.Now(),
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		r.log.Error("claim broadcast", zap.Uint64("broadcast_id", broadcastID), zap.Error(res.Error))
		return
	}
	if res.RowsAffected == 0 {
		return // someone else got it
	}

	var b model.Broadcast
	if err := r.store.DB.WithContext(ctx).First(&b, broadcastID).Error; err != nil {
		r.log.Error("load broadcast", zap.Uint64("broadcast_id", broadcastID), zap.Error(err))
		return
	}

	targets, err := r.resolveBroadcastTargets(ctx, b)
	if err != nil {
		r.finishBroadcast(ctx, b.ID, model.BroadcastStatusFailed, 0, 0, err.Error())
		return
	}
	if len(targets) == 0 {
		r.finishBroadcast(ctx, b.ID, model.BroadcastStatusFailed, 0, 0, "no targets matched")
		return
	}
	if err := r.store.DB.WithContext(ctx).Model(&model.Broadcast{}).
		Where("id = ?", b.ID).Update("total_targets", len(targets)).Error; err != nil {
		r.log.Warn("set broadcast total", zap.Uint64("broadcast_id", b.ID), zap.Error(err))
	}

	// Pre-create delivery rows so the UI can show progress live.
	deliveries := make([]model.BroadcastDelivery, 0, len(targets))
	for _, t := range targets {
		deliveries = append(deliveries, model.BroadcastDelivery{
			BroadcastID: b.ID,
			TargetType:  t.Type,
			TargetID:    t.ID,
			TargetName:  truncate(t.Name, 250),
			Status:      model.BroadcastDeliveryPending,
			CreatedAt:   time.Now(),
		})
	}
	if err := r.store.DB.WithContext(ctx).CreateInBatches(&deliveries, 500).Error; err != nil {
		r.log.Error("create broadcast deliveries", zap.Uint64("broadcast_id", b.ID), zap.Error(err))
		r.finishBroadcast(ctx, b.ID, model.BroadcastStatusFailed, 0, 0, err.Error())
		return
	}

	var sent, failed int
	var firstErr string

	for i := range deliveries {
		d := &deliveries[i]
		msg, err := r.sendBroadcastContent(ctx, b, d.TargetID)
		now := time.Now()
		d.SentAt = &now

		if err != nil {
			d.Status = model.BroadcastDeliveryFailed
			d.Error = truncate(err.Error(), 900)
			failed++
			if firstErr == "" {
				firstErr = fmt.Sprintf("chat %d: %s", d.TargetID, err.Error())
			}
		} else {
			d.Status = model.BroadcastDeliverySent
			if msg != nil {
				d.MessageID = msg.MessageId
			}
			sent++
		}

		if err := r.store.DB.WithContext(ctx).Save(d).Error; err != nil {
			r.log.Warn("save broadcast delivery", zap.Uint64("delivery_id", d.ID), zap.Error(err))
		}
		// Persist running counters so the web UI shows live progress.
		if (sent+failed)%25 == 0 {
			r.store.DB.WithContext(ctx).Model(&model.Broadcast{}).Where("id = ?", b.ID).
				Updates(map[string]any{"sent_count": sent, "failed_count": failed, "updated_at": time.Now()})
		}

		select {
		case <-ctx.Done():
			r.finishBroadcast(ctx, b.ID, model.BroadcastStatusFailed, sent, failed, "canceled")
			return
		default:
		}
		time.Sleep(broadcastSendDelay)
	}

	status := model.BroadcastStatusCompleted
	if sent == 0 {
		status = model.BroadcastStatusFailed
	}
	r.finishBroadcast(ctx, b.ID, status, sent, failed, firstErr)
	r.log.Info("broadcast finished",
		zap.Uint64("broadcast_id", b.ID),
		zap.Int("sent", sent),
		zap.Int("failed", failed),
	)
}

func (r *Runner) finishBroadcast(ctx context.Context, id uint64, status string, sent, failed int, lastErr string) {
	if err := r.store.DB.WithContext(ctx).Model(&model.Broadcast{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       status,
			"sent_count":   sent,
			"failed_count": failed,
			"last_error":   truncate(lastErr, 900),
			"finished_at":  time.Now(),
			"updated_at":   time.Now(),
		}).Error; err != nil {
		r.log.Error("finish broadcast", zap.Uint64("broadcast_id", id), zap.Error(err))
	}
}

// resolveBroadcastTargets turns a broadcast definition into a concrete list of
// users and/or chats. When ChatIDsJSON is empty every active chat is used.
func (r *Runner) resolveBroadcastTargets(ctx context.Context, b model.Broadcast) ([]broadcastTarget, error) {
	targets := make([]broadcastTarget, 0, 64)

	if b.TargetType == model.BroadcastTargetUsers || b.TargetType == model.BroadcastTargetBoth {
		var users []model.User
		query := r.store.DB.WithContext(ctx).
			Where("telegram_user_id IS NOT NULL").
			Where("is_active = ?", true).
			Where("status = ?", "active")
		if err := query.Find(&users).Error; err != nil {
			return nil, fmt.Errorf("load users: %w", err)
		}
		for _, u := range users {
			if u.TelegramUserID == nil || *u.TelegramUserID == 0 {
				continue
			}
			name := u.DisplayName
			if name == "" && u.Username != nil {
				name = "@" + *u.Username
			}
			targets = append(targets, broadcastTarget{
				Type: model.BroadcastTargetUsers,
				ID:   *u.TelegramUserID,
				Name: name,
			})
		}
	}

	if b.TargetType == model.BroadcastTargetGroups || b.TargetType == model.BroadcastTargetBoth {
		restrict := parseChatIDList(b.ChatIDsJSON)

		var chats []model.TelegramChat
		query := r.store.DB.WithContext(ctx).Where("status = ?", "active")
		if len(restrict) > 0 {
			query = query.Where("telegram_chat_id IN ?", restrict)
		}
		if err := query.Find(&chats).Error; err != nil {
			return nil, fmt.Errorf("load chats: %w", err)
		}
		for _, c := range chats {
			name := ""
			if c.Title != nil {
				name = *c.Title
			}
			if name == "" && c.Username != nil {
				name = "@" + *c.Username
			}
			targets = append(targets, broadcastTarget{
				Type: model.BroadcastTargetGroups,
				ID:   c.TelegramChatID,
				Name: name,
			})
		}
	}

	return targets, nil
}

func parseChatIDList(raw string) []int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "null" {
		return nil
	}
	var ids []int64
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil
	}
	return ids
}

// sendBroadcastContent delivers one broadcast payload to a single chat id.
// It mirrors the media handling used by scheduled posts so text, photo, video
// and document broadcasts (with inline buttons) all behave the same way.
func (r *Runner) sendBroadcastContent(ctx context.Context, b model.Broadcast, chatID int64) (*gotgbot.Message, error) {
	if r.tgBot == nil {
		return nil, fmt.Errorf("telegram bot is not configured")
	}

	content := strings.TrimSpace(b.Content)
	mediaURL := strings.TrimSpace(b.MediaURL)
	mediaName := strings.TrimSpace(b.MediaName)
	if mediaName == "" {
		mediaName = "broadcast-upload"
	}
	hasInlineMedia := len(b.MediaData) > 0

	keyboard, err := parseInlineKeyboard(b.InlineKeyboardJSON)
	if err != nil {
		r.log.Warn("broadcast has invalid inline keyboard", zap.Uint64("broadcast_id", b.ID), zap.Error(err))
	}

	parseMode := strings.TrimSpace(b.ParseMode)
	if parseMode == "" {
		parseMode = "HTML"
	}

	var sent *gotgbot.Message
	var sendErr error

	switch strings.ToLower(strings.TrimSpace(b.MediaType)) {
	case "photo":
		if !hasInlineMedia && mediaURL == "" {
			return nil, fmt.Errorf("photo broadcast requires media")
		}
		file := gotgbot.InputFileByURL(mediaURL)
		if hasInlineMedia {
			file = gotgbot.InputFileByReader(mediaName, bytes.NewReader(b.MediaData))
		}
		opts := &gotgbot.SendPhotoOpts{Caption: content, ParseMode: parseMode}
		if keyboard != nil {
			opts.ReplyMarkup = keyboard
		}
		sent, sendErr = r.sendWithRetry(ctx, chatID, func() (*gotgbot.Message, error) {
			return r.tgBot.SendPhotoWithContext(ctx, chatID, file, opts)
		})
	case "video":
		if !hasInlineMedia && mediaURL == "" {
			return nil, fmt.Errorf("video broadcast requires media")
		}
		file := gotgbot.InputFileByURL(mediaURL)
		if hasInlineMedia {
			file = gotgbot.InputFileByReader(mediaName, bytes.NewReader(b.MediaData))
		}
		opts := &gotgbot.SendVideoOpts{Caption: content, ParseMode: parseMode}
		if keyboard != nil {
			opts.ReplyMarkup = keyboard
		}
		sent, sendErr = r.sendWithRetry(ctx, chatID, func() (*gotgbot.Message, error) {
			return r.tgBot.SendVideoWithContext(ctx, chatID, file, opts)
		})
	case "document", "file":
		if !hasInlineMedia && mediaURL == "" {
			return nil, fmt.Errorf("document broadcast requires media")
		}
		file := gotgbot.InputFileByURL(mediaURL)
		if hasInlineMedia {
			file = gotgbot.InputFileByReader(mediaName, bytes.NewReader(b.MediaData))
		}
		opts := &gotgbot.SendDocumentOpts{Caption: content, ParseMode: parseMode}
		if keyboard != nil {
			opts.ReplyMarkup = keyboard
		}
		sent, sendErr = r.sendWithRetry(ctx, chatID, func() (*gotgbot.Message, error) {
			return r.tgBot.SendDocumentWithContext(ctx, chatID, file, opts)
		})
	default:
		if content == "" {
			return nil, fmt.Errorf("text broadcast requires content")
		}
		opts := &gotgbot.SendMessageOpts{ParseMode: parseMode}
		if keyboard != nil {
			opts.ReplyMarkup = keyboard
		}
		sent, sendErr = r.sendWithRetry(ctx, chatID, func() (*gotgbot.Message, error) {
			return r.tgBot.SendMessageWithContext(ctx, chatID, content, opts)
		})
	}

	return sent, sendErr
}

// sendWithRetry retries a send once Telegram asks us to back off.
func (r *Runner) sendWithRetry(ctx context.Context, chatID int64, fn func() (*gotgbot.Message, error)) (*gotgbot.Message, error) {
	const maxAttempts = 3
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		msg, err := fn()
		if err == nil {
			return msg, nil
		}
		lastErr = err

		wait, ok := retryAfter(err)
		if !ok || attempt == maxAttempts {
			return nil, err
		}
		r.log.Warn("broadcast flood wait",
			zap.Int64("chat_id", chatID),
			zap.Duration("wait", wait),
			zap.Int("attempt", attempt),
		)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
	return nil, lastErr
}

// retryAfter pulls the "retry after N" hint out of a Telegram 429 error.
func retryAfter(err error) (time.Duration, bool) {
	if err == nil {
		return 0, false
	}
	text := err.Error()
	if !strings.Contains(text, "Too Many Requests") && !strings.Contains(text, "retry after") {
		return 0, false
	}
	idx := strings.LastIndex(text, "retry after")
	if idx < 0 {
		return 0, false
	}
	fields := strings.Fields(text[idx:])
	for _, f := range fields {
		seconds, parseErr := strconv.Atoi(strings.Trim(f, ".,;:"))
		if parseErr == nil && seconds > 0 {
			if seconds > 60 {
				seconds = 60
			}
			return time.Duration(seconds)*time.Second + time.Second, true
		}
	}
	return 5 * time.Second, true
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
