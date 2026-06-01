import { buildUrl, request } from "@/api/http";
import { getStoredToken } from "@/api/session";
import type { BackupData, BackupImportResponse } from "@/types/api";

export async function exportBackup(scope: "business" | "full"): Promise<Blob> {
  const headers = new Headers();
  const token = getStoredToken();
  if (token) headers.set("Authorization", `Bearer ${token}`);
  const response = await fetch(buildUrl(`/backup/export?scope=${encodeURIComponent(scope)}`), { headers });
  if (!response.ok) {
    throw new Error(`export failed: ${response.status}`);
  }
  return response.blob();
}

export function importBackupFile(file: File, mode: "merge" | "overwrite"): Promise<BackupImportResponse> {
  const form = new FormData();
  form.append("file", file);
  return request<BackupImportResponse>(`/backup/import?mode=${encodeURIComponent(mode)}`, {
    method: "POST",
    body: form,
  });
}

export function importBackupData(data: BackupData, mode: "merge" | "overwrite"): Promise<BackupImportResponse> {
  return request<BackupImportResponse>(`/backup/import?mode=${encodeURIComponent(mode)}`, {
    method: "POST",
    body: data,
  });
}
