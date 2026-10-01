import { request } from "@/api/http";

export type ForwardMode = "copy" | "forward";

export type ForwardHistoryState =
  | "idle"
  | "running"
  | "paused"
  | "completed"
  | "failed";

export interface InlineKeyboardButton {
  text: string;
  url?: string;
  callback_data?: string;
}

export interface InlineKeyboardRow {
  buttons: InlineKeyboardButton[];
}

export interface AutoForward {
  id: number;
  name: string;
  enabled: boolean;

  source_chat_id: number;
  source_title: string;
  target_chat_id: number;
  target_title: string;

  mode: ForwardMode;
  parse_mode: string;
  realtime: boolean;

  history_from_id: number;
  history_to_id: number;
  history_cursor: number;
  history_state: ForwardHistoryState;
  history_sent: number;
  history_failed: number;
  history_skipped: number;
  history_started_at?: string | null;
  history_finished_at?: string | null;
  history_auto_stop_misses: number;

  delay_ms: number;
  batch_size: number;
  album_batch: boolean;

  skip_keywords: string;
  skip_forwarded: boolean;
  skip_no_media: boolean;
  skip_no_text: boolean;
  skip_duplicates: boolean;
  min_file_size_mb: number;
  max_file_size_mb: number;
  min_text_length: number;

  caption_template: string;
  caption_prefix: string;
  caption_suffix: string;
  remove_links: boolean;
  remove_mentions: boolean;
  remove_hashtags: boolean;
  drop_source_line: boolean;
  replacements_json: string;
  keep_original_text: boolean;

  buy_enabled: boolean;
  buy_text: string;
  buy_url_template: string;
  product_id_regex: string;
  buy_button_on_media: boolean;
  buy_button_on_text: boolean;
  extra_buttons_json: string;

  total_forwarded: number;
  total_failed: number;
  total_skipped: number;
  last_forward_at?: string | null;
  last_error: string;

  created_at: string;
  updated_at: string;
}

export interface AutoForwardPayload {
  name?: string;
  enabled?: boolean;
  source_chat_id?: number;
  source_title?: string;
  target_chat_id?: number;
  target_title?: string;
  mode?: ForwardMode;
  parse_mode?: string;
  realtime?: boolean;
  history_from_id?: number;
  history_to_id?: number;
  history_auto_stop_misses?: number;
  start_history?: boolean;
  delay_ms?: number;
  batch_size?: number;
  album_batch?: boolean;
  skip_keywords?: string;
  skip_forwarded?: boolean;
  skip_no_media?: boolean;
  skip_no_text?: boolean;
  skip_duplicates?: boolean;
  min_file_size_mb?: number;
  max_file_size_mb?: number;
  min_text_length?: number;
  caption_template?: string;
  caption_prefix?: string;
  caption_suffix?: string;
  remove_links?: boolean;
  remove_mentions?: boolean;
  remove_hashtags?: boolean;
  drop_source_line?: boolean;
  replacements_json?: string;
  keep_original_text?: boolean;
  buy_enabled?: boolean;
  buy_text?: string;
  buy_url_template?: string;
  product_id_regex?: string;
  buy_button_on_media?: boolean;
  buy_button_on_text?: boolean;
  extra_buttons_json?: string;
}

export interface AutoForwardLog {
  id: number;
  forward_id: number;
  rule_name: string;
  source_chat_id: number;
  source_message_id: number;
  target_chat_id: number;
  target_message_id: number;
  status: "sent" | "failed" | "skipped";
  media_type: string;
  file_size: number;
  text: string;
  error: string;
  created_at: string;
}

export interface AutoForwardPreviewPayload {
  text?: string;
  has_media?: boolean;
  media_type?: string;
  file_size?: number;
  is_forwarded?: boolean;
  message_id?: number;
}

export interface AutoForwardPreview {
  caption: string;
  buttons: InlineKeyboardRow[];
  buy_url: string;
  buy_button_text: string;
  skipped: boolean;
  skip_reason: string;
  warnings: string[] | null;
}

export interface AutoForwardStats {
  total: number;
  enabled: number;
  realtime: number;
  running: number;
  forwarded: number;
  failed: number;
  skipped: number;
}

export function fetchForwards(limit = 50): Promise<AutoForward[]> {
  return request<AutoForward[]>(`/forwards?limit=${limit}`);
}

export function fetchForwardStats(): Promise<AutoForwardStats> {
  return request<AutoForwardStats>("/forwards/stats");
}

export function createForward(payload: AutoForwardPayload): Promise<AutoForward> {
  return request<AutoForward>("/forwards", { method: "POST", body: payload });
}

export function updateForward(
  id: number,
  payload: AutoForwardPayload,
): Promise<AutoForward> {
  return request<AutoForward>(`/forwards/${id}`, { method: "PUT", body: payload });
}

export function deleteForward(id: number): Promise<void> {
  return request<void>(`/forwards/${id}`, { method: "DELETE" });
}

export function startForwardHistory(id: number): Promise<AutoForward> {
  return request<AutoForward>(`/forwards/${id}/history/start`, { method: "POST" });
}

export function pauseForwardHistory(id: number): Promise<AutoForward> {
  return request<AutoForward>(`/forwards/${id}/history/pause`, { method: "POST" });
}

export function resetForwardHistory(id: number): Promise<AutoForward> {
  return request<AutoForward>(`/forwards/${id}/history/reset`, { method: "POST" });
}

export function fetchForwardLogs(id: number, limit = 50): Promise<AutoForwardLog[]> {
  return request<AutoForwardLog[]>(`/forwards/${id}/logs?limit=${limit}`);
}

export function previewForward(
  id: number,
  payload: AutoForwardPreviewPayload,
): Promise<AutoForwardPreview> {
  return request<AutoForwardPreview>(`/forwards/${id}/preview`, {
    method: "POST",
    body: payload,
  });
}

export function testSendForward(
  id: number,
  payload: AutoForwardPreviewPayload,
): Promise<AutoForwardLog> {
  return request<AutoForwardLog>(`/forwards/${id}/test-send`, {
    method: "POST",
    body: payload,
  });
}
