"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { apiRequest, ApiRequestError } from "@/lib/api";
import type { MonitoringSnapshot } from "@/lib/types";

const REFRESH_MS = 10_000;

interface UseMonitoringReturn {
  snapshot: MonitoringSnapshot | null;
  isLoading: boolean;
  error: string | null;
  lastUpdated: Date | null;
  refresh: () => Promise<void>;
}

/**
 * useMonitoring polls the Gateway snapshot endpoint every REFRESH_MS while
 * the page is visible. The first error fills `error` but does not throw —
 * subsequent successful polls clear it. The hook stops polling on unmount
 * and pauses while the document is hidden so we do not waste API calls.
 */
export function useMonitoring(): UseMonitoringReturn {
  const [snapshot, setSnapshot] = useState<MonitoringSnapshot | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await apiRequest<MonitoringSnapshot>(
        "/api/v1/admin/monitoring/snapshot",
      );
      setSnapshot(data);
      setLastUpdated(new Date());
      setError(null);
    } catch (e) {
      const msg = e instanceof ApiRequestError ? e.message : "Failed to load monitoring snapshot";
      setError(msg);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();

    const start = () => {
      if (timerRef.current !== null) return;
      timerRef.current = setInterval(() => void refresh(), REFRESH_MS);
    };
    const stop = () => {
      if (timerRef.current === null) return;
      clearInterval(timerRef.current);
      timerRef.current = null;
    };

    start();

    const onVisibility = () => {
      if (document.hidden) stop();
      else {
        void refresh();
        start();
      }
    };
    document.addEventListener("visibilitychange", onVisibility);

    return () => {
      stop();
      document.removeEventListener("visibilitychange", onVisibility);
    };
  }, [refresh]);

  return { snapshot, isLoading, error, lastUpdated, refresh };
}
