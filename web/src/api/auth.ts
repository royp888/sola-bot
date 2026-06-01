import { request } from "@/api/http";
import type { LoginRequest, LoginResponse, TelegramLoginPayload, UserProfile } from "@/types/api";

export function login(payload: LoginRequest): Promise<LoginResponse> {
  return request<LoginResponse>("/auth/login", {
    method: "POST",
    body: payload,
  });
}

export function telegramLogin(payload: TelegramLoginPayload): Promise<LoginResponse> {
  return request<LoginResponse>("/auth/telegram", {
    method: "POST",
    body: payload,
  });
}

export function fetchCurrentUser(): Promise<UserProfile> {
  return request<UserProfile>("/auth/me");
}
