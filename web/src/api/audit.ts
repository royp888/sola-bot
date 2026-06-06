import { request } from "@/api/http";
import type { AuditLogEntry, ChatID } from "@/types/api";

export interface AuditLogListQuery {
  chatId?: ChatID;
  action?: string;
  actorUserId?: ChatID;
  targetId?: ChatID;
  limit?: number;
  cursor?: string;
}

export interface CursorListResponse<T> {
  items: T[];
  next_cursor?: string;
}

function buildQuery(query: AuditLogListQuery = {}): string {
  const params = new URLSearchParams();
  if (query.chatId) params.set("chat_id", String(query.chatId));
  if (query.action) params.set("action", query.action);
  if (query.actorUserId) params.set("actor_user_id", String(query.actorUserId));
  if (query.targetId) params.set("target_id", String(query.targetId));
  if (query.limit) params.set("limit", String(query.limit));
  if (query.cursor) params.set("cursor", query.cursor);
  const suffix = params.toString();
  return suffix ? `?${suffix}` : "";
}

export async function fetchAuditLogs(
  query: AuditLogListQuery = {},
): Promise<CursorListResponse<AuditLogEntry>> {
  const response = await request<AuditLogEntry[] | CursorListResponse<AuditLogEntry>>(`/audit-logs${buildQuery(query)}`);
  return Array.isArray(response) ? { items: response } : response;
}
