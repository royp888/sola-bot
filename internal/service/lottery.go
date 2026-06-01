package service

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/dabowin/sola/internal/api"
	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

type LotteryService struct {
	store *store.Store
	redis *redis.Client
}

func NewLotteryService(st *store.Store, clients ...*redis.Client) *LotteryService {
	var client *redis.Client
	if len(clients) > 0 {
		client = clients[0]
	}
	return &LotteryService{store: st, redis: client}
}

func (s *LotteryService) List(ctx context.Context, query api.LotteryListQuery) ([]api.Lottery, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return []api.Lottery{}, nil
	}

	db := s.store.DB.WithContext(ctx).Model(&model.Lottery{})
	ownedIDs, err := ownedTelegramChatIDs(ctx, s.store, query.OwnerUserID)
	if err != nil {
		return nil, err
	}
	db = scopeTelegramChatID(db, "chat_id", query.ChatID, ownedIDs)
	if strings.TrimSpace(query.Status) != "" {
		db = db.Where("status = ?", strings.TrimSpace(query.Status))
	}

	var records []model.Lottery
	if err := db.Order("created_at desc").
		Limit(normalLimit(query.Limit)).
		Offset(query.Offset).
		Find(&records).Error; err != nil {
		return nil, err
	}

	items := make([]api.Lottery, 0, len(records))
	for _, record := range records {
		item, err := s.lotteryToAPI(ctx, record)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *LotteryService) Create(ctx context.Context, req api.LotteryCreateRequest) (*api.Lottery, error) {
	if req.ChatID == 0 {
		return nil, errors.New("chat_id is required")
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		return nil, errors.New("title is required")
	}
	if req.WinnerCount <= 0 {
		req.WinnerCount = 1
	}
	joinType, joinKeyword, err := normalizeLotteryJoin(req.JoinType, req.JoinKeyword)
	if err != nil {
		return nil, err
	}
	if s == nil || s.store == nil || s.store.DB == nil {
		now := time.Now()
		return &api.Lottery{
			ID:              now.Unix(),
			ChatID:          req.ChatID,
			Title:           req.Title,
			Prize:           req.Prize,
			CostPoints:      nonNegative(req.CostPoints),
			MaxParticipants: nonNegative(req.MaxParticipants),
			WinnerCount:     req.WinnerCount,
			EndAt:           req.EndAt,
			Status:          "active",
			JoinType:        joinType,
			JoinKeyword:     joinKeyword,
			CreatedBy:       req.CreatedBy,
			CreatedAt:       now,
		}, nil
	}

	record := model.Lottery{
		ChatID:          req.ChatID,
		Title:           req.Title,
		Prize:           strings.TrimSpace(req.Prize),
		CostPoints:      nonNegative(req.CostPoints),
		MaxParticipants: nonNegative(req.MaxParticipants),
		WinnerCount:     req.WinnerCount,
		EndAt:           req.EndAt,
		Status:          "active",
		JoinType:        joinType,
		JoinKeyword:     joinKeyword,
		CreatedBy:       req.CreatedBy,
	}
	if err := s.store.DB.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, err
	}
	s.invalidateKeywordCache(ctx, record.ChatID)
	item, err := s.lotteryToAPI(ctx, record)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *LotteryService) Cancel(ctx context.Context, id int64) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	return s.cancelLottery(ctx, id, 0)
}

func (s *LotteryService) Entries(ctx context.Context, id int64) ([]api.LotteryEntry, error) {
	return s.listEntries(ctx, id, false)
}

func (s *LotteryService) Winners(ctx context.Context, id int64) ([]api.LotteryEntry, error) {
	return s.listEntries(ctx, id, true)
}

func (s *LotteryService) ListActive(ctx context.Context, chatID int64) (string, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return "抽奖服务尚未接入数据库。", nil
	}
	items, err := s.ListActiveItems(ctx, chatID, 10)
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "当前没有进行中的抽奖。", nil
	}

	var builder strings.Builder
	builder.WriteString("进行中的抽奖\n")
	for _, item := range items {
		builder.WriteString(fmt.Sprintf("#%d %s\n奖品：%s\n参与：%d", item.ID, emptyFallback(item.Title, "未命名抽奖"), emptyFallback(item.Prize, "未填写"), item.EntryCount))
		if item.MaxParticipants > 0 {
			builder.WriteString(fmt.Sprintf("/%d", item.MaxParticipants))
		}
		if item.CostPoints > 0 {
			builder.WriteString(fmt.Sprintf("\n消耗积分：%d", item.CostPoints))
		}
		if item.EndAt != nil {
			builder.WriteString(fmt.Sprintf("\n结束时间：%s", item.EndAt.Format("2006-01-02 15:04")))
		}
		builder.WriteString("\n\n")
	}
	builder.WriteString("命令：/lottery info <id> 或 /lottery join <id>")
	return strings.TrimSpace(builder.String()), nil
}

func (s *LotteryService) ListActiveItems(ctx context.Context, chatID int64, limit int) ([]api.Lottery, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return []api.Lottery{}, nil
	}
	return s.List(ctx, api.LotteryListQuery{ChatID: chatID, Status: "active", Limit: limit})
}

func (s *LotteryService) ListItems(ctx context.Context, chatID int64, limit int) ([]api.Lottery, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return []api.Lottery{}, nil
	}
	return s.List(ctx, api.LotteryListQuery{ChatID: chatID, Limit: limit})
}

func (s *LotteryService) GetItem(ctx context.Context, chatID int64, lotteryID int64) (api.Lottery, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return api.Lottery{}, nil
	}
	var record model.Lottery
	if err := s.store.DB.WithContext(ctx).Where("id = ? AND chat_id = ?", lotteryID, chatID).First(&record).Error; err != nil {
		return api.Lottery{}, err
	}
	return s.lotteryToAPI(ctx, record)
}

func (s *LotteryService) Info(ctx context.Context, chatID int64, lotteryID int64) (string, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return "抽奖服务尚未接入数据库。", nil
	}
	var record model.Lottery
	if err := s.store.DB.WithContext(ctx).Where("id = ? AND chat_id = ?", lotteryID, chatID).First(&record).Error; err != nil {
		return "", err
	}
	item, err := s.lotteryToAPI(ctx, record)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("抽奖 #%d\n", item.ID))
	builder.WriteString(fmt.Sprintf("标题：%s\n", emptyFallback(item.Title, "未命名抽奖")))
	builder.WriteString(fmt.Sprintf("奖品：%s\n", emptyFallback(item.Prize, "未填写")))
	builder.WriteString(fmt.Sprintf("状态：%s\n", item.Status))
	builder.WriteString(fmt.Sprintf("参与人数：%d", item.EntryCount))
	if item.MaxParticipants > 0 {
		builder.WriteString(fmt.Sprintf("/%d", item.MaxParticipants))
	}
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("中奖人数：%d", item.WinnerCount))
	if item.CostPoints > 0 {
		builder.WriteString(fmt.Sprintf("\n参与消耗积分：%d", item.CostPoints))
	}
	if item.EndAt != nil {
		builder.WriteString(fmt.Sprintf("\n结束时间：%s", item.EndAt.Format("2006-01-02 15:04")))
	}
	if item.Status == "ended" {
		winners, err := s.Winners(ctx, lotteryID)
		if err != nil {
			return "", err
		}
		if len(winners) == 0 {
			builder.WriteString("\n中奖用户：无")
		} else {
			builder.WriteString("\n中奖用户：")
			for _, winner := range winners {
				builder.WriteString(fmt.Sprintf("\n- 用户 %d", winner.UserID))
			}
		}
	}
	return builder.String(), nil
}

func (s *LotteryService) Join(ctx context.Context, chatID int64, lotteryID int64, userID int64, username ...string) (string, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return "抽奖服务尚未接入数据库。", nil
	}
	if userID == 0 {
		return "无法识别参与用户。", nil
	}

	var joined bool
	var shouldDraw bool
	participantName := ""
	if len(username) > 0 {
		participantName = sanitizeTelegramUsername(username[0])
	}
	err := s.store.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lottery model.Lottery
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND chat_id = ?", lotteryID, chatID).
			First(&lottery).Error; err != nil {
			return err
		}
		if lottery.Status != "active" {
			return errors.New("抽奖已结束或已取消")
		}
		if lottery.EndAt != nil && time.Now().After(*lottery.EndAt) {
			return errors.New("抽奖已到结束时间")
		}

		var existing model.LotteryEntry
		if err := tx.Where("lottery_id = ? AND user_id = ?", lotteryID, userID).First(&existing).Error; err == nil {
			return errors.New("你已经参与过这个抽奖")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if lottery.MaxParticipants > 0 {
			var count int64
			if err := tx.Model(&model.LotteryEntry{}).Where("lottery_id = ?", lotteryID).Count(&count).Error; err != nil {
				return err
			}
			if count >= int64(lottery.MaxParticipants) {
				return errors.New("抽奖名额已满")
			}
		}

		if err := s.reserveLotteryPointDebit(ctx, tx, lottery, userID); err != nil {
			return err
		}

		entry := model.LotteryEntry{LotteryID: lotteryID, UserID: userID, Username: participantName}
		if err := tx.Create(&entry).Error; err != nil {
			return err
		}
		if lottery.MaxParticipants > 0 && lottery.EndAt == nil {
			var count int64
			if err := tx.Model(&model.LotteryEntry{}).Where("lottery_id = ?", lotteryID).Count(&count).Error; err != nil {
				return err
			}
			shouldDraw = count >= int64(lottery.MaxParticipants)
		}
		joined = true
		return nil
	})
	if err != nil {
		return "", err
	}
	if !joined {
		return "未完成参与，请稍后重试。", nil
	}
	s.invalidateKeywordCache(ctx, chatID)
	if shouldDraw {
		if err := s.drawLottery(ctx, lotteryID); err != nil {
			return "", err
		}
		return fmt.Sprintf("已参与抽奖 #%d，人数已满，已自动开奖。", lotteryID), nil
	}
	return fmt.Sprintf("已参与抽奖 #%d。", lotteryID), nil
}

func (s *LotteryService) JoinByKeyword(ctx context.Context, chatID int64, keyword string, userID int64, username string) (bool, string, int64, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" || s == nil || s.store == nil || s.store.DB == nil {
		return false, "", 0, nil
	}
	items, err := s.keywordLotteries(ctx, chatID)
	if err != nil {
		return false, "", 0, err
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.JoinKeyword), keyword) {
			message, err := s.Join(ctx, chatID, item.ID, userID, username)
			return true, message, item.ID, err
		}
	}
	return false, "", 0, nil
}

func (s *LotteryService) CancelForChat(ctx context.Context, chatID int64, lotteryID int64, operatorID int64) (string, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return "抽奖服务尚未接入数据库。", nil
	}
	if err := s.cancelLottery(ctx, lotteryID, chatID); err != nil {
		return "", err
	}
	return fmt.Sprintf("已取消抽奖 #%d。", lotteryID), nil
}

func (s *LotteryService) DrawDue(ctx context.Context, now time.Time) ([]model.Lottery, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, nil
	}
	var due []model.Lottery
	if err := s.store.DB.WithContext(ctx).
		Where("status = ? AND end_at IS NOT NULL AND end_at <= ?", "active", now).
		Find(&due).Error; err != nil {
		return nil, err
	}

	drawn := make([]model.Lottery, 0, len(due))
	for _, lottery := range due {
		if err := s.drawLottery(ctx, lottery.ID); err != nil {
			return drawn, err
		}
		drawn = append(drawn, lottery)
	}
	return drawn, nil
}

func (s *LotteryService) drawLottery(ctx context.Context, lotteryID int64) error {
	return s.store.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lottery model.Lottery
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lottery, "id = ?", lotteryID).Error; err != nil {
			return err
		}
		if lottery.Status != "active" {
			return nil
		}

		var entries []model.LotteryEntry
		if err := tx.Where("lottery_id = ?", lotteryID).Find(&entries).Error; err != nil {
			return err
		}
		winnerCount := lottery.WinnerCount
		if winnerCount <= 0 {
			winnerCount = 1
		}
		if winnerCount > len(entries) {
			winnerCount = len(entries)
		}
		if err := secureShuffleLotteryEntries(entries); err != nil {
			return err
		}

		for i := 0; i < winnerCount; i++ {
			if err := tx.Model(&model.LotteryEntry{}).Where("id = ?", entries[i].ID).Update("is_winner", true).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&model.Lottery{}).Where("id = ?", lotteryID).Update("status", "ended").Error; err != nil {
			return err
		}
		s.invalidateKeywordCache(ctx, lottery.ChatID)
		return nil
	})
}

func (s *LotteryService) reserveLotteryPointDebit(ctx context.Context, tx *gorm.DB, lottery model.Lottery, userID int64) error {
	_ = ctx
	if lottery.CostPoints <= 0 {
		return nil
	}
	return adjustLotteryPointsTx(tx, lottery.ChatID, userID, -lottery.CostPoints, fmt.Sprintf("lottery_join:%d", lottery.ID))
}

func (s *LotteryService) cancelLottery(ctx context.Context, lotteryID int64, chatID int64) error {
	var cancelledChatID int64
	if err := s.store.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lottery model.Lottery
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status = ?", lotteryID, "active")
		if chatID != 0 {
			query = query.Where("chat_id = ?", chatID)
		}
		if err := query.First(&lottery).Error; err != nil {
			return err
		}
		if err := refundLotteryEntriesTx(tx, lottery); err != nil {
			return err
		}
		if err := tx.Model(&model.Lottery{}).Where("id = ?", lotteryID).Update("status", "cancelled").Error; err != nil {
			return err
		}
		cancelledChatID = lottery.ChatID
		return nil
	}); err != nil {
		return err
	}
	s.invalidateKeywordCache(ctx, cancelledChatID)
	return nil
}

func refundLotteryEntriesTx(tx *gorm.DB, lottery model.Lottery) error {
	if lottery.CostPoints <= 0 {
		return nil
	}
	var entries []model.LotteryEntry
	if err := tx.Where("lottery_id = ?", lottery.ID).Find(&entries).Error; err != nil {
		return err
	}
	for _, entry := range entries {
		if err := adjustLotteryPointsTx(tx, lottery.ChatID, entry.UserID, lottery.CostPoints, fmt.Sprintf("lottery_cancel:%d", lottery.ID)); err != nil {
			return err
		}
	}
	return nil
}

func adjustLotteryPointsTx(tx *gorm.DB, chatID int64, userID int64, delta int, reason string) error {
	if delta == 0 {
		return nil
	}
	now := time.Now()
	if delta < 0 {
		cost := int64(-delta)
		var point model.UserPoint
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("chat_id = ? AND user_id = ?", chatID, userID).
			First(&point).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("积分不足")
		}
		if err != nil {
			return err
		}
		if point.TotalPoints < cost {
			return errors.New("积分不足")
		}
		result := tx.Model(&model.UserPoint{}).
			Where("chat_id = ? AND user_id = ? AND total_points >= ?", chatID, userID, cost).
			Updates(map[string]any{
				"total_points": gorm.Expr("total_points + ?", delta),
				"updated_at":   now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("积分不足")
		}
	} else {
		userPoint := model.UserPoint{
			UserID:      userID,
			ChatID:      chatID,
			TotalPoints: int64(delta),
			UpdatedAt:   now,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "chat_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"total_points": gorm.Expr("user_points.total_points + ?", delta),
				"updated_at":   now,
			}),
		}).Create(&userPoint).Error; err != nil {
			return err
		}
	}
	return tx.Create(&model.PointLog{
		UserID:    userID,
		ChatID:    chatID,
		Delta:     delta,
		Reason:    truncateReason(reason),
		CreatedAt: now,
	}).Error
}

func (s *LotteryService) listEntries(ctx context.Context, id int64, winnersOnly bool) ([]api.LotteryEntry, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return []api.LotteryEntry{}, nil
	}
	db := s.store.DB.WithContext(ctx).Where("lottery_id = ?", id)
	if winnersOnly {
		db = db.Where("is_winner = ?", true)
	}
	var records []model.LotteryEntry
	if err := db.Order("joined_at asc").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]api.LotteryEntry, 0, len(records))
	for _, record := range records {
		items = append(items, lotteryEntryToAPI(record))
	}
	return items, nil
}

func (s *LotteryService) lotteryToAPI(ctx context.Context, record model.Lottery) (api.Lottery, error) {
	item := api.Lottery{
		ID:              record.ID,
		ChatID:          record.ChatID,
		Title:           record.Title,
		Prize:           record.Prize,
		CostPoints:      record.CostPoints,
		MaxParticipants: record.MaxParticipants,
		WinnerCount:     record.WinnerCount,
		EndAt:           record.EndAt,
		Status:          record.Status,
		CreatedBy:       record.CreatedBy,
		CreatedAt:       record.CreatedAt,
		JoinType:        emptyFallback(record.JoinType, "button"),
		JoinKeyword:     record.JoinKeyword,
	}
	if s == nil || s.store == nil || s.store.DB == nil {
		return item, nil
	}
	if err := s.store.DB.WithContext(ctx).Model(&model.LotteryEntry{}).Where("lottery_id = ?", record.ID).Count(&item.EntryCount).Error; err != nil {
		return item, err
	}
	if err := s.store.DB.WithContext(ctx).Model(&model.LotteryEntry{}).Where("lottery_id = ? AND is_winner = ?", record.ID, true).Count(&item.WinnerCountDone).Error; err != nil {
		return item, err
	}
	return item, nil
}

func lotteryEntryToAPI(record model.LotteryEntry) api.LotteryEntry {
	return api.LotteryEntry{
		ID:        record.ID,
		LotteryID: record.LotteryID,
		UserID:    record.UserID,
		Username:  record.Username,
		JoinedAt:  record.JoinedAt,
		IsWinner:  record.IsWinner,
	}
}

func normalizeLotteryJoin(joinType string, keyword string) (string, string, error) {
	joinType = strings.ToLower(strings.TrimSpace(joinType))
	if joinType == "" {
		joinType = "button"
	}
	switch joinType {
	case "button":
		return joinType, "", nil
	case "keyword", "both":
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			return "", "", errors.New("join_keyword is required for keyword lotteries")
		}
		return joinType, keyword, nil
	default:
		return "", "", errors.New("join_type must be button, keyword, or both")
	}
}

func sanitizeTelegramUsername(username string) string {
	username = strings.TrimSpace(strings.TrimPrefix(username, "@"))
	if len([]rune(username)) > 64 {
		return string([]rune(username)[:64])
	}
	return username
}

func secureShuffleLotteryEntries(entries []model.LotteryEntry) error {
	for i := len(entries) - 1; i > 0; i-- {
		n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		j := int(n.Int64())
		entries[i], entries[j] = entries[j], entries[i]
	}
	return nil
}

type lotteryKeywordCacheItem struct {
	ID              int64      `json:"id"`
	JoinKeyword     string     `json:"join_keyword"`
	CostPoints      int        `json:"cost_points"`
	MaxParticipants int        `json:"max_participants"`
	EntryCount      int64      `json:"entry_count"`
	EndAt           *time.Time `json:"end_at,omitempty"`
	JoinType        string     `json:"join_type"`
}

func lotteryKeywordCacheKey(chatID int64) string {
	return fmt.Sprintf("lottery:keywords:%d", chatID)
}

func (s *LotteryService) keywordLotteries(ctx context.Context, chatID int64) ([]lotteryKeywordCacheItem, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, nil
	}
	key := lotteryKeywordCacheKey(chatID)
	if s.redis != nil {
		if raw, err := s.redis.Get(ctx, key).Result(); err == nil && raw != "" {
			var cached []lotteryKeywordCacheItem
			if json.Unmarshal([]byte(raw), &cached) == nil {
				return cached, nil
			}
		}
	}

	var records []model.Lottery
	if err := s.store.DB.WithContext(ctx).
		Where("chat_id = ? AND status = ? AND join_type IN ? AND join_keyword <> ''", chatID, "active", []string{"keyword", "both"}).
		Where("(end_at IS NULL OR end_at > ?)", time.Now()).
		Order("created_at desc").
		Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]lotteryKeywordCacheItem, 0, len(records))
	var ttl time.Duration
	for _, record := range records {
		var count int64
		if err := s.store.DB.WithContext(ctx).Model(&model.LotteryEntry{}).Where("lottery_id = ?", record.ID).Count(&count).Error; err != nil {
			return nil, err
		}
		items = append(items, lotteryKeywordCacheItem{
			ID:              record.ID,
			JoinKeyword:     record.JoinKeyword,
			CostPoints:      record.CostPoints,
			MaxParticipants: record.MaxParticipants,
			EntryCount:      count,
			EndAt:           record.EndAt,
			JoinType:        record.JoinType,
		})
		if record.EndAt != nil {
			until := time.Until(*record.EndAt)
			if until > 0 && (ttl == 0 || until < ttl) {
				ttl = until
			}
		}
	}
	if s.redis != nil && len(items) > 0 {
		if ttl == 0 {
			ttl = 10 * time.Minute
		}
		if data, err := json.Marshal(items); err == nil {
			_ = s.redis.Set(ctx, key, data, ttl).Err()
		}
	}
	return items, nil
}

func (s *LotteryService) invalidateKeywordCache(ctx context.Context, chatID int64) {
	if s != nil && s.redis != nil && chatID != 0 {
		_ = s.redis.Del(ctx, lotteryKeywordCacheKey(chatID)).Err()
	}
}

func emptyFallback(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
