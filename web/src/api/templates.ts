import { request } from "@/api/http";
import type {
  ChatID,
  MessageTemplatePayload,
  MessageTemplateRecord,
  MessageTemplateUpdatePayload,
} from "@/types/api";

function encodeId(value: ChatID): string {
  return encodeURIComponent(String(value));
}

export function fetchTemplates(chatId?: ChatID): Promise<MessageTemplateRecord[]> {
  const params = new URLSearchParams();
  if (chatId) params.set("chatID", String(chatId));
  const suffix = params.toString() ? `?${params.toString()}` : "";
  return request<MessageTemplateRecord[]>(`/templates${suffix}`);
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
