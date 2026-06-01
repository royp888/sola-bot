import { request } from "@/api/http";
import type {
  AdjustUserPointsPayload,
  ChatID,
  ChatPointConfig,
  ChatPointConfigPayload,
  PointLogRecord,
  PointRankRecord,
  UserPointDetail,
} from "@/types/api";

function encodeId(value: ChatID): string {
  return encodeURIComponent(String(value));
}

export function fetchChatPointConfig(chatId: ChatID): Promise<ChatPointConfig> {
  return request<ChatPointConfig>(`/points/config/${encodeId(chatId)}`);
}

export function updateChatPointConfig(
  chatId: ChatID,
  payload: ChatPointConfigPayload,
): Promise<ChatPointConfig> {
  return request<ChatPointConfig>(`/points/config/${encodeId(chatId)}`, {
    method: "PUT",
    body: payload,
  });
}

export function fetchPointRank(chatId: ChatID, period = "all"): Promise<PointRankRecord[]> {
  return request<PointRankRecord[]>(
    `/points/rank/${encodeId(chatId)}?period=${encodeURIComponent(period)}`,
  );
}

export function fetchUserPointDetail(
  chatId: ChatID,
  userId: ChatID,
): Promise<UserPointDetail> {
  return request<UserPointDetail>(`/points/user/${encodeId(chatId)}/${encodeId(userId)}`);
}

export function updateUserPoints(
  chatId: ChatID,
  userId: ChatID,
  payload: AdjustUserPointsPayload,
): Promise<UserPointDetail> {
  return request<UserPointDetail>(`/points/user/${encodeId(chatId)}/${encodeId(userId)}`, {
    method: "PUT",
    body: payload,
  });
}

export interface PointLogQuery {
  limit?: number;
  offset?: number;
}

export function fetchPointLogs(
  chatId: ChatID,
  userId: ChatID,
  query: PointLogQuery = {},
): Promise<PointLogRecord[]> {
  const params = new URLSearchParams();
  if (query.limit) params.set("limit", String(query.limit));
  if (query.offset) params.set("offset", String(query.offset));
  const suffix = params.toString();
  return request<PointLogRecord[]>(
    `/points/logs/${encodeId(chatId)}/${encodeId(userId)}${suffix ? `?${suffix}` : ""}`,
  );
}
