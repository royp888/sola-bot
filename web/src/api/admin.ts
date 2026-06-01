import { request } from "@/api/http";
import type {
  BanPayload,
  BanRecord,
  ChatAdminConfig,
  ChatAdminConfigPayload,
  ChatID,
  WarnRecord,
} from "@/types/api";

function encodeId(value: ChatID): string {
  return encodeURIComponent(String(value));
}

export function fetchAdminConfig(chatId: ChatID): Promise<ChatAdminConfig> {
  return request<ChatAdminConfig>(`/admin/config/${encodeId(chatId)}`);
}

export function updateAdminConfig(
  chatId: ChatID,
  payload: ChatAdminConfigPayload,
): Promise<ChatAdminConfig> {
  return request<ChatAdminConfig>(`/admin/config/${encodeId(chatId)}`, {
    method: "PUT",
    body: payload,
  });
}

export function fetchBans(chatId: ChatID): Promise<BanRecord[]> {
  return request<BanRecord[]>(`/admin/bans/${encodeId(chatId)}`);
}

export function createBan(payload: BanPayload): Promise<BanRecord> {
  return request<BanRecord>("/admin/ban", {
    method: "POST",
    body: payload,
  });
}

export function deleteBan(chatId: ChatID, userId: ChatID): Promise<void> {
  return request<void>(`/admin/ban/${encodeId(chatId)}/${encodeId(userId)}`, {
    method: "DELETE",
  });
}

export function fetchWarns(chatId: ChatID, userId: ChatID): Promise<WarnRecord[]> {
  return request<WarnRecord[]>(`/admin/warns/${encodeId(chatId)}/${encodeId(userId)}`);
}
