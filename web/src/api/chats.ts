import { request } from "@/api/http";
import type { ChatRecord } from "@/types/api";

export function fetchChats(): Promise<ChatRecord[]> {
  return request<ChatRecord[]>("/chats");
}

export function bindChat(payload: {
  chat_id: number;
  chat_type: string;
  title: string;
  username?: string;
  bound_by?: string;
  description?: string;
}): Promise<ChatRecord> {
  return request<ChatRecord>("/chats/bind", {
    method: "POST",
    body: payload,
  });
}

export function unbindChat(chatId: number | string): Promise<void> {
  return request<void>(`/chats/${chatId}/bind`, { method: "DELETE" });
}
