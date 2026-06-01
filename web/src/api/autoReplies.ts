import { request } from "@/api/http";
import type {
  AutoReplyPayload,
  AutoReplyRecord,
  AutoReplyUpdatePayload,
  ChatID,
} from "@/types/api";

export interface AutoReplyListQuery {
  chatId?: ChatID;
  enabled?: boolean;
  limit?: number;
  offset?: number;
}

function encodeId(value: ChatID): string {
  return encodeURIComponent(String(value));
}

function buildQuery(query: AutoReplyListQuery = {}): string {
  const params = new URLSearchParams();
  if (query.chatId) params.set("chat_id", String(query.chatId));
  if (typeof query.enabled === "boolean") params.set("enabled", String(query.enabled));
  if (query.limit) params.set("limit", String(query.limit));
  if (query.offset) params.set("offset", String(query.offset));
  const suffix = params.toString();
  return suffix ? `?${suffix}` : "";
}

export function fetchAutoReplies(query: AutoReplyListQuery = {}): Promise<AutoReplyRecord[]> {
  return request<AutoReplyRecord[]>(`/auto-replies${buildQuery(query)}`);
}

export function createAutoReply(payload: AutoReplyPayload): Promise<AutoReplyRecord> {
  return request<AutoReplyRecord>("/auto-replies", {
    method: "POST",
    body: payload,
  });
}

export function updateAutoReply(
  id: ChatID,
  payload: AutoReplyUpdatePayload,
): Promise<AutoReplyRecord> {
  return request<AutoReplyRecord>(`/auto-replies/${encodeId(id)}`, {
    method: "PATCH",
    body: payload,
  });
}

export function deleteAutoReply(id: ChatID): Promise<void> {
  return request<void>(`/auto-replies/${encodeId(id)}`, {
    method: "DELETE",
  });
}
