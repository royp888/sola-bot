import { request } from "@/api/http";
import type { ChatID, ScheduledPostPayload, ScheduledPostRecord } from "@/types/api";

function encodeId(value: ChatID): string {
  return encodeURIComponent(String(value));
}

export function fetchScheduledPosts(): Promise<ScheduledPostRecord[]> {
  return request<ScheduledPostRecord[]>("/posts");
}

export function createScheduledPost(payload: ScheduledPostPayload): Promise<ScheduledPostRecord> {
  return request<ScheduledPostRecord>("/posts", {
    method: "POST",
    body: payload,
  });
}

export function updateScheduledPost(
  id: ChatID,
  payload: Partial<ScheduledPostPayload>,
): Promise<ScheduledPostRecord> {
  return request<ScheduledPostRecord>(`/posts/${encodeId(id)}`, {
    method: "PUT",
    body: payload,
  });
}

export function deleteScheduledPost(id: ChatID): Promise<void> {
  return request<void>(`/posts/${encodeId(id)}`, {
    method: "DELETE",
  });
}

export function toggleScheduledPost(id: ChatID, enabled: boolean): Promise<ScheduledPostRecord> {
  return request<ScheduledPostRecord>(`/posts/${encodeId(id)}/toggle`, {
    method: "PUT",
    body: { enabled },
  });
}
