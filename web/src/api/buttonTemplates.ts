import { request } from "@/api/http";
import type {
  ButtonTemplatePayload,
  ButtonTemplateRecord,
  ButtonTemplateUpdatePayload,
} from "@/types/api";

interface CursorListResponse<T> {
  items: T[];
  next_cursor?: string;
}

export async function fetchButtonTemplates(
  scope?: string,
  keyword?: string,
  cursor?: string,
): Promise<CursorListResponse<ButtonTemplateRecord>> {
  const params = new URLSearchParams();
  if (scope) params.set("scope", scope);
  if (keyword) params.set("keyword", keyword);
  if (cursor) params.set("cursor", cursor);
  const suffix = params.toString() ? `?${params.toString()}` : "";
  const response = await request<ButtonTemplateRecord[] | CursorListResponse<ButtonTemplateRecord>>(
    `/button-templates${suffix}`,
  );
  return Array.isArray(response) ? { items: response } : response;
}

export function createButtonTemplate(payload: ButtonTemplatePayload): Promise<ButtonTemplateRecord> {
  return request<ButtonTemplateRecord>("/button-templates", {
    method: "POST",
    body: payload,
  });
}

export function updateButtonTemplate(
  id: string,
  payload: ButtonTemplateUpdatePayload,
): Promise<ButtonTemplateRecord> {
  return request<ButtonTemplateRecord>(`/button-templates/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: payload,
  });
}

export function deleteButtonTemplate(id: string): Promise<void> {
  return request<void>(`/button-templates/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}
