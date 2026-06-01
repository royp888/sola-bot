import { ApiError, request } from "@/api/http";
import type { ChatID, LotteryEntryRecord, LotteryPayload, LotteryRecord } from "@/types/api";

function encodeId(value: ChatID): string {
  return encodeURIComponent(String(value));
}

export function fetchLotteries(): Promise<LotteryRecord[]> {
  return requestWithV1Fallback<LotteryRecord[]>("/lottery");
}

export function createLottery(payload: LotteryPayload): Promise<LotteryRecord> {
  return requestWithV1Fallback<LotteryRecord>("/lottery", {
    method: "POST",
    body: payload,
  });
}

export function cancelLottery(id: ChatID): Promise<void> {
  return requestWithV1Fallback<void>(`/lottery/${encodeId(id)}`, {
    method: "DELETE",
  });
}

export function fetchLotteryEntries(id: ChatID): Promise<LotteryEntryRecord[]> {
  return requestWithV1Fallback<LotteryEntryRecord[]>(`/lottery/${encodeId(id)}/entries`);
}

export function fetchLotteryWinners(id: ChatID): Promise<LotteryEntryRecord[]> {
  return requestWithV1Fallback<LotteryEntryRecord[]>(`/lottery/${encodeId(id)}/winners`);
}

async function requestWithV1Fallback<T>(path: string, options = {}): Promise<T> {
  try {
    return await request<T>(path, options);
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      return request<T>(`/v1${path}`, options);
    }
    throw error;
  }
}
