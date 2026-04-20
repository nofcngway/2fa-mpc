"use client";

import { useState, useCallback } from "react";
import { apiRequest, ApiRequestError } from "@/lib/api";
// Error mapping moved to components that have access to translations
import type {
  Setup2FAResponse,
  Verify2FAResponse,
  Get2FAStatusResponse,
} from "@/lib/types";

interface Use2FAReturn {
  status: Get2FAStatusResponse | null;
  isLoading: boolean;
  fetchStatus: (userId: string) => Promise<void>;
  setup: (userId: string, email: string) => Promise<Setup2FAResponse>;
  verify: (userId: string, otpCode: string) => Promise<Verify2FAResponse>;
  disable: (userId: string, otpCode: string) => Promise<void>;
}

export function use2FA(): Use2FAReturn {
  const [status, setStatus] = useState<Get2FAStatusResponse | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const fetchStatus = useCallback(async (userId: string) => {
    setIsLoading(true);
    try {
      const data = await apiRequest<Get2FAStatusResponse>(
        `/api/v1/2fa/status?userId=${encodeURIComponent(userId)}`,
      );
      setStatus(data);
    } catch (e) {
      if (e instanceof ApiRequestError) {
        throw new Error(e.message);
      }
      throw e;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const setup = useCallback(async (userId: string, email: string) => {
    const data = await apiRequest<Setup2FAResponse>("/api/v1/2fa/setup", {
      method: "POST",
      body: { userId, email },
    });
    return data;
  }, []);

  const verify = useCallback(async (userId: string, otpCode: string) => {
    const data = await apiRequest<Verify2FAResponse>("/api/v1/2fa/verify", {
      method: "POST",
      body: { userId, otpCode },
    });
    return data;
  }, []);

  const disable = useCallback(
    async (userId: string, otpCode: string) => {
      await apiRequest("/api/v1/2fa/disable", {
        method: "POST",
        body: { userId, otpCode },
      });
      setStatus({ isEnabled: false, createdAt: "" });
    },
    [],
  );

  return {
    status,
    isLoading,
    fetchStatus,
    setup,
    verify,
    disable,
  };
}
