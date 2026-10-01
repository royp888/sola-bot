package worker

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

func testDSN() string {
	if dsn := os.Getenv("SOLA_TEST_DATABASE_DSN"); dsn != "" {
		return dsn
	}
	return "postgres://sola:sola@127.0.0.1:5432/sola?sslmode=disable"
}

func newTestRunner(t *testing.T) (*Runner, *store.Store) {
	t.Helper()

	db, err := gorm.Open(postgres.Open(testDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := sqlDB.PingContext(context.Background()); err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}

	st := store.New(db, nil)
	runner := &Runner{
		store:            st,
		log:              zap.NewNop(),
		postFailureCount: map[uint64]int{},
	}
	return runner, st
}

// seedTargets inserts chats/users used by the resolution tests.
func seedTargets(t *testing.T, st *store.Store) {
	t.Helper()
	ctx := context.Background()

	if err := st.DB.WithContext(ctx).Exec(`
		INSERT INTO telegram_chats (id, telegram_chat_id, type, title, username, status, settings_json, created_at, updated_at)
		VALUES
		 (gen_random_uuid(), -1009000000101, 'supergroup', 'BC 测试群 1', 'bcgroup1', 'active', '{}', now(), now()),
		 (gen_random_uuid(), -1009000000102, 'supergroup', 'BC 测试群 2', 'bcgroup2', 'active', '{}', now(), now()),
		 (gen_random_uuid(), -1009000000103, 'supergroup', 'BC 已停用群', 'bcgroup3', 'inactive', '{}', now(), now())
		ON CONFLICT DO NOTHING;
	`).Error; err != nil {
		t.Fatalf("seed chats: %v", err)
	}

	if err := st.DB.WithContext(ctx).Exec(`
		INSERT INTO users (id, telegram_user_id, username, display_name, role, status, is_active, metadata_json, created_at, updated_at)
		VALUES
		 (gen_random_uuid(), 9000000101, 'bc_user_1', 'BC 用户 1', 'member', 'active', true, '{}', now(), now()),
		 (gen_random_uuid(), 9000000102, 'bc_user_2', 'BC 用户 2', 'member', 'active', true, '{}', now(), now()),
		 (gen_random_uuid(), 9000000103, 'bc_user_3', 'BC 已停用用户', 'member', 'disabled', false, '{}', now(), now())
		ON CONFLICT DO NOTHING;
	`).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
}

func cleanupSeed(t *testing.T, st *store.Store) {
	t.Helper()
	st.DB.Exec("DELETE FROM telegram_chats WHERE telegram_chat_id BETWEEN -1009000000199 AND -1009000000100")
	st.DB.Exec("DELETE FROM users WHERE telegram_user_id BETWEEN 9000000100 AND 9000000199")
}

func TestResolveBroadcastTargets(t *testing.T) {
	runner, st := newTestRunner(t)
	ctx := context.Background()

	cleanupSeed(t, st)
	seedTargets(t, st)
	t.Cleanup(func() { cleanupSeed(t, st) })

	seededUsers := map[int64]bool{9000000101: true, 9000000102: true, 9000000103: true}
	seededChats := map[int64]bool{-1009000000101: true, -1009000000102: true, -1009000000103: true}

	// Only look at the rows this test created, so the assertions do not depend
	// on whatever else happens to be in the database.
	filterSeeded := func(targets []broadcastTarget) []broadcastTarget {
		out := make([]broadcastTarget, 0, len(targets))
		for _, tgt := range targets {
			if tgt.Type == model.BroadcastTargetUsers && seededUsers[tgt.ID] {
				out = append(out, tgt)
			}
			if tgt.Type == model.BroadcastTargetGroups && seededChats[tgt.ID] {
				out = append(out, tgt)
			}
		}
		return out
	}

	countByType := func(targets []broadcastTarget) (users, chats int) {
		for _, tgt := range targets {
			switch tgt.Type {
			case model.BroadcastTargetUsers:
				users++
			case model.BroadcastTargetGroups:
				chats++
			}
		}
		return
	}

	t.Run("users only", func(t *testing.T) {
		targets, err := runner.resolveBroadcastTargets(ctx, model.Broadcast{
			TargetType: model.BroadcastTargetUsers,
		})
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		if _, chats := countByType(targets); chats != 0 {
			t.Errorf("expected no group targets, got %d", chats)
		}
		users, _ := countByType(filterSeeded(targets))
		if users != 2 {
			t.Errorf("expected 2 seeded active users, got %d", users)
		}
		for _, tgt := range targets {
			if tgt.ID == 9000000103 {
				t.Error("disabled user 9000000103 must not be a target")
			}
		}
	})

	t.Run("groups only skips inactive chats", func(t *testing.T) {
		targets, err := runner.resolveBroadcastTargets(ctx, model.Broadcast{
			TargetType:  model.BroadcastTargetGroups,
			ChatIDsJSON: "[]",
		})
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		if users, _ := countByType(targets); users != 0 {
			t.Errorf("expected no user targets, got %d", users)
		}
		// 3 seeded chats, one of which is inactive.
		_, chats := countByType(filterSeeded(targets))
		if chats != 2 {
			t.Errorf("expected 2 seeded active chats, got %d", chats)
		}
	})

	t.Run("both combines users and chats", func(t *testing.T) {
		targets, err := runner.resolveBroadcastTargets(ctx, model.Broadcast{
			TargetType: model.BroadcastTargetBoth,
		})
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		users, chats := countByType(filterSeeded(targets))
		if users != 2 || chats != 2 {
			t.Errorf("expected 2 users and 2 chats, got %d and %d", users, chats)
		}
	})

	t.Run("inactive targets are excluded", func(t *testing.T) {
		targets, err := runner.resolveBroadcastTargets(ctx, model.Broadcast{
			TargetType: model.BroadcastTargetBoth,
		})
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		for _, tgt := range targets {
			if tgt.ID == -1009000000103 {
				t.Error("inactive chat -1009000000103 must not be a target")
			}
			if tgt.ID == 9000000103 {
				t.Error("disabled user 9000000103 must not be a target")
			}
		}
	})

	t.Run("explicit chat list narrows the fan-out", func(t *testing.T) {
		targets, err := runner.resolveBroadcastTargets(ctx, model.Broadcast{
			TargetType:  model.BroadcastTargetGroups,
			ChatIDsJSON: "[-1009000000102]",
		})
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		_, chats := countByType(targets)
		if chats != 1 {
			t.Fatalf("expected exactly 1 chat, got %d", chats)
		}
		if targets[0].ID != -1009000000102 {
			t.Errorf("expected chat -1009000000102, got %d", targets[0].ID)
		}
	})

	t.Run("targets carry a display name", func(t *testing.T) {
		targets, err := runner.resolveBroadcastTargets(ctx, model.Broadcast{
			TargetType: model.BroadcastTargetBoth,
		})
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		for _, tgt := range targets {
			if tgt.Name == "" {
				t.Errorf("target %d (%s) has no display name", tgt.ID, tgt.Type)
			}
		}
	})
}

func TestParseChatIDList(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"empty", "", 0},
		{"empty array", "[]", 0},
		{"null", "null", 0},
		{"whitespace", "   ", 0},
		{"two ids", "[-1001234567890, -1009876543210]", 2},
		{"malformed", "not json", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := len(parseChatIDList(tc.in)); got != tc.want {
				t.Errorf("parseChatIDList(%q) len = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestRetryAfter(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		wantOK bool
		want   time.Duration
	}{
		{"nil", nil, false, 0},
		{"unrelated", errors.New("bot was blocked by the user"), false, 0},
		{
			"flood wait",
			errors.New("telegram: Too Many Requests: retry after 3"),
			true,
			4 * time.Second, // 3s + 1s jitter
		},
		{
			"huge wait is capped",
			errors.New("Too Many Requests: retry after 999"),
			true,
			61 * time.Second,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := retryAfter(tc.err)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && got != tc.want {
				t.Errorf("wait = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("abcdef", 3); got != "abc" {
		t.Errorf("truncate = %q, want %q", got, "abc")
	}
	if got := truncate("ab", 5); got != "ab" {
		t.Errorf("truncate = %q, want %q", got, "ab")
	}
}
