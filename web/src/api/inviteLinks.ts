import { request } from "@/api/http";
import type { ChatID, InviteLinkPayload, InviteLinkRecord } from "@/types/api";

function encodeId(value: ChatID): string {
  return encodeURIComponent(String(value));
}

export function fetchInviteLinks(chatId?: ChatID): Promise<InviteLinkRecord[]> {
  const params = new URLSearchParams();
  if (chatId) params.set("chatID", String(chatId));
  const suffix = params.toString() ? `?${params.toString()}` : "";
  return request<InviteLinkRecord[]>(`/invite-links${suffix}`);
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
