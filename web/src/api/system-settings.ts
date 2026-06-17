import { request } from "@/api/http";

export interface SystemSettings {
  turnstile_site_key: string;
  turnstile_secret_key_set: boolean;
  turnstile_verify_secret_set: boolean;
  admin_username: string;
  admin_password_override: boolean;
  bot_token_masked: string;
}

export interface SystemSettingsUpdatePayload {
  turnstile_site_key?: string;
  turnstile_secret_key?: string;
  turnstile_verify_secret?: string;
  new_admin_password?: string;
  current_admin_password?: string;
}

export function fetchSystemSettings(): Promise<SystemSettings> {
  return request<SystemSettings>("/system/settings");
}

export function updateSystemSettings(payload: SystemSettingsUpdatePayload): Promise<SystemSettings> {
  return request<SystemSettings>("/system/settings", {
    method: "PUT",
    body: payload,
  });
}
