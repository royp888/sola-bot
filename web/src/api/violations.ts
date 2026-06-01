import { request } from "@/api/http";
import type { AdminViolationRecord, AdminViolationUpdatePayload, ChatID } from "@/types/api";

export interface AdminViolationListQuery {
  chatId?: ChatID;
  userId?: ChatID;
  type?: string;
  status?: string;
  limit?: number;
  offset?: number;
}

function encodeId(value: ChatID): string {
  return encodeURIComponent(String(value));
}

function buildQuery(query: AdminViolationListQuery = {}): string {
  const params = new URLSearchParams();
  if (query.chatId) params.set("chat_id", String(query.chatId));
  if (query.userId) params.set("user_id", String(query.userId));
  if (query.type) params.set("type", query.type);
  if (query.status) params.set("status", query.status);
  if (query.limit) params.set("limit", String(query.limit));
  if (query.offset) params.set("offset", String(query.offset));
  const suffix = params.toString();
  return suffix ? `?${suffix}` : "";
}

export function fetchAdminViolations(
  query: AdminViolationListQuery = {},
): Promise<AdminViolationRecord[]> {
  return request<AdminViolationRecord[]>(`/admin/violations${buildQuery(query)}`);
}

export function updateAdminViolation(
  id: ChatID,
  payload: AdminViolationUpdatePayload,
): Promise<AdminViolationRecord> {
  return request<AdminViolationRecord>(`/admin/violations/${encodeId(id)}`, {
    method: "PATCH",
    body: payload,
  });
}
