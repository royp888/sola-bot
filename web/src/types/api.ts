export interface UserProfile {
  id: string;
  user_id?: string;
  telegram_user_id?: ChatID;
  name: string;
  display_name?: string;
  username?: string;
  email: string;
  role: "owner" | "admin" | "operator" | "super_admin";
  language: string;
  photo_url?: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  accessToken: string;
  user: UserProfile;
}

export interface TelegramLoginPayload {
  id: number;
  first_name?: string;
  last_name?: string;
  username?: string;
  photo_url?: string;
  auth_date: number;
  hash: string;
  [key: string]: string | number | undefined;
}

export interface OverviewMetric {
  label: string;
  value: string;
  delta: string;
  tone: "primary" | "success" | "warning" | "danger";
}

export interface ActivityItem {
  title: string;
  detail: string;
  time: string;
  status: "success" | "warning" | "danger" | "info";
}

export interface JobItem {
  title: string;
  schedule: string;
  nextRun: string;
  status: "live" | "paused" | "failed";
}

export interface HealthItem {
  label: string;
  value: number;
  note: string;
}

export interface DashboardSummary {
  metrics: OverviewMetric[];
  activity: ActivityItem[];
  jobs: JobItem[];
  health: HealthItem[];
}

export interface BotRecord {
  id: string;
  name: string;
  username: string;
  status: "online" | "degraded" | "offline";
  boundChats: number;
  lastHeartbeat: string;
  language: string;
}

export interface ChatRecord {
  id?: ChatID;
  chat_id?: ChatID;
  title: string;
  kind?: "group" | "channel" | string;
  chat_type?: "group" | "channel" | "supergroup" | "private" | string;
  username?: string;
  invite_link?: string;
  bound_by?: string;
  description?: string;
  bound_at?: string;
  permission?: string;
  members?: number;
  syncStatus?: "synced" | "pending" | "blocked";
  owner?: string;
}

export type ChatID = string | number;

export interface ChatPointConfig {
  chat_id: ChatID;
  enabled: boolean;
  cooldown_seconds: number;
  point_text: number;
  point_photo: number;
  point_sticker: number;
  point_video: number;
  point_file: number;
  point_voice: number;
}

export type ChatPointConfigPayload = Omit<ChatPointConfig, "chat_id">;

export interface StatSeriesPoint {
  label: string;
  value: number;
}

export interface TopPointsUser {
  rank: number;
  user_id: ChatID;
  label: string;
  points: number;
  share: number;
}

export interface StatsOverview {
  metrics: OverviewMetric[];
  series: StatSeriesPoint[];
  topPointsUsers: TopPointsUser[];
  sources: Array<{
    label: string;
    value: number;
    color: string;
  }>;
}

export interface UserRecord {
  id: ChatID;
  username: string;
  display_name: string;
  chat_id: ChatID;
  total_points: number;
  status: "active" | "muted" | "banned";
  last_seen_at: string;
}

export interface BatchUserPayload {
  chat_id: ChatID;
  user_ids: ChatID[];
  action: "ban" | "adjust_points";
  delta?: number;
  reason?: string;
}

export interface BatchUserResult {
  success_count: number;
  failed: string[];
}

export interface PointRankRecord {
  rank: number;
  user_id: ChatID;
  username?: string;
  nickname?: string;
  points?: number;
  total_points?: number;
}

export interface PointLogRecord {
  id: ChatID;
  user_id: ChatID;
  username?: string;
  chat_id: ChatID;
  delta: number;
  reason: string;
  created_at: string;
}

export interface PointLogListResponse {
  items: PointLogRecord[];
  next_cursor?: string;
}

export interface UserPointDetail {
  user_id: ChatID;
  chat_id: ChatID;
  total_points: number;
  updated_at: string;
}

export interface AdjustUserPointsPayload {
  delta: number;
  reason: string;
}

export interface ChatAdminConfig {
  chat_id: ChatID;
  welcome_text: string;
  verify_enabled: boolean;
  verify_timeout: number;
  warn_limit: number;
  updated_at?: string;
}

export type ChatAdminConfigPayload = Omit<ChatAdminConfig, "chat_id" | "updated_at">;

export interface BanRecord {
  id: ChatID;
  user_id: ChatID;
  username?: string;
  chat_id: ChatID;
  reason: string;
  banned_by: ChatID;
  banned_at: string;
  unbanned_at?: string;
}

export interface WarnRecord {
  id: ChatID;
  user_id: ChatID;
  username: string;
  chat_id: ChatID;
  reason: string;
  warned_by: ChatID;
  created_at: string;
  cleared: boolean;
}

export interface BanPayload {
  chat_id: ChatID;
  user_id: ChatID;
  reason: string;
}

export interface InlineMediaPayload {
  name: string;
  mime_type: string;
  data_base64: string;
}

export interface ScheduledPostRecord {
  id: ChatID;
  chat_id: ChatID;
  title: string;
  content: string;
  media_url?: string;
  media_name?: string;
  media_mime?: string;
  has_inline_media?: boolean;
  media_type: "text" | "photo" | "video" | "document";
  cron_expr?: string;
  run_once_at?: string | null;
  enabled: boolean;
  last_run_at?: string | null;
  publish_at?: string | null;
  status?: string;
  language?: string;
  template_key?: string;
  external_ref?: string;
  pin_after_send?: boolean;
  auto_delete_seconds?: number;
  clear_inline_media?: boolean;
  media_data_base64?: string;
  created_at: string;
  updated_at?: string;
}

export type ScheduledPostPayload = Omit<ScheduledPostRecord, "id" | "last_run_at" | "created_at" | "has_inline_media">;

export interface BackupData {
  version: string;
  exported_at: string;
  scope: "business" | "full" | string;
  tables: Record<string, unknown>;
}

export interface BackupImportResponse {
  message: string;
}

export interface MessageTemplateRecord {
  id: ChatID;
  chat_id?: ChatID | null;
  name: string;
  content: string;
  media_type: "text" | "photo" | "video";
  media_url?: string;
  parse_mode?: string;
  created_by?: ChatID;
  created_at: string;
  updated_at: string;
}

export interface MessageTemplatePayload {
  chat_id?: ChatID | null;
  name: string;
  content?: string;
  media_type?: "text" | "photo" | "video";
  media_url?: string;
  parse_mode?: string;
  created_by?: ChatID;
}

export type MessageTemplateUpdatePayload = Partial<MessageTemplatePayload>;

export interface InviteLinkRecord {
  id: ChatID;
  chat_id: ChatID;
  name: string;
  invite_link: string;
  creates_join_request: boolean;
  join_count: number;
  created_by?: ChatID;
  created_at: string;
  updated_at: string;
}

export interface InviteLinkPayload {
  chat_id: ChatID;
  name?: string;
  creates_join_request?: boolean;
  created_by?: ChatID;
}

export interface LotteryRecord {
  id: ChatID;
  chat_id: ChatID;
  title: string;
  prize: string;
  cost_points: number;
  max_participants: number;
  winner_count: number;
  end_at?: string | null;
  status: "active" | "ended" | "cancelled";
  join_type?: "button" | "keyword" | "both";
  join_keyword?: string;
  created_by?: ChatID;
  created_at: string;
  participants?: number;
  entry_count?: number;
  winner_count_done?: number;
}

export interface LotteryPayload {
  chat_id: ChatID;
  title: string;
  prize: string;
  cost_points: number;
  max_participants: number;
  winner_count: number;
  end_at?: string | null;
  created_by?: ChatID;
  join_type?: "button" | "keyword" | "both";
  join_keyword?: string;
}

export interface LotteryEntryRecord {
  id: ChatID;
  lottery_id: ChatID;
  user_id: ChatID;
  username?: string;
  joined_at: string;
  is_winner: boolean;
}

export interface LevelRecord {
  id: ChatID;
  chat_id: ChatID;
  name: string;
  min_points: number;
  badge?: string;
  permissions?: string[];
  created_at: string;
  updated_at: string;
}

export interface LevelPayload {
  chat_id: ChatID;
  name: string;
  min_points: number;
  badge?: string;
  permissions?: string[];
}

export type LevelUpdatePayload = Partial<Omit<LevelPayload, "chat_id">>;

export interface KeywordRecord {
  id: ChatID;
  chat_id: ChatID;
  pattern: string;
  match_type: string;
  action: string;
  scope?: string;
  reply_text?: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface KeywordPayload {
  chat_id: ChatID;
  pattern: string;
  match_type?: string;
  action: string;
  scope?: string;
  reply_text?: string;
  enabled?: boolean;
}

export type KeywordUpdatePayload = Partial<Omit<KeywordPayload, "chat_id">>;

export interface AutoReplyRecord {
  id: ChatID;
  chat_id: ChatID;
  keyword: string;
  match_type: string;
  reply_text: string;
  enabled: boolean;
  created_by?: ChatID;
  created_at: string;
  updated_at: string;
}

export interface AutoReplyPayload {
  chat_id: ChatID;
  keyword: string;
  match_type?: string;
  reply_text: string;
  enabled?: boolean;
  created_by?: ChatID;
}

export type AutoReplyUpdatePayload = Partial<Omit<AutoReplyPayload, "chat_id">>;

export interface AdminViolationRecord {
  id: ChatID;
  chat_id: ChatID;
  user_id: ChatID;
  username?: string;
  type: string;
  reason?: string;
  source?: string;
  status: string;
  count?: number;
  resolution?: string;
  created_at: string;
  resolved_at?: string | null;
}

export interface AdminViolationUpdatePayload {
  status?: string;
  resolution?: string;
}
