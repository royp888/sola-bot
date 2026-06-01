import { request } from "@/api/http";
import type { ChatID, LevelPayload, LevelRecord, LevelUpdatePayload } from "@/types/api";

export interface LevelListQuery {
  chatId?: ChatID;
  limit?: number;
  offset?: number;
}

function encodeId(value: ChatID): string {
  return encodeURIComponent(String(value));
}

function buildQuery(query: LevelListQuery = {}): string {
  const params = new URLSearchParams();
  if (query.chatId) params.set("chat_id", String(query.chatId));
  if (query.limit) params.set("limit", String(query.limit));
  if (query.offset) params.set("offset", String(query.offset));
  const suffix = params.toString();
  return suffix ? `?${suffix}` : "";
}

export function fetchLevels(query: LevelListQuery = {}): Promise<LevelRecord[]> {
  return request<LevelRecord[]>(`/levels${buildQuery(query)}`);
}

export function createLevel(payload: LevelPayload): Promise<LevelRecord> {
  return request<LevelRecord>("/levels", {
    method: "POST",
    body: payload,
  });
}

export function updateLevel(id: ChatID, payload: LevelUpdatePayload): Promise<LevelRecord> {
  return request<LevelRecord>(`/levels/${encodeId(id)}`, {
    method: "PATCH",
    body: payload,
  });
}

export function deleteLevel(id: ChatID): Promise<void> {
  return request<void>(`/levels/${encodeId(id)}`, {
    method: "DELETE",
  });
}
