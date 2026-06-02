package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"

	"github.com/dabowin/sola/internal/api"
	"github.com/dabowin/sola/internal/bot"
	"github.com/dabowin/sola/internal/model"
	"github.com/dabowin/sola/internal/store"
)

func TestPointsServiceConfigAwardAndRank(t *testing.T) {
	ctx := context.Background()
	st := newServiceTestStore(t)
	createPointTables(t, st.DB)
	svc := NewPointsService(st)

	cfg, err := svc.GetConfig(ctx, 1001)
	if err != nil {
		t.Fatalf("GetConfig returned error: %v", err)
	}
	if cfg.PointText != 1 || !cfg.Enabled || cfg.CooldownSeconds != 60 {
		t.Fatalf("default config = %+v, want text=1 enabled cooldown=60", cfg)
	}

	textPoints := 5
	cooldown := 0
	cfg, err = svc.UpdateConfig(ctx, 1001, bot.ChatPointConfigPatch{
		PointText:       &textPoints,
		CooldownSeconds: &cooldown,
	})
	if err != nil {
		t.Fatalf("UpdateConfig returned error: %v", err)
	}
	if cfg.PointText != 5 || cfg.CooldownSeconds != 0 {
		t.Fatalf("updated config = %+v, want text=5 cooldown=0", cfg)
	}

	result, err := svc.AwardMessage(ctx, bot.PointAwardRequest{ChatID: 1001, UserID: 2001, MessageType: "text"})
	if err != nil {
		t.Fatalf("AwardMessage text returned error: %v", err)
	}
	if !result.Awarded || result.Points != 5 {
		t.Fatalf("text award = %+v, want awarded 5", result)
	}

	result, err = svc.AwardMessage(ctx, bot.PointAwardRequest{ChatID: 1001, UserID: 2002, MessageType: "photo"})
	if err != nil {
		t.Fatalf("AwardMessage photo returned error: %v", err)
	}
	if !result.Awarded || result.Points != 3 {
		t.Fatalf("photo award = %+v, want awarded 3", result)
	}

	result, err = svc.AwardMessage(ctx, bot.PointAwardRequest{ChatID: 1001, UserID: 2003, MessageType: "text", CooldownScope: "sign", ReasonPrefix: "sign"})
	if err != nil {
		t.Fatalf("AwardMessage sign returned error: %v", err)
	}
	if !result.Awarded || result.Points != 5 || result.Reason != "sign:text" {
		t.Fatalf("sign award = %+v, want awarded 5 reason sign:text", result)
	}

	point, err := svc.AdjustUserPoints(ctx, 1001, 2001, 4, "manual bonus")
	if err != nil {
		t.Fatalf("AdjustUserPoints returned error: %v", err)
	}
	if point.TotalPoints != 9 {
		t.Fatalf("user 2001 points = %d, want 9", point.TotalPoints)
	}

	entries, err := svc.GetRankEntries(ctx, 1001, "all", 10)
	if err != nil {
		t.Fatalf("GetRankEntries all returned error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("rank entries len = %d, want 3: %+v", len(entries), entries)
	}
	if entries[0].UserID != 2001 || entries[0].Points != 9 || entries[1].UserID != 2003 || entries[1].Points != 5 || entries[2].UserID != 2002 || entries[2].Points != 3 {
		t.Fatalf("rank entries = %+v, want users sorted by points 2001, 2003, 2002", entries)
	}

	dayRank, err := svc.GetRank(ctx, 1001, "day", 10)
	if err != nil {
		t.Fatalf("GetRank day returned error: %v", err)
	}
	if !strings.Contains(dayRank, "2001 - 9") || !strings.Contains(dayRank, "2003 - 5") || !strings.Contains(dayRank, "2002 - 3") {
		t.Fatalf("day rank = %q, want awarded users", dayRank)
	}
}

func TestPointsServiceListPointLogsCursorPagination(t *testing.T) {
	ctx := context.Background()
	st := newServiceTestStore(t)
	createPointTables(t, st.DB)
	svc := NewPointsService(st)

	base := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	seed := []model.PointLog{
		{UserID: 2001, ChatID: 1001, Delta: 5, Reason: "latest", CreatedAt: base},
		{UserID: 2001, ChatID: 1001, Delta: 4, Reason: "same-ts-a", CreatedAt: base.Add(-time.Minute)},
		{UserID: 2001, ChatID: 1001, Delta: 3, Reason: "same-ts-b", CreatedAt: base.Add(-time.Minute)},
		{UserID: 2001, ChatID: 1001, Delta: 2, Reason: "older", CreatedAt: base.Add(-2 * time.Minute)},
	}
	for _, log := range seed {
		if err := st.DB.WithContext(ctx).Create(&log).Error; err != nil {
			t.Fatalf("seed point log: %v", err)
		}
	}

	firstPage, nextCursor, err := svc.ListPointLogs(ctx, 1001, 2001, 2, 0, "")
	if err != nil {
		t.Fatalf("ListPointLogs first page returned error: %v", err)
	}
	if len(firstPage) != 2 || firstPage[0].Reason != "latest" || firstPage[1].Reason != "same-ts-b" {
		t.Fatalf("first page = %+v", firstPage)
	}
	if nextCursor == "" {
		t.Fatal("expected next cursor for first page")
	}

	secondPage, finalCursor, err := svc.ListPointLogs(ctx, 1001, 2001, 2, 0, nextCursor)
	if err != nil {
		t.Fatalf("ListPointLogs second page returned error: %v", err)
	}
	if len(secondPage) != 2 || secondPage[0].Reason != "same-ts-a" || secondPage[1].Reason != "older" {
		t.Fatalf("second page = %+v", secondPage)
	}
	if finalCursor != "" {
		t.Fatalf("final cursor = %q, want empty", finalCursor)
	}

	legacyPage, _, err := svc.ListPointLogs(ctx, 1001, 2001, 2, 1, "")
	if err != nil {
		t.Fatalf("ListPointLogs offset fallback returned error: %v", err)
	}
	if len(legacyPage) != 2 || legacyPage[0].Reason != "same-ts-b" || legacyPage[1].Reason != "same-ts-a" {
		t.Fatalf("legacy page = %+v", legacyPage)
	}

	if _, _, err := svc.ListPointLogs(ctx, 1001, 2001, 2, 0, "bad-cursor"); err == nil {
		t.Fatal("expected invalid cursor error")
	}

	decoded, err := base64.RawURLEncoding.DecodeString(nextCursor)
	if err != nil {
		t.Fatalf("decode next cursor: %v", err)
	}
	var payload struct {
		CreatedAt time.Time `json:"created_at"`
		ID        uint64    `json:"id"`
	}
	if err := json.Unmarshal(decoded, &payload); err != nil {
		t.Fatalf("unmarshal next cursor: %v", err)
	}
	if payload.ID != firstPage[1].ID {
		t.Fatalf("cursor id = %d, want %d", payload.ID, firstPage[1].ID)
	}
}

func TestPointCooldownKeyScopes(t *testing.T) {
	if got := pointCooldownKey(1001, 2001, ""); got != "cooldown:1001:2001" {
		t.Fatalf("message cooldown key = %q", got)
	}
	if got := pointCooldownKey(1001, 2001, "message"); got != "cooldown:1001:2001" {
		t.Fatalf("explicit message cooldown key = %q", got)
	}
	if got := pointCooldownKey(1001, 2001, "sign"); got != "cooldown:sign:1001:2001" {
		t.Fatalf("sign cooldown key = %q", got)
	}
}

func TestModerationServiceKeywordsAndViolations(t *testing.T) {
	ctx := context.Background()
	st := newServiceTestStore(t)
	createModerationTables(t, st.DB)
	svc := NewModerationService(st)

	message, err := svc.AddKeyword(ctx, 3001, " spam ", 9001)
	if err != nil {
		t.Fatalf("AddKeyword returned error: %v", err)
	}
	if !strings.Contains(message, "spam") {
		t.Fatalf("AddKeyword message = %q, want keyword", message)
	}

	if _, err := svc.AddKeyword(ctx, 3001, "spam", 9002); err != nil {
		t.Fatalf("AddKeyword upsert returned error: %v", err)
	}

	var filters []model.KeywordFilter
	if err := st.DB.WithContext(ctx).Find(&filters).Error; err != nil {
		t.Fatalf("query keyword filters: %v", err)
	}
	if len(filters) != 1 || filters[0].Keyword != "spam" || filters[0].CreatedBy != 9002 {
		t.Fatalf("keyword filters = %+v, want one upserted spam filter", filters)
	}

	list, err := svc.ListKeywords(ctx, 3001)
	if err != nil {
		t.Fatalf("ListKeywords returned error: %v", err)
	}
	if !strings.Contains(list, "spam") || !strings.Contains(list, "contains/delete") {
		t.Fatalf("keyword list = %q, want spam contains/delete", list)
	}

	if err := svc.RecordViolation(ctx, model.ViolationRecord{
		ChatID:        3001,
		UserID:        4001,
		ViolationType: "keyword",
		ActionTaken:   "delete",
		MessageText:   "contains spam",
	}); err != nil {
		t.Fatalf("RecordViolation returned error: %v", err)
	}

	records, err := svc.ListViolations(ctx, 3001, 4001, 10, 0)
	if err != nil {
		t.Fatalf("ListViolations returned error: %v", err)
	}
	if len(records) != 1 || records[0].DetectedBy != "rule" || records[0].ViolationType != "keyword" {
		t.Fatalf("violation records = %+v, want default detected_by rule", records)
	}

	if _, err := svc.DeleteKeyword(ctx, 3001, "spam", 9002); err != nil {
		t.Fatalf("DeleteKeyword returned error: %v", err)
	}
	list, err = svc.ListKeywords(ctx, 3001)
	if err != nil {
		t.Fatalf("ListKeywords after delete returned error: %v", err)
	}
	if strings.Contains(list, "spam") {
		t.Fatalf("keyword list after delete = %q, want spam removed", list)
	}
}

func TestModerationServiceListViolationsCursorPagination(t *testing.T) {
	ctx := context.Background()
	st := newServiceTestStore(t)
	createModerationTables(t, st.DB)
	svc := NewModerationService(st)

	records := []model.ViolationRecord{
		{BaseModel: model.BaseModel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), CreatedAt: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)}, ChatID: 3001, UserID: 4001, ViolationType: "spam", ActionTaken: "mute"},
		{BaseModel: model.BaseModel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)}, ChatID: 3001, UserID: 4001, ViolationType: "flood", ActionTaken: "delete"},
		{BaseModel: model.BaseModel{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, ChatID: 3001, UserID: 4001, ViolationType: "keyword", ActionTaken: "warn"},
	}
	for _, record := range records {
		if err := st.DB.WithContext(ctx).Create(&record).Error; err != nil {
			t.Fatalf("seed violation: %v", err)
		}
	}

	firstPage, err := svc.ListViolationsFiltered(ctx, ViolationListFilter{ChatID: 3001, UserID: 4001, Limit: 2})
	if err != nil {
		t.Fatalf("ListViolationsFiltered first page returned error: %v", err)
	}
	if len(firstPage) != 2 || firstPage[0].ViolationType != "spam" || firstPage[1].ViolationType != "flood" {
		t.Fatalf("first page = %+v, want [spam flood]", firstPage)
	}

	cursor := encodeUUIDCursor(firstPage[1].CreatedAt, firstPage[1].ID)
	secondPage, err := svc.ListViolationsFiltered(ctx, ViolationListFilter{ChatID: 3001, UserID: 4001, Limit: 2, Cursor: cursor})
	if err != nil {
		t.Fatalf("ListViolationsFiltered cursor page returned error: %v", err)
	}
	if len(secondPage) != 1 || secondPage[0].ViolationType != "keyword" {
		t.Fatalf("second page = %+v, want [keyword]", secondPage)
	}

	offsetPage, err := svc.ListViolationsFiltered(ctx, ViolationListFilter{ChatID: 3001, UserID: 4001, Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("ListViolationsFiltered offset page returned error: %v", err)
	}
	if len(offsetPage) != 1 || offsetPage[0].ViolationType != "flood" {
		t.Fatalf("offset page = %+v, want [flood]", offsetPage)
	}
}

func TestTemplateAndInviteListCursorPagination(t *testing.T) {
	ctx := context.Background()
	st := newServiceTestStore(t)
	execSQL(t, st.DB,
		`CREATE TABLE telegram_chats (id text primary key, telegram_chat_id integer not null unique, owner_user_id text, created_at datetime, updated_at datetime, deleted_at datetime)`,
		`CREATE TABLE message_templates (id text primary key, chat_id integer, name text, content text, media_type text, media_url text, parse_mode text, created_by integer, created_at datetime, updated_at datetime, deleted_at datetime)`,
		`CREATE TABLE invite_links (id text primary key, chat_id integer not null, name text, invite_link text, creates_join_request boolean not null default false, join_count integer not null default 0, created_by integer, created_at datetime, updated_at datetime, deleted_at datetime)`,
	)

	ownerID := "11111111-1111-1111-1111-111111111111"
	chatID := int64(-1001001)
	execSQL(t, st.DB,
		fmt.Sprintf("INSERT INTO telegram_chats (id, telegram_chat_id, owner_user_id, created_at, updated_at) VALUES ('chat-a', %d, '%s', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')", chatID, ownerID),
		fmt.Sprintf("INSERT INTO message_templates (id, chat_id, name, content, media_type, media_url, parse_mode, created_by, created_at, updated_at) VALUES ('00000000-0000-0000-0000-000000000003', %d, 't1', '', 'text', '', '', 1, '2026-01-03T00:00:00Z', '2026-01-03T00:00:00Z')", chatID),
		fmt.Sprintf("INSERT INTO message_templates (id, chat_id, name, content, media_type, media_url, parse_mode, created_by, created_at, updated_at) VALUES ('00000000-0000-0000-0000-000000000002', %d, 't2', '', 'text', '', '', 1, '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z')", chatID),
		fmt.Sprintf("INSERT INTO message_templates (id, chat_id, name, content, media_type, media_url, parse_mode, created_by, created_at, updated_at) VALUES ('00000000-0000-0000-0000-000000000001', %d, 't3', '', 'text', '', '', 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')", chatID),
		fmt.Sprintf("INSERT INTO invite_links (id, chat_id, name, invite_link, creates_join_request, join_count, created_by, created_at, updated_at) VALUES ('00000000-0000-0000-0000-000000000103', %d, 'i1', 'https://t.me/+1', 0, 1, 1, '2026-01-03T00:00:00Z', '2026-01-03T00:00:00Z')", chatID),
		fmt.Sprintf("INSERT INTO invite_links (id, chat_id, name, invite_link, creates_join_request, join_count, created_by, created_at, updated_at) VALUES ('00000000-0000-0000-0000-000000000102', %d, 'i2', 'https://t.me/+2', 0, 2, 1, '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z')", chatID),
		fmt.Sprintf("INSERT INTO invite_links (id, chat_id, name, invite_link, creates_join_request, join_count, created_by, created_at, updated_at) VALUES ('00000000-0000-0000-0000-000000000101', %d, 'i3', 'https://t.me/+3', 0, 3, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')", chatID),
	)

	templateSvc := &templateAPIService{templates: NewMessageTemplateService(st)}
	firstTemplates, err := templateSvc.List(ctx, api.TemplateListQuery{OwnerUserID: ownerID, ChatID: chatID, Limit: 2})
	if err != nil {
		t.Fatalf("template List first page returned error: %v", err)
	}
	if len(firstTemplates.Items) != 2 || firstTemplates.Items[0].Name != "t1" || firstTemplates.Items[1].Name != "t2" || firstTemplates.NextCursor == "" {
		t.Fatalf("template first page = %+v", firstTemplates)
	}
	secondTemplates, err := templateSvc.List(ctx, api.TemplateListQuery{OwnerUserID: ownerID, ChatID: chatID, Limit: 2, Cursor: firstTemplates.NextCursor})
	if err != nil {
		t.Fatalf("template List second page returned error: %v", err)
	}
	if len(secondTemplates.Items) != 1 || secondTemplates.Items[0].Name != "t3" {
		t.Fatalf("template second page = %+v, want [t3]", secondTemplates)
	}
	offsetTemplates, err := templateSvc.List(ctx, api.TemplateListQuery{OwnerUserID: ownerID, ChatID: chatID, Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("template List offset page returned error: %v", err)
	}
	if len(offsetTemplates.Items) != 1 || offsetTemplates.Items[0].Name != "t2" {
		t.Fatalf("template offset page = %+v, want [t2]", offsetTemplates)
	}

	inviteSvc := &inviteLinkAPIService{inviteLinks: NewInviteLinkService(st, "")}
	firstLinks, err := inviteSvc.List(ctx, api.InviteLinkListQuery{OwnerUserID: ownerID, ChatID: chatID, Limit: 2})
	if err != nil {
		t.Fatalf("invite List first page returned error: %v", err)
	}
	if len(firstLinks.Items) != 2 || firstLinks.Items[0].Name != "i1" || firstLinks.Items[1].Name != "i2" || firstLinks.NextCursor == "" {
		t.Fatalf("invite first page = %+v", firstLinks)
	}
	secondLinks, err := inviteSvc.List(ctx, api.InviteLinkListQuery{OwnerUserID: ownerID, ChatID: chatID, Limit: 2, Cursor: firstLinks.NextCursor})
	if err != nil {
		t.Fatalf("invite List second page returned error: %v", err)
	}
	if len(secondLinks.Items) != 1 || secondLinks.Items[0].Name != "i3" {
		t.Fatalf("invite second page = %+v, want [i3]", secondLinks)
	}
	offsetLinks, err := inviteSvc.List(ctx, api.InviteLinkListQuery{OwnerUserID: ownerID, ChatID: chatID, Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("invite List offset page returned error: %v", err)
	}
	if len(offsetLinks.Items) != 1 || offsetLinks.Items[0].Name != "i2" {
		t.Fatalf("invite offset page = %+v, want [i2]", offsetLinks)
	}
}

func TestAutoReplyServiceMatchAndCRUD(t *testing.T) {
	ctx := context.Background()
	st := newServiceTestStore(t)
	createAutoReplyTables(t, st.DB)
	svc := NewAutoReplyService(st)

	if _, err := svc.Create(ctx, &model.AutoReply{ChatID: 4101, Keyword: "价格", MatchType: "contains", ReplyText: "请查看置顶价格表。", Enabled: true}); err != nil {
		t.Fatalf("Create contains auto reply returned error: %v", err)
	}
	if _, err := svc.Create(ctx, &model.AutoReply{ChatID: 4101, Keyword: "报名", MatchType: "exact", ReplyText: "报名入口已开放。", Enabled: true}); err != nil {
		t.Fatalf("Create exact auto reply returned error: %v", err)
	}
	if _, err := svc.Create(ctx, &model.AutoReply{ChatID: 4101, Keyword: `^#[0-9]+$`, MatchType: "regex", ReplyText: "编号已收到。", Enabled: true}); err != nil {
		t.Fatalf("Create regex auto reply returned error: %v", err)
	}
	if _, err := svc.Create(ctx, &model.AutoReply{ChatID: 4101, Keyword: "关闭", MatchType: "contains", ReplyText: "不会出现。", Enabled: false}); err != nil {
		t.Fatalf("Create disabled auto reply returned error: %v", err)
	}

	matches, err := svc.MatchAll(ctx, 4101, "请问价格和报名")
	if err != nil {
		t.Fatalf("MatchAll returned error: %v", err)
	}
	if len(matches) != 1 || matches[0].Keyword != "价格" {
		t.Fatalf("contains matches = %+v, want 价格 only", matches)
	}

	matches, err = svc.MatchAll(ctx, 4101, "报名")
	if err != nil {
		t.Fatalf("MatchAll exact returned error: %v", err)
	}
	if len(matches) != 1 || matches[0].Keyword != "报名" {
		t.Fatalf("exact matches = %+v, want 报名", matches)
	}

	matches, err = svc.MatchAll(ctx, 4101, "#123")
	if err != nil {
		t.Fatalf("MatchAll regex returned error: %v", err)
	}
	if len(matches) != 1 || matches[0].ReplyText != "编号已收到。" {
		t.Fatalf("regex matches = %+v, want 编号 reply", matches)
	}

	matches, err = svc.MatchAll(ctx, 4101, "关闭")
	if err != nil {
		t.Fatalf("MatchAll disabled returned error: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("disabled matches = %+v, want none", matches)
	}

	if err := svc.DeleteByKeyword(ctx, 4101, "价格"); err != nil {
		t.Fatalf("DeleteByKeyword returned error: %v", err)
	}
	matches, err = svc.MatchAll(ctx, 4101, "价格")
	if err != nil {
		t.Fatalf("MatchAll after delete returned error: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("matches after delete = %+v, want none", matches)
	}
}

func TestLotteryServiceJoinCancelAndDrawDue(t *testing.T) {
	ctx := context.Background()
	st := newServiceTestStore(t)
	createLotteryTables(t, st.DB)
	svc := NewLotteryService(st)

	future := time.Now().Add(time.Hour)
	lottery, err := svc.Create(ctx, api.LotteryCreateRequest{
		ChatID:          5001,
		Title:           "launch",
		Prize:           "badge",
		MaxParticipants: 2,
		WinnerCount:     1,
		EndAt:           &future,
		CreatedBy:       9001,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if _, err := svc.Join(ctx, 5001, lottery.ID, 6001, "alice"); err != nil {
		t.Fatalf("Join first user returned error: %v", err)
	}
	if _, err := svc.Join(ctx, 5001, lottery.ID, 6001); err == nil {
		t.Fatalf("Join duplicate user returned nil error, want duplicate rejection")
	}
	if _, err := svc.Join(ctx, 5001, lottery.ID, 6002); err != nil {
		t.Fatalf("Join second user returned error: %v", err)
	}
	if _, err := svc.Join(ctx, 5001, lottery.ID, 6003); err == nil {
		t.Fatalf("Join over capacity returned nil error, want capacity rejection")
	}

	entries, err := svc.Entries(ctx, lottery.ID)
	if err != nil {
		t.Fatalf("Entries returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries len = %d, want 2", len(entries))
	}
	if entries[0].Username != "alice" {
		t.Fatalf("entry username = %q, want alice", entries[0].Username)
	}

	if _, err := svc.CancelForChat(ctx, 5001, lottery.ID, 9001); err != nil {
		t.Fatalf("CancelForChat returned error: %v", err)
	}
	if _, err := svc.Join(ctx, 5001, lottery.ID, 6004); err == nil {
		t.Fatalf("Join cancelled lottery returned nil error, want inactive rejection")
	}

	drawLottery, err := svc.Create(ctx, api.LotteryCreateRequest{
		ChatID:      5001,
		Title:       "draw",
		Prize:       "token",
		WinnerCount: 1,
		CreatedBy:   9001,
	})
	if err != nil {
		t.Fatalf("Create draw lottery returned error: %v", err)
	}
	if _, err := svc.Join(ctx, 5001, drawLottery.ID, 7001); err != nil {
		t.Fatalf("Join draw user 7001 returned error: %v", err)
	}
	if _, err := svc.Join(ctx, 5001, drawLottery.ID, 7002); err != nil {
		t.Fatalf("Join draw user 7002 returned error: %v", err)
	}
	past := time.Now().Add(-time.Minute)
	if err := st.DB.WithContext(ctx).Model(&model.Lottery{}).Where("id = ?", drawLottery.ID).Update("end_at", past).Error; err != nil {
		t.Fatalf("mark lottery due: %v", err)
	}

	drawn, err := svc.DrawDue(ctx, time.Now())
	if err != nil {
		t.Fatalf("DrawDue returned error: %v", err)
	}
	if len(drawn) != 1 || drawn[0].ID != drawLottery.ID {
		t.Fatalf("drawn lotteries = %+v, want draw lottery only", drawn)
	}

	var ended model.Lottery
	if err := st.DB.WithContext(ctx).First(&ended, "id = ?", drawLottery.ID).Error; err != nil {
		t.Fatalf("query ended lottery: %v", err)
	}
	if ended.Status != "ended" {
		t.Fatalf("draw lottery status = %q, want ended", ended.Status)
	}

	winners, err := svc.Winners(ctx, drawLottery.ID)
	if err != nil {
		t.Fatalf("Winners returned error: %v", err)
	}
	if len(winners) != 1 || !winners[0].IsWinner {
		t.Fatalf("winners = %+v, want exactly one winner", winners)
	}

	keywordLottery, err := svc.Create(ctx, api.LotteryCreateRequest{
		ChatID:      5001,
		Title:       "keyword",
		WinnerCount: 1,
		EndAt:       &future,
		JoinType:    "keyword",
		JoinKeyword: "888",
	})
	if err != nil {
		t.Fatalf("Create keyword lottery returned error: %v", err)
	}
	matched, message, matchedID, err := svc.JoinByKeyword(ctx, 5001, " 888 ", 8001, "bob")
	if err != nil {
		t.Fatalf("JoinByKeyword returned error: %v", err)
	}
	if !matched || matchedID != keywordLottery.ID || !strings.Contains(message, fmt.Sprintf("#%d", keywordLottery.ID)) {
		t.Fatalf("JoinByKeyword = matched %v id %d message %q, want keyword lottery", matched, matchedID, message)
	}
}

func TestStatsOverviewScopesScheduledJobsByOwner(t *testing.T) {
	ctx := context.Background()
	st := newServiceTestStore(t)
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	ownerA := "11111111-1111-1111-1111-111111111111"
	ownerB := "22222222-2222-2222-2222-222222222222"
	chatA := int64(-1001001)
	chatB := int64(-1002002)

	execSQL(t, st.DB,
		`CREATE TABLE telegram_chats (
			id text PRIMARY KEY,
			telegram_chat_id integer NOT NULL UNIQUE,
			owner_user_id text,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime
		)`,
		`CREATE TABLE scheduled_posts (
			id integer PRIMARY KEY AUTOINCREMENT,
			chat_id integer NOT NULL,
			title text,
			created_at datetime NOT NULL,
			enabled boolean NOT NULL DEFAULT true,
			last_run_at datetime,
			run_once_at datetime
		)`,
		`CREATE TABLE scheduled_jobs (
			id text PRIMARY KEY,
			status text NOT NULL,
			metadata_json text NOT NULL,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime
		)`,
		`CREATE TABLE user_points (
			id integer PRIMARY KEY AUTOINCREMENT,
			user_id integer NOT NULL,
			chat_id integer NOT NULL,
			total_points integer NOT NULL DEFAULT 0,
			updated_at datetime
		)`,
		`CREATE TABLE point_logs (
			id integer PRIMARY KEY AUTOINCREMENT,
			user_id integer NOT NULL,
			chat_id integer NOT NULL,
			delta integer NOT NULL,
			reason text,
			created_at datetime NOT NULL
		)`,
	)

	execSQL(t, st.DB,
		fmt.Sprintf("INSERT INTO telegram_chats (id, telegram_chat_id, owner_user_id, created_at, updated_at) VALUES ('chat-a', %d, '%s', '%s', '%s')", chatA, ownerA, now.Format(time.RFC3339), now.Format(time.RFC3339)),
		fmt.Sprintf("INSERT INTO telegram_chats (id, telegram_chat_id, owner_user_id, created_at, updated_at) VALUES ('chat-b', %d, '%s', '%s', '%s')", chatB, ownerB, now.Format(time.RFC3339), now.Format(time.RFC3339)),
		fmt.Sprintf("INSERT INTO scheduled_posts (id, chat_id, title, created_at) VALUES (1, %d, 'owner-a', '%s')", chatA, now.Format(time.RFC3339)),
		fmt.Sprintf("INSERT INTO scheduled_posts (id, chat_id, title, created_at) VALUES (2, %d, 'owner-b', '%s')", chatB, now.Format(time.RFC3339)),
		fmt.Sprintf("INSERT INTO scheduled_jobs (id, status, metadata_json, created_at, updated_at) VALUES ('00000000-0000-0000-0000-000000000001', 'pending', '{\"telegram_chat_id\":%d}', '%s', '%s')", chatA, now.Format(time.RFC3339), now.Format(time.RFC3339)),
		fmt.Sprintf("INSERT INTO scheduled_jobs (id, status, metadata_json, created_at, updated_at) VALUES ('00000000-0000-0000-0000-000000000002', 'running', '{\"chat_id\":%d}', '%s', '%s')", chatA, now.Format(time.RFC3339), now.Format(time.RFC3339)),
		fmt.Sprintf("INSERT INTO scheduled_jobs (id, status, metadata_json, created_at, updated_at) VALUES ('00000000-0000-0000-0000-000000000003', 'completed', '{\"telegram_chat_id\":%d}', '%s', '%s')", chatA, now.Format(time.RFC3339), now.Format(time.RFC3339)),
		fmt.Sprintf("INSERT INTO scheduled_jobs (id, status, metadata_json, created_at, updated_at) VALUES ('00000000-0000-0000-0000-000000000004', 'pending', '{\"telegram_chat_id\":%d}', '%s', '%s')", chatB, now.Format(time.RFC3339), now.Format(time.RFC3339)),
		fmt.Sprintf("INSERT INTO user_points (user_id, chat_id, total_points, updated_at) VALUES (2001, %d, 15, '%s')", chatA, now.Format(time.RFC3339)),
		fmt.Sprintf("INSERT INTO user_points (user_id, chat_id, total_points, updated_at) VALUES (3001, %d, 30, '%s')", chatB, now.Format(time.RFC3339)),
		fmt.Sprintf("INSERT INTO point_logs (user_id, chat_id, delta, reason, created_at) VALUES (2001, %d, 5, 'owner-a', '%s')", chatA, now.Format(time.RFC3339)),
		fmt.Sprintf("INSERT INTO point_logs (user_id, chat_id, delta, reason, created_at) VALUES (3001, %d, 9, 'owner-b', '%s')", chatB, now.Format(time.RFC3339)),
	)

	svc := &statsAPIService{store: st}
	overview, err := svc.Overview(ctx, api.StatsQuery{
		OwnerUserID: ownerA,
		From:        now,
		To:          now,
	})
	if err != nil {
		t.Fatalf("Overview returned error: %v", err)
	}
	if overview.TotalChats != 1 || overview.TotalPosts != 1 || overview.TotalSchedules != 3 {
		t.Fatalf("overview chat/post/schedule counts = %+v, want chats=1 posts=1 schedules=3", overview)
	}
	if overview.OpenTasks != 2 {
		t.Fatalf("overview.OpenTasks = %d, want 2", overview.OpenTasks)
	}
	if overview.TotalMembers != 1 || overview.ActiveUsers != 1 || overview.PointsIssued != 5 {
		t.Fatalf("overview member/activity counts = %+v, want members=1 active=1 points=5", overview)
	}
}

func newServiceTestStore(t *testing.T) *store.Store {
	t.Helper()
	dbName := fmt.Sprintf("file:sola_service_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: dbName}, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("close sqlite test db: %v", err)
		}
	})
	return store.New(db, nil)
}

func createPointTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	execSQL(t, db,
		`CREATE TABLE chat_point_configs (
			chat_id integer PRIMARY KEY,
			point_text integer NOT NULL DEFAULT 1,
			point_photo integer NOT NULL DEFAULT 3,
			point_sticker integer NOT NULL DEFAULT 2,
			point_video integer NOT NULL DEFAULT 3,
			point_file integer NOT NULL DEFAULT 2,
			point_voice integer NOT NULL DEFAULT 3,
			cooldown_seconds integer NOT NULL DEFAULT 60,
			enabled boolean NOT NULL DEFAULT true,
			updated_at datetime
		)`,
		`CREATE TABLE user_points (
			id integer PRIMARY KEY AUTOINCREMENT,
			user_id integer NOT NULL,
			chat_id integer NOT NULL,
			total_points integer NOT NULL DEFAULT 0,
			updated_at datetime
		)`,
		`CREATE UNIQUE INDEX idx_user_points_user_chat ON user_points(user_id, chat_id)`,
		`CREATE INDEX idx_user_points_chat_id ON user_points(chat_id)`,
		`CREATE TABLE point_logs (
			id integer PRIMARY KEY AUTOINCREMENT,
			user_id integer NOT NULL,
			chat_id integer NOT NULL,
			delta integer NOT NULL,
			reason text,
			created_at datetime NOT NULL
		)`,
		`CREATE INDEX idx_point_logs_user_chat ON point_logs(user_id, chat_id)`,
		`CREATE INDEX idx_point_logs_chat_id ON point_logs(chat_id)`,
		`CREATE INDEX idx_point_logs_created ON point_logs(created_at)`,
	)
}

func createModerationTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	execSQL(t, db,
		`CREATE TABLE keyword_filters (
			id text PRIMARY KEY DEFAULT '00000000-0000-0000-0000-000000000000',
			created_at datetime,
			updated_at datetime,
			deleted_at datetime,
			chat_id integer NOT NULL,
			keyword text NOT NULL,
			match_type text NOT NULL DEFAULT 'contains',
			action text NOT NULL DEFAULT 'delete',
			scope text NOT NULL DEFAULT 'chat',
			reply_text text NOT NULL DEFAULT '',
			enabled boolean NOT NULL DEFAULT true,
			created_by integer
		)`,
		`CREATE UNIQUE INDEX idx_keyword_filters_chat_keyword ON keyword_filters(chat_id, keyword)`,
		`CREATE INDEX idx_keyword_filters_deleted_at ON keyword_filters(deleted_at)`,
		`CREATE TABLE violation_records (
			id text PRIMARY KEY DEFAULT '00000000-0000-0000-0000-000000000000',
			created_at datetime,
			updated_at datetime,
			deleted_at datetime,
			user_id integer NOT NULL,
			chat_id integer NOT NULL,
			violation_type text NOT NULL,
			action_taken text NOT NULL,
			message_text text,
			detected_by text NOT NULL DEFAULT 'rule',
			duration_seconds integer DEFAULT 0,
			cleared boolean NOT NULL DEFAULT false
		)`,
		`CREATE INDEX idx_violation_records_user_chat ON violation_records(user_id, chat_id)`,
		`CREATE INDEX idx_violation_records_chat_id ON violation_records(chat_id)`,
		`CREATE INDEX idx_violation_records_deleted_at ON violation_records(deleted_at)`,
	)
}

func createAutoReplyTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	execSQL(t, db,
		`CREATE TABLE auto_replies (
			id text PRIMARY KEY DEFAULT '00000000-0000-0000-0000-000000000000',
			created_at datetime,
			updated_at datetime,
			deleted_at datetime,
			chat_id integer NOT NULL,
			keyword text NOT NULL,
			match_type text NOT NULL DEFAULT 'contains',
			reply_text text NOT NULL DEFAULT '',
			enabled boolean NOT NULL DEFAULT true,
			created_by integer
		)`,
		`CREATE UNIQUE INDEX idx_auto_replies_chat_keyword ON auto_replies(chat_id, keyword)`,
		`CREATE INDEX idx_auto_replies_chat_id ON auto_replies(chat_id)`,
		`CREATE INDEX idx_auto_replies_enabled ON auto_replies(enabled)`,
		`CREATE INDEX idx_auto_replies_deleted_at ON auto_replies(deleted_at)`,
	)
}

func createLotteryTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	execSQL(t, db,
		`CREATE TABLE lotteries (
			id integer PRIMARY KEY AUTOINCREMENT,
			chat_id integer NOT NULL,
			title text,
			prize text,
			cost_points integer NOT NULL DEFAULT 0,
			max_participants integer NOT NULL DEFAULT 0,
			winner_count integer NOT NULL DEFAULT 1,
			end_at datetime,
			status text NOT NULL DEFAULT 'active',
			join_type text NOT NULL DEFAULT 'button',
			join_keyword text,
			created_by integer,
			created_at datetime
		)`,
		`CREATE INDEX idx_lotteries_chat_id ON lotteries(chat_id)`,
		`CREATE INDEX idx_lotteries_status ON lotteries(status)`,
		`CREATE INDEX idx_lotteries_end_at ON lotteries(end_at)`,
		`CREATE TABLE lottery_entries (
			id integer PRIMARY KEY AUTOINCREMENT,
			lottery_id integer NOT NULL,
			user_id integer NOT NULL,
			username text NOT NULL DEFAULT '',
			joined_at datetime,
			is_winner boolean NOT NULL DEFAULT false
		)`,
		`CREATE UNIQUE INDEX idx_lottery_entry_user ON lottery_entries(lottery_id, user_id)`,
		`CREATE INDEX idx_lottery_entries_lottery_id ON lottery_entries(lottery_id)`,
		`CREATE INDEX idx_lottery_entries_is_winner ON lottery_entries(is_winner)`,
	)
}

func execSQL(t *testing.T, db *gorm.DB, statements ...string) {
	t.Helper()
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("exec schema statement %q: %v", statement, err)
		}
	}
}
func TestPostAPIServiceListSupportsCursorPagination(t *testing.T) {
	ctx := context.Background()
	st := newServiceTestStore(t)
	execSQL(t, st.DB,
		`CREATE TABLE telegram_chats (
			id text PRIMARY KEY,
			telegram_chat_id integer NOT NULL UNIQUE,
			owner_user_id text,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime
		)`,
		`CREATE TABLE scheduled_posts (
			id integer PRIMARY KEY AUTOINCREMENT,
			chat_id integer NOT NULL,
			title text,
			content text,
			media_url text,
			media_type text,
			cron_expr text,
			run_once_at datetime,
			enabled boolean NOT NULL DEFAULT true,
			last_run_at datetime,
			created_at datetime NOT NULL
		)`)

	ownerID := "11111111-1111-1111-1111-111111111111"
	chatID := int64(-1001001)
	execSQL(t, st.DB,
		fmt.Sprintf("INSERT INTO telegram_chats (id, telegram_chat_id, owner_user_id, created_at, updated_at) VALUES ('chat-a', %d, '%s', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')", chatID, ownerID),
		fmt.Sprintf("INSERT INTO scheduled_posts (id, chat_id, title, created_at) VALUES (1, %d, 'first', '2026-01-03T00:00:00Z')", chatID),
		fmt.Sprintf("INSERT INTO scheduled_posts (id, chat_id, title, created_at) VALUES (2, %d, 'second', '2026-01-02T00:00:00Z')", chatID),
		fmt.Sprintf("INSERT INTO scheduled_posts (id, chat_id, title, created_at) VALUES (3, %d, 'third', '2026-01-01T00:00:00Z')", chatID),
	)

	svc := &postAPIService{store: st}
	firstPage, err := svc.List(ctx, api.CommonListQuery{OwnerUserID: ownerID, Limit: 2})
	if err != nil {
		t.Fatalf("List first page returned error: %v", err)
	}
	if len(firstPage) != 2 || firstPage[0].ID != "1" || firstPage[1].ID != "2" {
		t.Fatalf("first page = %+v, want ids [1 2]", firstPage)
	}

	cursor := EncodePostCursor(firstPage[1].CreatedAt, 2)
	secondPage, err := svc.List(ctx, api.CommonListQuery{OwnerUserID: ownerID, Limit: 2, Cursor: cursor})
	if err != nil {
		t.Fatalf("List second page returned error: %v", err)
	}
	if len(secondPage) != 1 || secondPage[0].ID != "3" {
		t.Fatalf("second page = %+v, want id [3]", secondPage)
	}

	offsetPage, err := svc.List(ctx, api.CommonListQuery{OwnerUserID: ownerID, Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("List offset page returned error: %v", err)
	}
	if len(offsetPage) != 1 || offsetPage[0].ID != "2" {
		t.Fatalf("offset page = %+v, want id [2]", offsetPage)
	}
}
