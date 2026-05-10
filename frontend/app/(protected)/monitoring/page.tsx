"use client";

import { RefreshCw } from "lucide-react";
import { GlassCard } from "@/components/ui/glass-card";
import { GlassButton } from "@/components/ui/glass-button";
import { LoadingSpinner } from "@/components/ui/loading-spinner";
import { ThroughputOverview } from "@/components/widgets/throughput-overview";
import { ServiceHealthGrid } from "@/components/widgets/service-health-grid";
import { MpcNodeStatus } from "@/components/widgets/mpc-node-status";
import { useMonitoring } from "@/hooks/use-monitoring";
import { useTranslations, useLocale } from "@/lib/i18n";

export default function MonitoringPage() {
  const t = useTranslations();
  const { locale } = useLocale();
  const { snapshot, isLoading, error, lastUpdated, refresh } = useMonitoring();

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-start justify-between gap-4 flex-wrap">
        <div>
          <h1 className="text-2xl font-semibold">{t.monitoring.title}</h1>
          <p className="text-sm text-muted mt-1">{t.monitoring.subtitle}</p>
        </div>
        <div className="flex items-center gap-3">
          {lastUpdated && (
            <span className="text-xs text-muted">
              {t.monitoring.updatedAt} {formatTime(lastUpdated, locale)}
            </span>
          )}
          <GlassButton
            variant="ghost"
            size="sm"
            onPress={() => void refresh()}
            icon={<RefreshCw size={14} />}
            isLoading={isLoading && !snapshot}
            aria-label={t.monitoring.refresh}
          >
            {t.monitoring.refresh}
          </GlassButton>
        </div>
      </header>

      {error && !snapshot && (
        <GlassCard className="p-6">
          <p className="text-sm text-[var(--glass-danger)]">
            {t.monitoring.errorLoading}: {error}
          </p>
        </GlassCard>
      )}

      {isLoading && !snapshot && (
        <GlassCard className="p-6">
          <LoadingSpinner size="sm" label={t.monitoring.loading} />
        </GlassCard>
      )}

      {snapshot && (
        <>
          <ThroughputOverview snapshot={snapshot} />
          <MpcNodeStatus snapshot={snapshot} />
          <ServiceHealthGrid snapshot={snapshot} />
        </>
      )}
    </div>
  );
}

function formatTime(d: Date, locale: string): string {
  return d.toLocaleTimeString(locale === "ru" ? "ru-RU" : "en-US", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}
