import type { UserProfile } from "@/types/api";

const STORAGE_KEY = "sola-admin-session";

interface StoredSessionPayload {
  token: string;
  user: UserProfile;
}

const sessionState: { token: string | null; user: UserProfile | null } = {
  token: null,
  user: null,
};

let hydrated = false;

function hydrateSession(): void {
  if (hydrated || typeof window === "undefined") {
    return;
  }

  hydrated = true;
  try {
    const raw = window.sessionStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return;
    }
    const parsed = JSON.parse(raw) as Partial<StoredSessionPayload>;
    sessionState.token = typeof parsed.token === "string" ? parsed.token : null;
    sessionState.user = parsed.user && typeof parsed.user === "object" ? (parsed.user as UserProfile) : null;
  } catch {
    window.sessionStorage.removeItem(STORAGE_KEY);
  }
}

function persistSession(): void {
  if (typeof window === "undefined") {
    return;
  }

  if (!sessionState.token || !sessionState.user) {
    window.sessionStorage.removeItem(STORAGE_KEY);
    return;
  }

  const payload: StoredSessionPayload = {
    token: sessionState.token,
    user: sessionState.user,
  };
  window.sessionStorage.setItem(STORAGE_KEY, JSON.stringify(payload));
}

export function getStoredToken(): string | null {
  hydrateSession();
  return sessionState.token;
}

export function getStoredUser(): UserProfile | null {
  hydrateSession();
  return sessionState.user;
}

export function setSession(token: string, user: UserProfile): void {
  hydrated = true;
  sessionState.token = token;
  sessionState.user = user;
  persistSession();
}

export function clearSession(): void {
  hydrated = true;
  sessionState.token = null;
  sessionState.user = null;
  persistSession();
}

export function hasSession(): boolean {
  return Boolean(getStoredToken());
}