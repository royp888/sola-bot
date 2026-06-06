package api_test

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dabowin/sola/internal/api"
	"github.com/dabowin/sola/internal/config"
	"github.com/dabowin/sola/internal/service"
	"github.com/dabowin/sola/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
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

func TestTenantIsolationBodyQueryEndpointsReturnForbiddenOrScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: "file:tenant-isolation-body-query?mode=memory&cache=shared"}, &gorm.Config{})
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

	postJSON := func(path string, payload string, want int) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(payload))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != want {
			t.Fatalf("%s status = %d, want %d, body = %s", path, rec.Code, want, rec.Body.String())
		}
	}

	postJSON("/api/v1/admin/ban", `{"chat_id":-1002002,"user_id":9001,"reason":"cross tenant"}`, http.StatusForbidden)
	postJSON("/api/v1/users/batch", `{"chat_id":-1002002,"user_ids":[9001],"action":"ban","reason":"cross tenant"}`, http.StatusForbidden)
	postJSON("/api/v1/templates", `{"name":"global-template","content":"hello"}`, http.StatusForbidden)
	assertTableCount(t, db, "ban_logs", 0)

	for _, path := range []string{
		"/api/v1/users/export?chat_id=-1002002",
		"/api/v1/admin/violations?chat_id=-1002002",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s status = %d, want %d, body = %s", path, rec.Code, http.StatusForbidden, rec.Body.String())
		}
	}

	for _, path := range []string{
		"/api/v1/admin/config/-1002002",
		"/api/v1/admin/bans/-1002002",
		"/api/v1/admin/warns/-1002002/9001",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s status = %d, want 403, body = %s", path, rec.Code, rec.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/export", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("export users status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}
	csvRows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(rec.Body.String(), "\ufeff"))).ReadAll()
	if err != nil {
		t.Fatalf("decode export csv: %v", err)
	}
	if len(csvRows) != 2 {
		t.Fatalf("export row count = %d, want header + 1 owner row, body = %s", len(csvRows), rec.Body.String())
	}
	if csvRows[1][3] != strconv.FormatInt(chatA, 10) {
		t.Fatalf("export chat_id = %s, want %d", csvRows[1][3], chatA)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/violations", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list violations status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}
	var violations api.CursorListResponse[api.AdminViolation]
	if err := json.Unmarshal(rec.Body.Bytes(), &violations); err != nil {
		t.Fatalf("decode violations: %v", err)
	}
	if len(violations.Items) != 1 {
		t.Fatalf("violation count = %d, want 1, body = %s", len(violations.Items), rec.Body.String())
	}
	if violations.Items[0].ChatID != chatA {
		t.Fatalf("violation chat_id = %d, want %d", violations.Items[0].ChatID, chatA)
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/admin/violations/77777777-7777-4777-8777-777777777777", bytes.NewBufferString(`{"status":"cleared"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("patch violation status = %d, want 403, body = %s", rec.Code, rec.Body.String())
	}
	var cleared bool
	if err := db.Raw(`SELECT cleared FROM violation_records WHERE id = ?`, "77777777-7777-4777-8777-777777777777").Scan(&cleared).Error; err != nil {
		t.Fatalf("query violation cleared: %v", err)
	}
	if cleared {
		t.Fatal("cross-tenant violation was updated")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/schedules", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list schedules status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}
	var schedules []api.Schedule
	if err := json.Unmarshal(rec.Body.Bytes(), &schedules); err != nil {
		t.Fatalf("decode schedules: %v", err)
	}
	if len(schedules) != 1 {
		t.Fatalf("schedule count = %d, want 1, body = %s", len(schedules), rec.Body.String())
	}
	if schedules[0].ChatID != chatA {
		t.Fatalf("schedule chat_id = %d, want %d", schedules[0].ChatID, chatA)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard summary status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}
	var summary struct {
		Metrics []struct {
			Label string `json:"label"`
			Value string `json:"value"`
		} `json:"metrics"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode dashboard summary: %v", err)
	}
	metricValues := make(map[string]string, len(summary.Metrics))
	for _, metric := range summary.Metrics {
		metricValues[metric.Label] = metric.Value
	}
	if metricValues["绑定频道/群"] != "1" {
		t.Fatalf("dashboard chat metric = %q, want 1", metricValues["绑定频道/群"])
	}
	if metricValues["定时任务"] != "1" {
		t.Fatalf("dashboard schedule metric = %q, want 1", metricValues["定时任务"])
	}
}

func createTenantIsolationTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	statements := []string{
		`CREATE TABLE telegram_chats (id TEXT PRIMARY KEY, telegram_chat_id INTEGER NOT NULL UNIQUE, owner_user_id TEXT, deleted_at DATETIME)`,
		`CREATE TABLE lotteries (id INTEGER PRIMARY KEY, chat_id INTEGER NOT NULL, status TEXT)`,
		`CREATE TABLE level_configs (id TEXT PRIMARY KEY, chat_id INTEGER NOT NULL, min_points INTEGER NOT NULL DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE keyword_filters (id TEXT PRIMARY KEY, chat_id INTEGER NOT NULL, deleted_at DATETIME)`,
		`CREATE TABLE auto_replies (id TEXT PRIMARY KEY, chat_id INTEGER NOT NULL, deleted_at DATETIME)`,
		`CREATE TABLE scheduled_posts (id INTEGER PRIMARY KEY, chat_id INTEGER NOT NULL)`,
		`CREATE TABLE scheduled_jobs (id TEXT PRIMARY KEY, metadata_json TEXT, status TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE message_templates (id TEXT PRIMARY KEY, chat_id INTEGER, deleted_at DATETIME)`,
		`CREATE TABLE invite_links (id TEXT PRIMARY KEY, chat_id INTEGER NOT NULL, invite_link TEXT, deleted_at DATETIME)`,
		`CREATE TABLE ban_logs (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, chat_id INTEGER NOT NULL, reason TEXT, banned_by INTEGER, banned_at DATETIME, unbanned_at DATETIME)`,
		`CREATE TABLE warn_records (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, chat_id INTEGER NOT NULL, reason TEXT, warned_by INTEGER, created_at DATETIME, cleared BOOLEAN NOT NULL DEFAULT FALSE)`,
		`CREATE TABLE user_points (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, chat_id INTEGER NOT NULL, total_points INTEGER NOT NULL DEFAULT 0, updated_at DATETIME)`,
		`CREATE TABLE point_logs (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, chat_id INTEGER NOT NULL, delta INTEGER NOT NULL, reason TEXT, created_at DATETIME)`,
		`CREATE TABLE users (id TEXT PRIMARY KEY, telegram_user_id INTEGER, username TEXT, display_name TEXT, status TEXT, created_at DATETIME, updated_at DATETIME, last_login_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE violation_records (id TEXT PRIMARY KEY, user_id INTEGER NOT NULL, chat_id INTEGER NOT NULL, violation_type TEXT NOT NULL, action_taken TEXT NOT NULL, message_text TEXT, detected_by TEXT, duration_seconds INTEGER, cleared BOOLEAN NOT NULL DEFAULT FALSE, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
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
	exec(`INSERT INTO level_configs (id, chat_id, min_points) VALUES (?, ?, ?)`, "11111111-1111-4111-8111-111111111111", chatB, 0)
	exec(`INSERT INTO keyword_filters (id, chat_id) VALUES (?, ?)`, "22222222-2222-4222-8222-222222222222", chatB)
	exec(`INSERT INTO auto_replies (id, chat_id) VALUES (?, ?)`, "33333333-3333-4333-8333-333333333333", chatB)
	exec(`INSERT INTO scheduled_posts (id, chat_id) VALUES (?, ?)`, 77, chatB)
	exec(`INSERT INTO scheduled_jobs (id, metadata_json, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`, "44444444-4444-4444-8444-444444444444", `{"telegram_chat_id":-1002002}`, "pending", time.Now(), time.Now())
	exec(`INSERT INTO scheduled_jobs (id, metadata_json, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`, "44444444-4444-4444-8444-444444444445", `{"telegram_chat_id":-1001001}`, "pending", time.Now().Add(time.Second), time.Now())
	exec(`INSERT INTO message_templates (id, chat_id) VALUES (?, ?)`, "55555555-5555-4555-8555-555555555555", chatB)
	exec(`INSERT INTO invite_links (id, chat_id, invite_link) VALUES (?, ?, ?)`, "66666666-6666-4666-8666-666666666666", chatB, "https://t.me/+test")
	exec(`INSERT INTO users (id, telegram_user_id, username, display_name, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, "11111111-1111-4111-8111-111111111112", 8001, "owner_a", "Owner A", "active", time.Now(), time.Now())
	exec(`INSERT INTO user_points (user_id, chat_id, total_points, updated_at) VALUES (?, ?, ?, ?)`, 8001, chatA, 10, time.Now())
	exec(`INSERT INTO user_points (user_id, chat_id, total_points, updated_at) VALUES (?, ?, ?, ?)`, 8002, chatB, 20, time.Now())
	exec(`INSERT INTO violation_records (id, user_id, chat_id, violation_type, action_taken, detected_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, "77777777-7777-4777-8777-777777777776", 8001, chatA, "spam", "warn", "rule", time.Now(), time.Now())
	exec(`INSERT INTO violation_records (id, user_id, chat_id, violation_type, action_taken, detected_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, "77777777-7777-4777-8777-777777777777", 8002, chatB, "spam", "warn", "rule", time.Now(), time.Now())
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

func assertTableCount(t *testing.T, db *gorm.DB, table string, want int64) {
	t.Helper()
	var count int64
	if err := db.Table(table).Count(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != want {
		t.Fatalf("%s count = %d, want %d", table, count, want)
	}
}
