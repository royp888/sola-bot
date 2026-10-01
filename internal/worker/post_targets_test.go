package worker

import (
	"testing"

	"github.com/dabowin/sola/internal/model"
)

func TestPostDeliveryTargets(t *testing.T) {
	post := model.ScheduledPost{
		ChatID:            -1001111111111,
		TargetChatIDsJSON: `[-1002222222222,-1003333333333]`,
	}
	targets := postDeliveryTargets(post)
	if len(targets) != 3 {
		t.Fatalf("targets = %#v, want 3 entries", targets)
	}
	if targets[0] != -1001111111111 || targets[2] != -1003333333333 {
		t.Fatalf("unexpected order: %#v", targets)
	}

	// 主目标重复出现在额外目标里时只发一次
	duplicated := model.ScheduledPost{
		ChatID:            -1001111111111,
		TargetChatIDsJSON: `[-1001111111111,-1002222222222]`,
	}
	if got := postDeliveryTargets(duplicated); len(got) != 2 {
		t.Fatalf("duplicates should be dropped, got %#v", got)
	}

	// 兜底：没有主目标时只用额外目标
	onlyExtra := model.ScheduledPost{TargetChatIDsJSON: `[-1002222222222]`}
	if got := postDeliveryTargets(onlyExtra); len(got) != 1 || got[0] != -1002222222222 {
		t.Fatalf("unexpected targets: %#v", got)
	}

	empty := postDeliveryTargets(model.ScheduledPost{ChatID: -1001111111111})
	if len(empty) != 1 {
		t.Fatalf("empty extra targets should keep the primary one, got %#v", empty)
	}
	failed := postDeliveryTargets(model.ScheduledPost{TargetChatIDsJSON: "{oops"})
	if len(failed) != 0 {
		t.Fatalf("invalid json should be ignored, got %#v", failed)
	}
}
