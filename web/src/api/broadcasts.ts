import { request } from "@/api/http";

export type BroadcastTargetType = "users" | "groups" | "both";

export type BroadcastStatus =
  | "draft"
  | "queued"
  | "sending"
  | "completed"
  | "failed"
  | "canceled";

export interface InlineKeyboardButton {
  text: string;
  url?: string;
  callback_data?: string;
}

export interface InlineKeyboardRow {
  buttons: InlineKeyboardButton[];
}

export interface Broadcast {
  id: number;
  title: string;
  content: string;
  media_url: string;
  media_name: string;
  media_type: string;
  has_media: boolean;
  parse_mode: string;
  inline_keyboard_json: string;
  target_type: BroadcastTargetType;
  chat_ids: number[];
  status: BroadcastStatus;
  scheduled_at?: string | null;
  started_at?: string | null;
  finished_at?: string | null;
  total_targets: number;
  sent_count: number;
  failed_count: number;
  last_error: string;
  created_at: string;
  updated_at: string;
}

export interface BroadcastCreatePayload {
  title?: string;
  content?: string;
  media_url?: string;
  media_name?: string;
  media_mime?: string;
  media_data_base64?: string;
  media_type?: string;
  parse_mode?: string;
  inline_keyboard?: InlineKeyboardRow[];
  target_type: BroadcastTargetType;
  chat_ids?: number[];
  scheduled_at?: string | null;
  send_now?: boolean;
}

export interface BroadcastUpdatePayload {
  title?: string;
  content?: string;
  media_url?: string;
  media_name?: string;
  media_mime?: string;
  media_data_base64?: string;
  media_type?: string;
  parse_mode?: string;
  inline_keyboard?: InlineKeyboardRow[];
  target_type?: BroadcastTargetType;
  chat_ids?: number[];
  scheduled_at?: string | null;
}

export interface BroadcastTargetPreview {
  users: number;
  chats: number;
  total: number;
}

export interface BroadcastStats {
  total: number;
  draft: number;
  queued: number;
  sending: number;
  completed: number;
  failed: number;
}

export interface BroadcastDelivery {
  id: number;
  target_type: string;
  target_id: number;
  target_name: string;
  status: string;
  message_id: number;
  error: string;
  sent_at?: string | null;
  created_at: string;
}

export function fetchBroadcasts(limit = 50): Promise<Broadcast[]> {
  return request<Broadcast[]>(`/broadcasts?limit=${limit}`);
}

export function fetchBroadcastStats(): Promise<BroadcastStats> {
  return request<BroadcastStats>("/broadcasts/stats");
}

export function createBroadcast(payload: BroadcastCreatePayload): Promise<Broadcast> {
  return request<Broadcast>("/broadcasts", { method: "POST", body: payload });
}

export function updateBroadcast(
  id: number,
  payload: BroadcastUpdatePayload,
): Promise<Broadcast> {
  return request<Broadcast>(`/broadcasts/${id}`, { method: "PUT", body: payload });
}

export function deleteBroadcast(id: number): Promise<void> {
  return request<void>(`/broadcasts/${id}`, { method: "DELETE" });
}

export function sendBroadcast(id: number): Promise<Broadcast> {
  return request<Broadcast>(`/broadcasts/${id}/send`, { method: "POST" });
}

export function cancelBroadcast(id: number): Promise<Broadcast> {
  return request<Broadcast>(`/broadcasts/${id}/cancel`, { method: "POST" });
}

export function previewBroadcastTargets(
  payload: BroadcastCreatePayload,
): Promise<BroadcastTargetPreview> {
  return request<BroadcastTargetPreview>("/broadcasts/preview", {
    method: "POST",
    body: payload,
  });
}

export function fetchBroadcastDeliveries(
  id: number,
  limit = 100,
): Promise<BroadcastDelivery[]> {
  return request<BroadcastDelivery[]>(`/broadcasts/${id}/deliveries?limit=${limit}`);
}
