"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { apiRequest, ApiRequestError } from "@/lib/api";
import {
  getAccessToken,
  setAccessToken,
  getStoredUser,
  setStoredUser,
  setSessionCookie,
  clearAuth,
  initAuth,
  setPending2FA,
} from "@/lib/auth";
import { mapApiErrorMessage } from "@/lib/utils";
import type { User, LoginResponse, RegisterResponse, Get2FAStatusResponse } from "@/lib/types";

interface UseAuthReturn {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  logoutAll: () => Promise<void>;
}

export function useAuth(): UseAuthReturn {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const restore = async () => {
      const stored = getStoredUser();
      if (stored && getAccessToken()) {
        setUser(stored);
        setIsLoading(false);
        return;
      }

      const refreshed = await initAuth();
      if (refreshed) {
        setUser(getStoredUser());
      }
      setIsLoading(false);
    };
    restore();
  }, []);

  const login = useCallback(
    async (email: string, password: string) => {
      const data = await apiRequest<LoginResponse>("/api/v1/auth/login", {
        method: "POST",
        body: { email, password },
        auth: false,
      });

      setAccessToken(data.tokens.accessToken);
      await setSessionCookie(data.tokens.refreshToken);
      const u = { id: data.user.id, email: data.user.email };
      setStoredUser(u);
      setUser(u);

      // Check if 2FA is enabled — if so, require verification before dashboard
      try {
        const status = await apiRequest<Get2FAStatusResponse>(
          `/api/v1/2fa/status?userId=${encodeURIComponent(u.id)}`,
        );
        if (status.isEnabled) {
          setPending2FA(true);
          router.replace("/2fa/verify");
          return;
        }
      } catch {
        // If status check fails, proceed to dashboard
      }

      setPending2FA(false);
      router.replace("/dashboard");
    },
    [router],
  );

  const register = useCallback(
    async (email: string, password: string) => {
      const data = await apiRequest<RegisterResponse>("/api/v1/auth/register", {
        method: "POST",
        body: { email, password },
        auth: false,
      });

      setAccessToken(data.tokens.accessToken);
      await setSessionCookie(data.tokens.refreshToken);
      const u = { id: data.user.id, email: data.user.email };
      setStoredUser(u);
      setUser(u);
      router.replace("/dashboard");
    },
    [router],
  );

  const logout = useCallback(async () => {
    try {
      await apiRequest("/api/v1/auth/logout", {
        method: "POST",
        body: { refreshToken: "" },
      });
    } catch {
      // Proceed with local cleanup even if API fails
    }
    await clearAuth();
    setUser(null);
    router.replace("/login");
  }, [router]);

  const logoutAll = useCallback(async () => {
    if (!user) return;
    try {
      await apiRequest("/api/v1/auth/logout-all", {
        method: "POST",
        body: { userId: user.id },
      });
    } catch (e) {
      if (e instanceof ApiRequestError) {
        throw new Error(mapApiErrorMessage(e.code, e.message));
      }
      throw e;
    }
    await clearAuth();
    setUser(null);
    router.replace("/login");
  }, [user, router]);

  return {
    user,
    isAuthenticated: !!user && !!getAccessToken(),
    isLoading,
    login,
    register,
    logout,
    logoutAll,
  };
}
