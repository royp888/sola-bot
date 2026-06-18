import { request } from "@/api/http";

export interface BotConfig {
  enabled: boolean;
  default_language: string;
  time_zone: string;
  auto_delete_enabled: boolean;
  auto_delete_after_secs: number;
  allow_forwarded_posts: boolean;
  enable_stats_tracking: boolean;
  enable_points: boolean;
}

export function fetchBotConfig(): Promise<BotConfig> {
  return request<BotConfig>("/bot/config");
}

export function updateBotConfig(payload: Partial<BotConfig>): Promise<BotConfig> {
  return request<BotConfig>("/bot/config", { method: "PUT", body: payload });
}
