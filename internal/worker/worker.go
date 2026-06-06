package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/go-co-op/gocron/v2"
	robfigcron "github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/dabowin/sola/internal/config"
	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/service"
	"github.com/dabowin/sola/internal/store"
)

type Runner struct {
	cfg              config.Config
	log              *zap.Logger
	store            *store.Store
	sched            gocron.Scheduler
	tgBot            *gotgbot.Bot
	mu               sync.Mutex
	postFailureCount map[uint64]int
}

func New(cfg config.Config, st *store.Store, log *zap.Logger) *Runner {
	return &Runner{cfg: cfg, store: st, log: log, postFailureCount: make(map[uint64]int)}
}

func (r *Runner) Run(ctx context.Context) error {
	sched, err := gocron.NewScheduler()
	if err != nil {
		return err
	}
	r.sched = sched
	if token := strings.TrimSpace(r.cfg.Bot.Token); token != "" {
		tgBot, err := gotgbot.NewBot(token, nil)
		if err != nil {
			r.log.Warn("telegram bot unavailable for scheduled posts", zap.Error(err))
		} else {
			r.tgBot = tgBot
		}
	}

	_, err = sched.NewJob(
		gocron.DurationJob(time.Second),
		gocron.NewTask(r.runDueJobs),
	)
	if err != nil {
		return err
	}

	r.runDueJobs()
	r.registerEnabledScheduledPosts(ctx)
	sched.Start()
	r.log.Info("worker scheduler started")

	go r.scanVerifyTimeoutsLoop(ctx)

	<-ctx.Done()
	return sched.Shutdown()
}

func (r *Runner) registerEnabledScheduledPosts(ctx context.Context) {
	if r.store == nil || r.store.DB == nil || r.sched == nil {
		return
	}
	var posts []model.ScheduledPost
	if err := r.store.DB.WithContext(ctx).Where("enabled = ?", true).Find(&posts).Error; err != nil {
		r.log.Error("load scheduled posts", zap.Error(err))
		return
	}
	for _, post := range posts {
		if err := r.registerScheduledPost(post); err != nil {
			r.log.Warn("register scheduled post", zap.Uint64("post_id", post.ID), zap.Error(err))
		}
	}
}

func (r *Runner) registerScheduledPost(post model.ScheduledPost) error {
	if r.sched == nil || !post.Enabled {
		return nil
	}
	task := gocron.NewTask(func() {
		r.runScheduledPost(post.ID)
	})
	tag := fmt.Sprintf("scheduled_post:%d", post.ID)
	name := fmt.Sprintf("scheduled-post-%d", post.ID)

	if cronExpr := strings.TrimSpace(post.CronExpr); cronExpr != "" {
		if duration, ok := parseEveryDuration(cronExpr); ok {
			_, err := r.sched.NewJob(gocron.DurationJob(duration), task, gocron.WithName(name), gocron.WithTags("scheduled_post", tag))
			return err
		}
		_, err := r.sched.NewJob(gocron.CronJob(cronExpr, false), task, gocron.WithName(name), gocron.WithTags("scheduled_post", tag))
		return err
	}
	if post.RunOnceAt == nil || post.LastRunAt != nil {
		return nil
	}
	runAt := *post.RunOnceAt
	if !runAt.After(time.Now()) {
		_, err := r.sched.NewJob(gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()), task, gocron.WithName(name), gocron.WithTags("scheduled_post", tag))
		return err
	}
	_, err := r.sched.NewJob(gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(runAt)), task, gocron.WithName(name), gocron.WithTags("scheduled_post", tag))
	return err
}

func (r *Runner) runScheduledPost(postID uint64) {
	if r.store == nil || r.store.DB == nil {
		return
	}
	var post model.ScheduledPost
	if err := r.store.DB.First(&post, "id = ?", postID).Error; err != nil {
		r.log.Error("load scheduled post", zap.Uint64("post_id", postID), zap.Error(err))
		return
	}
	if !post.Enabled {
		return
	}
	if !r.scheduledPostDue(post, time.Now()) {
		return
	}
	if err := r.sendScheduledPost(context.Background(), post); err != nil {
		failures := r.incrementScheduledPostFailure(post.ID)
		r.log.Error("send scheduled post", zap.Uint64("post_id", post.ID), zap.Int64("chat_id", post.ChatID), zap.Int("consecutive_failures", failures), zap.Error(err))
		if failures >= 5 {
			if disableErr := r.disableScheduledPost(post.ID, err.Error()); disableErr != nil {
				r.log.Error("disable scheduled post after repeated failures", zap.Uint64("post_id", post.ID), zap.Error(disableErr))
			} else {
				r.log.Warn("scheduled post disabled after repeated failures", zap.Uint64("post_id", post.ID), zap.Int64("chat_id", post.ChatID), zap.Int("consecutive_failures", failures))
			}
		}
		return
	}
	r.resetScheduledPostFailure(post.ID)
	now := time.Now()
	updates := map[string]any{"last_run_at": now}
	if strings.TrimSpace(post.CronExpr) == "" {
		updates["enabled"] = false
	}
	if err := r.store.DB.Model(&model.ScheduledPost{}).Where("id = ?", post.ID).Updates(updates).Error; err != nil {
		r.log.Error("update scheduled post last run", zap.Uint64("post_id", post.ID), zap.Error(err))
	}
}

func (r *Runner) incrementScheduledPostFailure(postID uint64) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.postFailureCount[postID]++
	return r.postFailureCount[postID]
}

func (r *Runner) resetScheduledPostFailure(postID uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.postFailureCount, postID)
}

func (r *Runner) disableScheduledPost(postID uint64, reason string) error {
	r.resetScheduledPostFailure(postID)
	if r.store == nil || r.store.DB == nil {
		return nil
	}
	updates := map[string]any{
		"enabled": false,
	}
	return r.store.DB.Model(&model.ScheduledPost{}).Where("id = ?", postID).Updates(updates).Error
}

func (r *Runner) scanDueScheduledPosts(ctx context.Context, now time.Time) {
	var posts []model.ScheduledPost
	if err := r.store.DB.WithContext(ctx).
		Where("enabled = ? AND ((run_once_at IS NOT NULL AND run_once_at <= ? AND last_run_at IS NULL) OR COALESCE(cron_expr, '') <> '')", true, now).
		Limit(200).
		Find(&posts).Error; err != nil {
		r.log.Error("load due scheduled posts", zap.Error(err))
		return
	}
	for _, post := range posts {
		if !r.scheduledPostDue(post, now) {
			continue
		}
		r.runScheduledPost(post.ID)
	}
}

func (r *Runner) scheduledPostDue(post model.ScheduledPost, now time.Time) bool {
	if !post.Enabled {
		return false
	}
	if cronExpr := strings.TrimSpace(post.CronExpr); cronExpr != "" {
		if duration, ok := parseEveryDuration(cronExpr); ok {
			base := post.CreatedAt
			if post.LastRunAt != nil {
				base = *post.LastRunAt
			}
			if base.IsZero() {
				base = now.Add(-duration)
			}
			return !base.Add(duration).After(now)
		}
		schedule, err := robfigcron.ParseStandard(cronExpr)
		if err != nil {
			r.log.Warn("invalid scheduled post cron", zap.Uint64("post_id", post.ID), zap.String("cron", cronExpr), zap.Error(err))
			return false
		}
		base := now.Add(-time.Minute)
		if post.LastRunAt != nil {
			base = *post.LastRunAt
		} else if !post.CreatedAt.IsZero() {
			base = post.CreatedAt.Add(-time.Nanosecond)
		}
		return !schedule.Next(base).After(now)
	}
	return post.RunOnceAt != nil && post.LastRunAt == nil && !post.RunOnceAt.After(now)
}

func parseEveryDuration(expr string) (time.Duration, bool) {
	const prefix = "@every "
	text := strings.TrimSpace(expr)
	if !strings.HasPrefix(text, prefix) {
		return 0, false
	}
	duration, err := time.ParseDuration(strings.TrimSpace(strings.TrimPrefix(text, prefix)))
	if err != nil || duration <= 0 {
		return 0, false
	}
	return duration, true
}

func (r *Runner) sendScheduledPost(ctx context.Context, post model.ScheduledPost) error {
	if r.tgBot == nil {
		return fmt.Errorf("telegram bot is not configured")
	}
	content := strings.TrimSpace(post.Content)
	if content == "" {
		content = strings.TrimSpace(post.Title)
	}
	mediaURL := strings.TrimSpace(post.MediaURL)
	mediaName := strings.TrimSpace(post.MediaName)
	if mediaName == "" {
		mediaName = "media-upload"
	}
	hasInlineMedia := len(post.MediaData) > 0

	keyboard, err := parseInlineKeyboard(post.InlineKeyboardJSON)
	if err != nil {
		r.log.Warn("scheduled post has invalid inline keyboard", zap.Uint64("post_id", post.ID), zap.Error(err))
	}

	parseMode := strings.TrimSpace(post.ParseMode)
	if parseMode == "" {
		parseMode = "HTML"
	}

	var sent *gotgbot.Message
	switch strings.ToLower(strings.TrimSpace(post.MediaType)) {
	case "photo":
		if !hasInlineMedia && mediaURL == "" {
			return fmt.Errorf("photo scheduled post requires media")
		}
		photoFile := gotgbot.InputFileByURL(mediaURL)
		if hasInlineMedia {
			photoFile = gotgbot.InputFileByReader(mediaName, bytes.NewReader(post.MediaData))
		}
		opts := &gotgbot.SendPhotoOpts{Caption: content, ParseMode: parseMode}
		if keyboard != nil {
			opts.ReplyMarkup = keyboard
		}
		sent, err = r.tgBot.SendPhotoWithContext(ctx, post.ChatID, photoFile, opts)
	case "video":
		if !hasInlineMedia && mediaURL == "" {
			return fmt.Errorf("video scheduled post requires media")
		}
		videoFile := gotgbot.InputFileByURL(mediaURL)
		if hasInlineMedia {
			videoFile = gotgbot.InputFileByReader(mediaName, bytes.NewReader(post.MediaData))
		}
		opts := &gotgbot.SendVideoOpts{Caption: content, ParseMode: parseMode}
		if keyboard != nil {
			opts.ReplyMarkup = keyboard
		}
		sent, err = r.tgBot.SendVideoWithContext(ctx, post.ChatID, videoFile, opts)
	case "document", "file":
		if !hasInlineMedia && mediaURL == "" {
			return fmt.Errorf("document scheduled post requires media")
		}
		documentFile := gotgbot.InputFileByURL(mediaURL)
		if hasInlineMedia {
			documentFile = gotgbot.InputFileByReader(mediaName, bytes.NewReader(post.MediaData))
		}
		opts := &gotgbot.SendDocumentOpts{Caption: content, ParseMode: parseMode}
		if keyboard != nil {
			opts.ReplyMarkup = keyboard
		}
		sent, err = r.tgBot.SendDocumentWithContext(ctx, post.ChatID, documentFile, opts)
	default:
		if content == "" {
			return fmt.Errorf("text scheduled post requires content")
		}
		opts := &gotgbot.SendMessageOpts{ParseMode: parseMode}
		if keyboard != nil {
			opts.ReplyMarkup = keyboard
		}
		sent, err = r.tgBot.SendMessageWithContext(ctx, post.ChatID, content, opts)
	}
	if err != nil {
		return err
	}
	r.afterScheduledPostSent(ctx, post, sent)
	return nil
}

func (r *Runner) afterScheduledPostSent(ctx context.Context, post model.ScheduledPost, msg *gotgbot.Message) {
	if r.tgBot == nil || msg == nil {
		return
	}
	if post.PinAfterSend {
		if _, err := r.tgBot.PinChatMessageWithContext(ctx, post.ChatID, msg.MessageId, &gotgbot.PinChatMessageOpts{DisableNotification: true}); err != nil {
			r.log.Warn("pin scheduled post", zap.Uint64("post_id", post.ID), zap.Int64("chat_id", post.ChatID), zap.Int64("message_id", msg.MessageId), zap.Error(err))
		}
	}
	if post.AutoDeleteSeconds > 0 {
		autoDeleteAt := time.Now().Add(time.Duration(post.AutoDeleteSeconds) * time.Second)
		delivery := model.ScheduledPostDelivery{
			ScheduledPostID: post.ID,
			ChatID:          post.ChatID,
			MessageID:       msg.MessageId,
			AutoDeleteAt:    &autoDeleteAt,
			CreatedAt:       time.Now(),
		}
		if err := r.store.DB.WithContext(ctx).Create(&delivery).Error; err != nil {
			r.log.Warn("record scheduled post delivery", zap.Uint64("post_id", post.ID), zap.Int64("chat_id", post.ChatID), zap.Int64("message_id", msg.MessageId), zap.Error(err))
		}
	}
}

func (r *Runner) runDueJobs() {
	if r.store == nil || r.store.DB == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	ctx := context.Background()
	r.scanDueScheduledPosts(ctx, now)
	r.scanDueScheduledPostAutoDeletes(ctx, now)
	r.scanDueVerifyTimeouts(ctx, now)
	r.drawDueLotteries(ctx, now)
	r.processLegacyScheduledJobs(ctx, now)
}

func (r *Runner) scanDueScheduledPostAutoDeletes(ctx context.Context, now time.Time) {
	if r.tgBot == nil {
		return
	}
	var deliveries []model.ScheduledPostDelivery
	if err := r.store.DB.WithContext(ctx).
		Where("auto_delete_at IS NOT NULL AND auto_delete_at <= ? AND auto_deleted_at IS NULL", now).
		Order("auto_delete_at asc").
		Limit(100).
		Find(&deliveries).Error; err != nil {
		r.log.Error("load scheduled post auto deletes", zap.Error(err))
		return
	}
	for _, delivery := range deliveries {
		deleteCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		_, err := r.tgBot.DeleteMessageWithContext(deleteCtx, delivery.ChatID, delivery.MessageID, nil)
		cancel()
		if err != nil {
			r.log.Warn("auto delete scheduled post", zap.Uint64("delivery_id", delivery.ID), zap.Uint64("post_id", delivery.ScheduledPostID), zap.Int64("chat_id", delivery.ChatID), zap.Int64("message_id", delivery.MessageID), zap.Error(err))
			continue
		}
		deletedAt := time.Now()
		if err := r.store.DB.WithContext(ctx).
			Model(&model.ScheduledPostDelivery{}).
			Where("id = ? AND auto_deleted_at IS NULL", delivery.ID).
			Update("auto_deleted_at", deletedAt).Error; err != nil {
			r.log.Warn("mark scheduled post auto deleted", zap.Uint64("delivery_id", delivery.ID), zap.Error(err))
		}
	}
}

func (r *Runner) scanDueVerifyTimeouts(ctx context.Context, now time.Time) {
	if r.tgBot == nil || r.store == nil || r.store.Redis == nil {
		return
	}
	adminService := service.NewAdminService(r.store, r.store.Redis)
	timeouts, err := adminService.DueVerifyTimeouts(ctx, now, 100)
	if err != nil {
		r.log.Error("load due verify timeouts", zap.Error(err))
		return
	}
	for _, item := range timeouts {
		if err := r.kickUnverifiedMember(ctx, item.ChatID, item.UserID); err != nil {
			r.log.Warn("kick unverified member", zap.Int64("chat_id", item.ChatID), zap.Int64("user_id", item.UserID), zap.Error(err))
			continue
		}
		if err := adminService.RecordVerifyEvent(ctx, item.ChatID, item.UserID, "verify_timeout", "验证超时自动踢出"); err != nil {
			r.log.Warn("record verify timeout event", zap.Int64("chat_id", item.ChatID), zap.Int64("user_id", item.UserID), zap.Error(err))
		}
		if item.Challenge.MessageID != 0 {
			if _, err := r.tgBot.DeleteMessageWithContext(ctx, item.ChatID, item.Challenge.MessageID, nil); err != nil {
				r.log.Warn("delete verify challenge message", zap.Int64("chat_id", item.ChatID), zap.Int64("message_id", item.Challenge.MessageID), zap.Error(err))
			}
		}
		if err := adminService.ClearVerifyChallenge(ctx, item.ChatID, item.UserID); err != nil {
			r.log.Warn("clear verify challenge", zap.Int64("chat_id", item.ChatID), zap.Int64("user_id", item.UserID), zap.Error(err))
		}
		if r.store.Redis != nil {
			_ = r.store.Redis.Del(ctx, fmt.Sprintf("unverified:%d:%d", item.ChatID, item.UserID)).Err()
		}
	}
}

func (r *Runner) scanVerifyTimeoutsLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.scanVerifyTimeoutsFromDB(ctx)
		}
	}
}

func (r *Runner) scanVerifyTimeoutsFromDB(ctx context.Context) {
	if r.tgBot == nil || r.store == nil || r.store.DB == nil || r.store.Redis == nil {
		return
	}
	var configs []model.ChatAdminConfig
	if err := r.store.DB.WithContext(ctx).Where("verify_enabled = ?", true).Find(&configs).Error; err != nil {
		r.log.Error("load verify enabled chats", zap.Error(err))
		return
	}
	adminService := service.NewAdminService(r.store, r.store.Redis)
	for _, cfg := range configs {
		r.scanVerifyKeysForChat(ctx, adminService, cfg.ChatID)
	}
}

func (r *Runner) scanVerifyKeysForChat(ctx context.Context, adminService *service.AdminService, chatID int64) {
	pattern := fmt.Sprintf("verify:%d:*", chatID)
	keys, err := r.store.Redis.Keys(ctx, pattern).Result()
	if err != nil {
		r.log.Warn("scan verify keys", zap.Int64("chat_id", chatID), zap.Error(err))
		return
	}
	for _, key := range keys {
		ttl, err := r.store.Redis.TTL(ctx, key).Result()
		if err != nil {
			continue
		}
		if ttl > 0 {
			continue
		}
		// Key exists but has no TTL or is expired (should not happen with TTL-set keys, but handle it)
		// Actually, if TTL is -2, the key doesn't exist (expired). Since KEYS found it, it might be -1 (no expiry) or positive.
		// Only handle keys with TTL -2 (expired but still in scan) or -1 (no TTL set).
		if ttl == -2 {
			continue // key already gone
		}
		// For keys with no TTL or keys we want to force check
		parts := strings.Split(key, ":")
		if len(parts) < 3 {
			continue
		}
		userID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			continue
		}
		// Check if challenge exists
		challenge, ok, err := adminService.GetVerifyChallenge(ctx, chatID, userID)
		if err != nil || !ok {
			continue
		}
		if time.Now().After(challenge.ExpireAt) {
			if err := r.kickUnverifiedMember(ctx, chatID, userID); err != nil {
				r.log.Warn("kick unverified member (db scan)", zap.Int64("chat_id", chatID), zap.Int64("user_id", userID), zap.Error(err))
				continue
			}
			if err := adminService.RecordVerifyEvent(ctx, chatID, userID, "verify_timeout", "验证超时自动踢出"); err != nil {
				r.log.Warn("record verify timeout event", zap.Int64("chat_id", chatID), zap.Int64("user_id", userID), zap.Error(err))
			}
			if challenge.MessageID != 0 {
				if _, err := r.tgBot.DeleteMessageWithContext(ctx, chatID, challenge.MessageID, nil); err != nil {
					r.log.Warn("delete verify challenge message", zap.Int64("chat_id", chatID), zap.Int64("message_id", challenge.MessageID), zap.Error(err))
				}
			}
			if err := adminService.ClearVerifyChallenge(ctx, chatID, userID); err != nil {
				r.log.Warn("clear verify challenge", zap.Int64("chat_id", chatID), zap.Int64("user_id", userID), zap.Error(err))
			}
			if r.store.Redis != nil {
				_ = r.store.Redis.Del(ctx, fmt.Sprintf("unverified:%d:%d", chatID, userID)).Err()
			}
		}
	}
}

func (r *Runner) kickUnverifiedMember(ctx context.Context, chatID int64, userID int64) error {
	if _, err := r.tgBot.BanChatMemberWithContext(ctx, chatID, userID, &gotgbot.BanChatMemberOpts{RevokeMessages: false}); err != nil {
		return err
	}
	_, err := r.tgBot.UnbanChatMemberWithContext(ctx, chatID, userID, &gotgbot.UnbanChatMemberOpts{})
	return err
}

func (r *Runner) processLegacyScheduledJobs(ctx context.Context, now time.Time) {
	var jobs []model.ScheduledJob
	if err := r.store.DB.WithContext(ctx).Where("status = ? AND run_at <= ?", "queued", now).Limit(50).Find(&jobs).Error; err != nil {
		r.log.Error("load due jobs", zap.Error(err))
		return
	}
	for _, job := range jobs {
		_ = r.store.DB.WithContext(ctx).Model(&model.ScheduledJob{}).Where("id = ?", job.ID).Update("status", "sent").Error
		chatID := ""
		if job.ChatID != nil {
			chatID = job.ChatID.String()
		}
		r.log.Info("processed scheduled job", zap.String("job_id", job.ID.String()), zap.String("chat_id", chatID))
	}
}

func (r *Runner) drawDueLotteries(ctx context.Context, now time.Time) {
	lotteryService := service.NewLotteryService(r.store, r.store.Redis)
	drawn, err := lotteryService.DrawDue(ctx, now)
	if err != nil {
		r.log.Error("draw due lotteries", zap.Error(err))
		return
	}
	for _, lottery := range drawn {
		if err := r.announceLotteryResult(ctx, lottery.ID); err != nil {
			r.log.Error("announce lottery result", zap.Int64("lottery_id", lottery.ID), zap.Int64("chat_id", lottery.ChatID), zap.Error(err))
		}
	}
}

func (r *Runner) announceLotteryResult(ctx context.Context, lotteryID int64) error {
	if r.tgBot == nil {
		return fmt.Errorf("telegram bot is not configured")
	}
	var lottery model.Lottery
	if err := r.store.DB.WithContext(ctx).First(&lottery, "id = ?", lotteryID).Error; err != nil {
		return err
	}
	var winners []model.LotteryEntry
	if err := r.store.DB.WithContext(ctx).
		Where("lottery_id = ? AND is_winner = ?", lotteryID, true).
		Order("joined_at asc").
		Find(&winners).Error; err != nil {
		return err
	}

	var builder strings.Builder
	builder.WriteString("抽奖已开奖\n")
	if strings.TrimSpace(lottery.Title) != "" {
		builder.WriteString(fmt.Sprintf("标题：%s\n", strings.TrimSpace(lottery.Title)))
	}
	if strings.TrimSpace(lottery.Prize) != "" {
		builder.WriteString(fmt.Sprintf("奖品：%s\n", strings.TrimSpace(lottery.Prize)))
	}
	if len(winners) == 0 {
		builder.WriteString("本次没有中奖用户。")
	} else {
		builder.WriteString("中奖用户：")
		for _, winner := range winners {
			builder.WriteString(fmt.Sprintf("\n- %s", lotteryWinnerName(winner)))
		}
	}
	_, err := r.tgBot.SendMessageWithContext(ctx, lottery.ChatID, builder.String(), nil)
	return err
}

func lotteryWinnerName(winner model.LotteryEntry) string {
	if username := strings.TrimSpace(winner.Username); username != "" {
		return "@" + username
	}
	return fmt.Sprintf("用户 %d", winner.UserID)
}

func (r *Runner) String() string {
	return fmt.Sprintf("worker(%s)", r.cfg.App.Env)
}

// inlineKeyboardButton is a minimal representation of a Telegram inline
// keyboard button used for JSON deserialization.
type inlineKeyboardButton struct {
	Text         string `json:"text"`
	URL          string `json:"url,omitempty"`
	CallbackData string `json:"callback_data,omitempty"`
}

// parseInlineKeyboard unmarshals a JSON inline keyboard definition
// and returns a gotgbot.InlineKeyboardMarkup suitable for ReplyMarkup.
// Returns nil when the input is empty or "[]".
func parseInlineKeyboard(raw string) (*gotgbot.InlineKeyboardMarkup, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "null" {
		return nil, nil
	}
	var rows [][]inlineKeyboardButton
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil, fmt.Errorf("parse inline keyboard: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	keyboard := make([][]gotgbot.InlineKeyboardButton, 0, len(rows))
	for _, row := range rows {
		buttons := make([]gotgbot.InlineKeyboardButton, 0, len(row))
		for _, btn := range row {
			buttons = append(buttons, gotgbot.InlineKeyboardButton{
				Text:         btn.Text,
				Url:          btn.URL,
				CallbackData: btn.CallbackData,
			})
		}
		keyboard = append(keyboard, buttons)
	}
	return &gotgbot.InlineKeyboardMarkup{InlineKeyboard: keyboard}, nil
}
