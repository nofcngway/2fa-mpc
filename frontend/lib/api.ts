import {
  getAccessToken,
  setAccessToken,
  getSessionCookie,
  setSessionCookie,
  clearAuth,
} from "@/lib/auth";
import type { ApiError } from "@/lib/types";

export class ApiRequestError extends Error {
  code: number;
  details?: ApiError["details"];

  constructor(error: ApiError) {
    super(error.message);
    this.name = "ApiRequestError";
    this.code = error.code;
    this.details = error.details;
  }
}

let refreshPromise: Promise<boolean> | null = null;

async function tryRefresh(): Promise<boolean> {
  if (refreshPromise) return refreshPromise;

  refreshPromise = (async () => {
    try {
      const refreshToken = await getSessionCookie();
      if (!refreshToken) return false;

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
    } finally {
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  headers?: Record<string, string>;
  auth?: boolean;
}

export async function apiRequest<T>(
  url: string,
  options: RequestOptions = {},
): Promise<T> {
  const { method = "GET", body, headers = {}, auth = true } = options;

  const buildHeaders = (): Record<string, string> => {
    const h: Record<string, string> = {
      "Content-Type": "application/json",
      ...headers,
    };
    if (auth) {
      const token = getAccessToken();
      if (token) {
        h["Authorization"] = `Bearer ${token}`;
      }
    }
    return h;
  };

  let res = await fetch(url, {
    method,
    headers: buildHeaders(),
    body: body ? JSON.stringify(body) : undefined,
  });

  if (res.status === 401 && auth) {
    const refreshed = await tryRefresh();
    if (refreshed) {
      res = await fetch(url, {
        method,
        headers: buildHeaders(),
        body: body ? JSON.stringify(body) : undefined,
      });
    } else {
      await clearAuth();
      window.location.href = "/login";
      throw new ApiRequestError({ code: 16, message: "Session expired" });
    }
  }

  if (!res.ok) {
    const error: ApiError = await res.json().catch(() => ({
      code: res.status,
      message: res.statusText || "Request failed",
    }));
    throw new ApiRequestError(error);
  }

  const text = await res.text();
  if (!text) return {} as T;
  return JSON.parse(text) as T;
}
