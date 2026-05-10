"use client";

import { Server } from "lucide-react";
import { GlassCard } from "@/components/ui/glass-card";
import { ServiceStatusDot } from "@/components/ui/service-status-dot";
import { useTranslations } from "@/lib/i18n";
import type { MonitoringSnapshot } from "@/lib/types";

interface MpcNodeStatusProps {
  snapshot: MonitoringSnapshot | null;
}

/**
 * MpcNodeStatus — detailed view of the 3 MPC nodes that hold Shamir shares.
 * Renders a "2-of-3 healthy" headline so an operator can confirm the
 * threshold model is satisfied at a glance, plus a card per node.
 */
export function MpcNodeStatus({ snapshot }: MpcNodeStatusProps) {
  const t = useTranslations();

  const nodes = (snapshot?.services ?? []).filter((s) => s.name.startsWith("mpc-node-"));
  const upCount = nodes.filter((n) => n.up).length;
  const headline = thresholdHeadline(upCount, t);

  return (
    <GlassCard className="p-6">
      <div className="flex items-start justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold">{t.monitoring.mpcTitle}</h2>
          <p className="text-sm text-muted">{t.monitoring.mpcSubtitle}</p>
        </div>
        <div className={`text-sm font-medium ${headline.tone}`}>{headline.text}</div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        {nodes.map((n) => (
          <div
            key={n.name}
            className="glass-card-flat p-4 flex items-center gap-3"
          >
            <div
              className="w-10 h-10 rounded-xl flex items-center justify-center shrink-0"
              style={{
                background: n.up
                  ? "color-mix(in oklch, var(--glass-success) 12%, transparent)"
                  : "color-mix(in oklch, var(--glass-danger) 12%, transparent)",
                color: n.up ? "var(--glass-success)" : "var(--glass-danger)",
              }}
            >
              <Server size={18} />
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <ServiceStatusDot
                  health={n.up ? "up" : "down"}
                  pulse={!n.up}
                  size="sm"
                  ariaLabel={n.up ? t.monitoring.statusUp : t.monitoring.statusDown}
                />
                <span className="font-medium truncate">{n.displayName}</span>
              </div>
              <p className="text-xs text-muted mt-0.5">
                {n.up
                  ? `${formatNumber(n.latencyP95Ms, 0)} ${t.monitoring.unitMs} p95`
                  : t.monitoring.statusDown}
              </p>
            </div>
          </div>
        ))}
      </div>
    </GlassCard>
  );
}

interface Headline {
  text: string;
  tone: string;
}

function thresholdHeadline(upCount: number, t: ReturnType<typeof useTranslations>): Headline {
  if (upCount >= 3) {
    return { text: t.monitoring.mpcAllHealthy, tone: "text-[var(--glass-success)]" };
  }
  if (upCount === 2) {
    return { text: t.monitoring.mpcThresholdOk, tone: "text-[var(--glass-warning)]" };
  }
  return { text: t.monitoring.mpcThresholdLost, tone: "text-[var(--glass-danger)]" };
}

function formatNumber(n: number, digits: number): string {
  if (!Number.isFinite(n)) return "—";
  return n.toLocaleString("en-US", {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  });
}
