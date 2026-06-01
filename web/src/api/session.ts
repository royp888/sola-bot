import type { UserProfile } from "@/types/api";

const sessionState: { token: string | null; user: UserProfile | null } = {
  token: null,
  user: null,
};

export function getStoredToken(): string | null {
  return sessionState.token;
}

export function getStoredUser(): UserProfile | null {
  return sessionState.user;
}

export function setSession(token: string, user: UserProfile): void {
  sessionState.token = token;
  sessionState.user = user;
}

export function clearSession(): void {
  sessionState.token = null;
  sessionState.user = null;
}

export function hasSession(): boolean {
  return Boolean(sessionState.token);
}
