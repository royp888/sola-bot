package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dabowin/sola/internal/api"
	"github.com/dabowin/sola/internal/config"
	"github.com/dabowin/sola/internal/service"
	"github.com/dabowin/sola/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTenantIsolationByIDWritesReturnForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: "file:tenant-isolation?mode=memory&cache=shared"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	createTenantIsolationTables(t, db)

	ownerA := uuid.New()
	ownerB := uuid.New()
	chatA := int64(-1001001)
	chatB := int64(-1002002)
	seedTenantIsolationData(t, db, ownerA, ownerB, chatA, chatB)

	cfg := config.Config{}
	cfg.JWT.Secret = "tenant-isolation-test-secret"
	cfg.JWT.Issuer = "sola-test"
	cfg.JWT.AccessTokenTTL = time.Hour
	deps := service.NewAPIDependencies(cfg, store.New(db, nil))
	router := api.NewRouter(deps)
	token := signOwnerToken(t, cfg, ownerA.String())

	levelID := "11111111-1111-4111-8111-111111111111"
	keywordID := "22222222-2222-4222-8222-222222222222"
	autoReplyID := "33333333-3333-4333-8333-333333333333"
	scheduleID := "44444444-4444-4444-8444-444444444444"
	templateID := "55555555-5555-4555-8555-555555555555"
	inviteLinkID := "66666666-6666-4666-8666-666666666666"

	cases := []struct {
		name string
		path string
	}{
		{name: "cancel lottery", path: "/api/v1/lottery/42"},
		{name: "delete level", path: "/api/v1/levels/" + levelID},
		{name: "delete keyword", path: "/api/v1/keywords/" + keywordID},
		{name: "delete auto reply", path: "/api/v1/auto-replies/" + autoReplyID},
		{name: "delete post", path: "/api/v1/posts/77"},
		{name: "delete schedule", path: "/api/v1/schedules/" + scheduleID},
		{name: "delete template", path: "/api/v1/templates/" + templateID},
		{name: "delete invite link", path: "/api/v1/invite-links/" + inviteLinkID},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, tc.path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403, body = %s", rec.Code, rec.Body.String())
			}
		})
	}

	assertResourceStillPresent(t, db, "lotteries", "id", int64(42))
	assertResourceStillPresent(t, db, "level_configs", "id", levelID)
	assertResourceStillPresent(t, db, "keyword_filters", "id", keywordID)
	assertResourceStillPresent(t, db, "auto_replies", "id", autoReplyID)
	assertResourceStillPresent(t, db, "scheduled_posts", "id", uint64(77))
	assertResourceStillPresent(t, db, "scheduled_jobs", "id", scheduleID)
	assertResourceStillPresent(t, db, "message_templates", "id", templateID)
	assertResourceStillPresent(t, db, "invite_links", "id", inviteLinkID)
	_ = chatA
}

func createTenantIsolationTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	statements := []string{
		`CREATE TABLE telegram_chats (id TEXT PRIMARY KEY, telegram_chat_id INTEGER NOT NULL UNIQUE, owner_user_id TEXT, deleted_at DATETIME)`,
		`CREATE TABLE lotteries (id INTEGER PRIMARY KEY, chat_id INTEGER NOT NULL, status TEXT)`,
		`CREATE TABLE level_configs (id TEXT PRIMARY KEY, chat_id INTEGER NOT NULL, deleted_at DATETIME)`,
		`CREATE TABLE keyword_filters (id TEXT PRIMARY KEY, chat_id INTEGER NOT NULL, deleted_at DATETIME)`,
		`CREATE TABLE auto_replies (id TEXT PRIMARY KEY, chat_id INTEGER NOT NULL, deleted_at DATETIME)`,
		`CREATE TABLE scheduled_posts (id INTEGER PRIMARY KEY, chat_id INTEGER NOT NULL)`,
		`CREATE TABLE scheduled_jobs (id TEXT PRIMARY KEY, metadata_json TEXT, deleted_at DATETIME)`,
		`CREATE TABLE message_templates (id TEXT PRIMARY KEY, chat_id INTEGER, deleted_at DATETIME)`,
		`CREATE TABLE invite_links (id TEXT PRIMARY KEY, chat_id INTEGER NOT NULL, invite_link TEXT, deleted_at DATETIME)`,
	}
	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create test table: %v", err)
		}
	}
}

func seedTenantIsolationData(t *testing.T, db *gorm.DB, ownerA uuid.UUID, ownerB uuid.UUID, chatA int64, chatB int64) {
	t.Helper()
	exec := func(query string, args ...any) {
		t.Helper()
		if err := db.Exec(query, args...).Error; err != nil {
			t.Fatalf("seed data: %v", err)
		}
	}
	exec(`INSERT INTO telegram_chats (id, telegram_chat_id, owner_user_id) VALUES (?, ?, ?)`, uuid.New().String(), chatA, ownerA.String())
	exec(`INSERT INTO telegram_chats (id, telegram_chat_id, owner_user_id) VALUES (?, ?, ?)`, uuid.New().String(), chatB, ownerB.String())
	exec(`INSERT INTO lotteries (id, chat_id, status) VALUES (?, ?, ?)`, 42, chatB, "active")
	exec(`INSERT INTO level_configs (id, chat_id) VALUES (?, ?)`, "11111111-1111-4111-8111-111111111111", chatB)
	exec(`INSERT INTO keyword_filters (id, chat_id) VALUES (?, ?)`, "22222222-2222-4222-8222-222222222222", chatB)
	exec(`INSERT INTO auto_replies (id, chat_id) VALUES (?, ?)`, "33333333-3333-4333-8333-333333333333", chatB)
	exec(`INSERT INTO scheduled_posts (id, chat_id) VALUES (?, ?)`, 77, chatB)
	exec(`INSERT INTO scheduled_jobs (id, metadata_json) VALUES (?, ?)`, "44444444-4444-4444-8444-444444444444", `{"telegram_chat_id":-1002002}`)
	exec(`INSERT INTO message_templates (id, chat_id) VALUES (?, ?)`, "55555555-5555-4555-8555-555555555555", chatB)
	exec(`INSERT INTO invite_links (id, chat_id, invite_link) VALUES (?, ?, ?)`, "66666666-6666-4666-8666-666666666666", chatB, "https://t.me/+test")
}

func signOwnerToken(t *testing.T, cfg config.Config, userID string) string {
	t.Helper()
	claims := api.AdminClaims{
		AdminID:  userID,
		UserID:   userID,
		Username: "owner-a",
		Role:     "owner",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.JWT.Issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.JWT.Secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func assertResourceStillPresent(t *testing.T, db *gorm.DB, table string, column string, value any) {
	t.Helper()
	var count int64
	if err := db.Table(table).Where(column+" = ?", value).Count(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != 1 {
		t.Fatalf("%s.%s=%v count = %d, want 1", table, column, value, count)
	}
}
