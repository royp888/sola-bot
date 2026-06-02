import { request } from "@/api/http";
import type {
  ChatID,
  MessageTemplatePayload,
  MessageTemplateRecord,
  MessageTemplateUpdatePayload,
} from "@/types/api";

interface CursorListResponse<T> {
  items: T[];
  next_cursor?: string;
}

function encodeId(value: ChatID): string {
  return encodeURIComponent(String(value));
}

export async function fetchTemplates(
  chatId?: ChatID,
  cursor?: string,
): Promise<CursorListResponse<MessageTemplateRecord>> {
  const params = new URLSearchParams();
  if (chatId) params.set("chatID", String(chatId));
  if (cursor) params.set("cursor", cursor);
  const suffix = params.toString() ? `?${params.toString()}` : "";
  const response = await request<MessageTemplateRecord[] | CursorListResponse<MessageTemplateRecord>>(`/templates${suffix}`);
  return Array.isArray(response) ? { items: response } : response;
}

export function createTemplate(payload: MessageTemplatePayload): Promise<MessageTemplateRecord> {
  return request<MessageTemplateRecord>("/templates", {
    method: "POST",
    body: payload,
  });
}

export function updateTemplate(
  id: ChatID,
  payload: MessageTemplateUpdatePayload,
): Promise<MessageTemplateRecord> {
  return request<MessageTemplateRecord>(`/templates/${encodeId(id)}`, {
    method: "PUT",
    body: payload,
  });
}

export function deleteTemplate(id: ChatID): Promise<void> {
  return request<void>(`/templates/${encodeId(id)}`, {
    method: "DELETE",
  });
}
