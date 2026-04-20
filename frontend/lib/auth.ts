import type { User } from "@/lib/types";

let accessToken: string | null = null;

// --- Access Token (in-memory only) ---

export function getAccessToken(): string | null {
  return accessToken;
}

export function setAccessToken(token: string | null): void {
  accessToken = token;
}

// --- User (localStorage) ---

export function getStoredUser(): User | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem("user");
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    if (parsed && typeof parsed.id === "string" && typeof parsed.email === "string") {
      return { id: parsed.id, email: parsed.email };
    }
    return null;
  } catch {
    return null;
  }
}

export function setStoredUser(user: User | null): void {
  if (typeof window === "undefined") return;
  if (user) {
    localStorage.setItem("user", JSON.stringify({ id: user.id, email: user.email }));
  } else {
    localStorage.removeItem("user");
  }
}

// --- Session Cookie (refresh token via API route) ---

export async function setSessionCookie(refreshToken: string): Promise<void> {
  await fetch("/api/auth/session", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refreshToken }),
  });
}

export async function getSessionCookie(): Promise<string | null> {
  const res = await fetch("/api/auth/session");
  if (!res.ok) return null;
  const data = await res.json();
  return data.refreshToken ?? null;
}

export async function clearSessionCookie(): Promise<void> {
  await fetch("/api/auth/session", { method: "DELETE" });
}

// --- Pending 2FA (sessionStorage) ---

export function getPending2FA(): boolean {
  if (typeof window === "undefined") return false;
  return sessionStorage.getItem("pending2fa") === "true";
}

export function setPending2FA(pending: boolean): void {
  if (typeof window === "undefined") return;
  if (pending) {
    sessionStorage.setItem("pending2fa", "true");
  } else {
    sessionStorage.removeItem("pending2fa");
  }
}

// --- Clear All Auth State ---

export async function clearAuth(): Promise<void> {
  setAccessToken(null);
  setStoredUser(null);
  setPending2FA(false);
  await clearSessionCookie();
}

// --- Restore access token from refresh on page load ---

export async function initAuth(): Promise<boolean> {
  const refreshToken = await getSessionCookie();
  if (!refreshToken) return false;

  try {
    const res = await fetch("/api/v1/auth/refresh", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refreshToken }),
    });

    if (!res.ok) return false;

    const data = await res.json();
    setAccessToken(data.tokens.accessToken);
    await setSessionCookie(data.tokens.refreshToken);
    return true;
  } catch {
    return false;
  }
}
