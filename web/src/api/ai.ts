import { request } from "@/api/http";

export interface AiSettings {
  enabled: boolean;
  provider: string;
  api_key_set: boolean;
  model: string;
  endpoint: string;
  system_prompt: string;
  temperature: number;
}

export interface AiSettingsUpdatePayload {
  enabled?: boolean;
  provider?: string;
  api_key?: string;
  model?: string;
  endpoint?: string;
  system_prompt?: string;
  temperature?: number;
}

export function fetchAiSettings(): Promise<AiSettings> {
  return request<AiSettings>("/system/ai-settings");
}

export function updateAiSettings(payload: AiSettingsUpdatePayload): Promise<AiSettings> {
  return request<AiSettings>("/system/ai-settings", {
    method: "PUT",
    body: payload,
  });
}
