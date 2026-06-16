import type { ChatID } from "@/types/api";
import { formatChinaDateTime } from "./datetime";

export function parseNumericId(value?: ChatID): number | undefined {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}

export function formatDateTime(value?: string | null, fallback = "-"): string {
  return formatChinaDateTime(value, fallback);
}

export function errorMessage(error: unknown): string {
  const payload = (error as { payload?: { error?: string } })?.payload;
  return payload?.error || "接口不可用";
}
