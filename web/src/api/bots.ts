import { request } from "@/api/http";
import type { BotRecord } from "@/types/api";

export function fetchBots(): Promise<BotRecord[]> {
  return request<BotRecord[]>("/bots");
}
