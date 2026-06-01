import { buildUrl, request } from "@/api/http";
import { getStoredToken } from "@/api/session";
import type { BatchUserPayload, BatchUserResult, ChatID, UserRecord } from "@/types/api";

export interface UserListQuery {
  keyword?: string;
  chatId?: ChatID;
  limit?: number;
}

export function fetchUsers(query: UserListQuery = {}): Promise<UserRecord[]> {
  const params = new URLSearchParams();
  params.set("limit", String(query.limit ?? 100));
  if (query.chatId) params.set("chat_id", String(query.chatId));
  if (query.keyword?.trim()) params.set("keyword", query.keyword.trim());
  return request<UserRecord[]>(`/users?${params.toString()}`);
}

export async function exportUsersCsv(query: UserListQuery = {}): Promise<Blob> {
  const params = new URLSearchParams();
  if (query.chatId) params.set("chat_id", String(query.chatId));
  if (query.keyword?.trim()) params.set("keyword", query.keyword.trim());
  const headers = new Headers();
  const token = getStoredToken();
  if (token) headers.set("Authorization", `Bearer ${token}`);
  const response = await fetch(buildUrl(`/users/export?${params.toString()}`), { headers });
  if (!response.ok) {
    throw new Error(`export failed: ${response.status}`);
  }
  return response.blob();
}

export function batchUsers(payload: BatchUserPayload): Promise<BatchUserResult> {
  return request<BatchUserResult>("/users/batch", {
    method: "POST",
    body: payload,
  });
}
