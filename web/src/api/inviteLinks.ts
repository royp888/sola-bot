import { request } from "@/api/http";
import type { ChatID, InviteLinkPayload, InviteLinkRecord } from "@/types/api";

interface CursorListResponse<T> {
  items: T[];
  next_cursor?: string;
}

function encodeId(value: ChatID): string {
  return encodeURIComponent(String(value));
}

export async function fetchInviteLinks(
  chatId?: ChatID,
  cursor?: string,
): Promise<CursorListResponse<InviteLinkRecord>> {
  const params = new URLSearchParams();
  if (chatId) params.set("chatID", String(chatId));
  if (cursor) params.set("cursor", cursor);
  const suffix = params.toString() ? `?${params.toString()}` : "";
  const response = await request<InviteLinkRecord[] | CursorListResponse<InviteLinkRecord>>(`/invite-links${suffix}`);
  return Array.isArray(response) ? { items: response } : response;
}

export function createInviteLink(payload: InviteLinkPayload): Promise<InviteLinkRecord> {
  return request<InviteLinkRecord>("/invite-links", {
    method: "POST",
    body: payload,
  });
}

export function deleteInviteLink(id: ChatID): Promise<void> {
  return request<void>(`/invite-links/${encodeId(id)}`, {
    method: "DELETE",
  });
}
