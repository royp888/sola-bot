import { clearSession, getStoredToken } from "@/api/session";

export class ApiError extends Error {
  status: number;
  payload: unknown;

  constructor(status: number, payload: unknown) {
    super(`API request failed with status ${status}`);
    this.status = status;
    this.payload = payload;
  }
}

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.trim() || "/api";

interface RequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
}

export function buildUrl(path: string): string {
  if (/^https?:\/\//i.test(path)) {
    return path;
  }

  const base = API_BASE_URL.replace(/\/$/, "");
  const suffix = path.replace(/^\//, "");
  return `${base}/${suffix}`;
}

function normalizeBody(body: unknown): BodyInit | undefined {
  if (body == null) {
    return undefined;
  }

  if (
    typeof body === "string" ||
    body instanceof Blob ||
    body instanceof ArrayBuffer ||
    body instanceof FormData ||
    body instanceof URLSearchParams
  ) {
    return body;
  }

  return JSON.stringify(body);
}

function parsePayload(raw: string): unknown {
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw) as unknown;
  } catch {
    return raw;
  }
}

export async function request<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const headers = new Headers(options.headers ?? {});
  headers.set("Accept", "application/json");

  const body = normalizeBody(options.body);
  if (body && !(body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const token = getStoredToken();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(buildUrl(path), {
    ...options,
    body,
    headers,
  });

  const raw = await response.text();
  const payload = parsePayload(raw);

  if (!response.ok) {
    if (response.status === 401) {
      clearSession();
      window.location.href = "/login";
    }
    throw new ApiError(response.status, payload);
  }

  return payload as T;
}
