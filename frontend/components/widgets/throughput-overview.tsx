"use client";

import { Activity, Gauge, AlertTriangle } from "lucide-react";
import { MetricCard } from "@/components/ui/metric-card";
import { useTranslations } from "@/lib/i18n";
import type { MonitoringSnapshot } from "@/lib/types";

interface ThroughputOverviewProps {
  snapshot: MonitoringSnapshot | null;
}

/**
 * ThroughputOverview — three top-row metric cards summarising the whole mesh:
 * total RPS across all services, weighted-average p95 latency, and aggregate
 * error rate. Drives the "is anything on fire?" gut check at a glance.
 */
export function ThroughputOverview({ snapshot }: ThroughputOverviewProps) {
  const t = useTranslations();
  const stats = aggregate(snapshot);

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
      <MetricCard
        label={t.monitoring.totalRps}
        value={formatNumber(stats.rps, 1)}
        unit={t.monitoring.unitRps}
        icon={<Activity size={18} />}
        tone="accent"
      />
      <MetricCard
        label={t.monitoring.avgLatency}
        value={formatNumber(stats.p95, 0)}
        unit={t.monitoring.unitMs}
        icon={<Gauge size={18} />}
        tone={stats.p95 > 1000 ? "warning" : "neutral"}
      />
      <MetricCard
        label={t.monitoring.errorRate}
        value={formatNumber(stats.errorRatePct, 2)}
        unit={t.monitoring.unitPercent}
        icon={<AlertTriangle size={18} />}
        tone={stats.errorRatePct > 1 ? "danger" : "success"}
      />
    </div>
  );
}

interface AggregatedStats {
  rps: number;
  p95: number;
  errorRatePct: number;
}

function aggregate(snap: MonitoringSnapshot | null): AggregatedStats {
  if (!snap || snap.services.length === 0) {
    return { rps: 0, p95: 0, errorRatePct: 0 };
  }

  let totalRps = 0;
  let weightedP95 = 0;
  let weightedErrors = 0;

  for (const s of snap.services) {
    totalRps += s.rps;
    // Weight latency / error rate by RPS so quiet nodes do not skew the avg.
    weightedP95 += s.latencyP95Ms * s.rps;
    weightedErrors += s.errorRate * s.rps;
  }

  // Fallback: if all services are idle (rps=0), show simple averages so users
  // see something instead of zeros.
  if (totalRps === 0) {
    const n = snap.services.length;
    const avgP95 = snap.services.reduce((sum, s) => sum + s.latencyP95Ms, 0) / n;
    const avgErr = snap.services.reduce((sum, s) => sum + s.errorRate, 0) / n;
    return { rps: 0, p95: avgP95, errorRatePct: avgErr * 100 };
  }

  return {
    rps: totalRps,
    p95: weightedP95 / totalRps,
    errorRatePct: (weightedErrors / totalRps) * 100,
  };
}

function formatNumber(n: number, digits: number): string {
  if (!Number.isFinite(n)) return "—";
  return n.toLocaleString("en-US", {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  });
}
