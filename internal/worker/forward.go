package worker

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/dabowin/sola/internal/model"
)

// runDueForwards 推进所有处于 running 状态的历史补发规则。
// 每轮只取少量规则、每次只处理一小批消息，避免和其他任务抢 FloodWait 配额。
func (r *Runner) runDueForwards() {
	if r.store == nil || r.store.DB == nil || r.tgBot == nil || r.forwardEngine == nil {
		return
	}
	if !r.muForwards.TryLock() {
		return
	}
	defer r.muForwards.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var rules []model.AutoForward
	err := r.store.DB.WithContext(ctx).
		Where("enabled = ? AND history_state = ?", true, model.ForwardStateRunning).
		Order("id asc").
		Limit(5).
		Find(&rules).Error
	if err != nil {
		if !isForwardTableMissing(err) {
			r.log.Warn("load forwarding rules", zap.Error(err))
		}
		return
	}
	if len(rules) == 0 {
		return
	}

	for _, rule := range rules {
		if ctx.Err() != nil {
			return
		}
		processed, err := r.forwardEngine.ProcessHistoryBatch(ctx, r.tgBot, rule)
		if err != nil {
			r.log.Warn("forward history batch failed", zap.Uint64("rule_id", rule.ID), zap.Error(err))
			continue
		}
		if processed > 0 {
			r.log.Info("forward history batch processed",
				zap.Uint64("rule_id", rule.ID),
				zap.Int("processed", processed),
				zap.Int64("cursor", rule.HistoryCursor))
		}
	}
}

func isForwardTableMissing(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "does not exist") ||
		strings.Contains(message, "no such table")
}
