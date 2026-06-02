const CHINA_TIME_ZONE = "Asia/Shanghai";

type DateLike = string | number | Date;

function toDate(value?: DateLike | null): Date | undefined {
  if (value == null || value === "") return undefined;
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return undefined;
  return date;
}

function formatterParts(value?: DateLike | null): Record<string, string> | undefined {
  const date = toDate(value);
  if (!date) return undefined;
  const parts = new Intl.DateTimeFormat("zh-CN", {
    timeZone: CHINA_TIME_ZONE,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  }).formatToParts(date);
  const output: Record<string, string> = {};
  parts.forEach((part) => {
    if (part.type !== "literal") {
      output[part.type] = part.value;
    }
  });
  return output;
}

export function formatChinaDateTime(value?: DateLike | null, fallback = "-"): string {
  const parts = formatterParts(value ?? undefined);
  if (!parts) return fallback;
  return `${parts.year}-${parts.month}-${parts.day} ${parts.hour}:${parts.minute}:${parts.second}`;
}

export function formatChinaDate(value?: DateLike | null, fallback = "-"): string {
  const parts = formatterParts(value ?? undefined);
  if (!parts) return fallback;
  return `${parts.year}-${parts.month}-${parts.day}`;
}

export function formatChinaInputDateTime(value?: DateLike | null): string {
  return formatChinaDateTime(value, "");
}

export function parseChinaLocalDateTimeToISO(value?: string | null): string | undefined {
  const text = value?.trim();
  if (!text) return undefined;
  if (/[zZ]|[+-]\d{2}:?\d{2}$/.test(text)) {
    const direct = new Date(text);
    return Number.isNaN(direct.getTime()) ? undefined : direct.toISOString();
  }
  const normalized = text.replace("T", " ");
  const match = normalized.match(/^(\d{4})-(\d{2})-(\d{2})\s+(\d{2}):(\d{2})(?::(\d{2}))?$/);
  if (!match) return undefined;
  const [, year, month, day, hour, minute, second = "00"] = match;
  const utcMillis = Date.UTC(Number(year), Number(month) - 1, Number(day), Number(hour) - 8, Number(minute), Number(second));
  const date = new Date(utcMillis);
  return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
}
