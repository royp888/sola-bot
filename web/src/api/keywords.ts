import { request } from "@/api/http";
import type {
  ChatID,
  KeywordPayload,
  KeywordRecord,
  KeywordUpdatePayload,
} from "@/types/api";

export interface KeywordListQuery {
  chatId?: ChatID;
  scope?: string;
  action?: string;
  enabled?: boolean;
  limit?: number;
  offset?: number;
}

function encodeId(value: ChatID): string {
  return encodeURIComponent(String(value));
}

function buildQuery(query: KeywordListQuery = {}): string {
  const params = new URLSearchParams();
  if (query.chatId) params.set("chat_id", String(query.chatId));
  if (query.scope) params.set("scope", query.scope);
  if (query.action) params.set("action", query.action);
  if (typeof query.enabled === "boolean") params.set("enabled", String(query.enabled));
  if (query.limit) params.set("limit", String(query.limit));
  if (query.offset) params.set("offset", String(query.offset));
  const suffix = params.toString();
  return suffix ? `?${suffix}` : "";
}

export function fetchKeywords(query: KeywordListQuery = {}): Promise<KeywordRecord[]> {
  return request<KeywordRecord[]>(`/keywords${buildQuery(query)}`);
}

export function createKeyword(payload: KeywordPayload): Promise<KeywordRecord> {
  return request<KeywordRecord>("/keywords", {
    method: "POST",
    body: payload,
  });
}

export function updateKeyword(
  id: ChatID,
  payload: KeywordUpdatePayload,
): Promise<KeywordRecord> {
  return request<KeywordRecord>(`/keywords/${encodeId(id)}`, {
    method: "PATCH",
    body: payload,
  });
}

export function deleteKeyword(id: ChatID): Promise<void> {
  return request<void>(`/keywords/${encodeId(id)}`, {
    method: "DELETE",
  });
}
