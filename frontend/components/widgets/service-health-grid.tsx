"use client";

import { GlassCard } from "@/components/ui/glass-card";
import { ServiceStatusDot } from "@/components/ui/service-status-dot";
import { useTranslations } from "@/lib/i18n";
import type { MonitoringServiceSnapshot, MonitoringSnapshot } from "@/lib/types";

interface ServiceHealthGridProps {
  snapshot: MonitoringSnapshot | null;
}

/**
 * ServiceHealthGrid — table of every monitored service with status dot,
 * RPS, p95 latency, and error rate. The widget is read-only; clicking a row
 * is intentionally a no-op for now (drill-down is a future iteration).
 */
export function ServiceHealthGrid({ snapshot }: ServiceHealthGridProps) {
  const t = useTranslations();

  if (!snapshot || snapshot.services.length === 0) {
    return (
      <GlassCard className="p-6">
        <p className="text-sm text-muted">{t.monitoring.noData}</p>
      </GlassCard>
    );
  }

  return (
    <GlassCard className="p-0 overflow-hidden">
      <div className="px-6 py-4 border-b border-[var(--glass-border-subtle)]">
        <h2 className="text-lg font-semibold">{t.monitoring.servicesTitle}</h2>
        <p className="text-sm text-muted">{t.monitoring.servicesSubtitle}</p>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="text-xs uppercase tracking-wider text-muted">
              <th className="text-left font-medium px-6 py-3">{t.monitoring.colService}</th>
              <th className="text-right font-medium px-6 py-3 font-mono">{t.monitoring.colRps}</th>
              <th className="text-right font-medium px-6 py-3 font-mono">{t.monitoring.colP95}</th>
              <th className="text-right font-medium px-6 py-3 font-mono">{t.monitoring.colErrors}</th>
            </tr>
          </thead>
          <tbody>
            {snapshot.services.map((svc) => (
              <ServiceRow key={svc.name} service={svc} t={t} />
            ))}
          </tbody>
        </table>
      </div>
    </GlassCard>
  );
}

interface ServiceRowProps {
  service: MonitoringServiceSnapshot;
  t: ReturnType<typeof useTranslations>;
}

function ServiceRow({ service, t }: ServiceRowProps) {
  const errPct = service.errorRate * 100;
  const errTone =
    errPct > 1 ? "text-[var(--glass-danger)]" : errPct > 0 ? "text-[var(--glass-warning)]" : "";

  return (
    <tr className="border-t border-[var(--glass-border-subtle)] hover:bg-[var(--accent-subtle)] transition-colors">
      <td className="px-6 py-4">
        <div className="flex items-center gap-3">
          <ServiceStatusDot
            health={service.up ? "up" : "down"}
            pulse={!service.up}
            ariaLabel={service.up ? t.monitoring.statusUp : t.monitoring.statusDown}
          />
          <span className="font-medium">{service.displayName}</span>
        </div>
      </td>
      <td className="text-right px-6 py-4 font-mono tabular-nums">
        {service.up ? formatNumber(service.rps, 1) : "—"}
      </td>
      <td className="text-right px-6 py-4 font-mono tabular-nums">
        {service.up ? `${formatNumber(service.latencyP95Ms, 0)} ${t.monitoring.unitMs}` : "—"}
      </td>
      <td className={`text-right px-6 py-4 font-mono tabular-nums ${errTone}`}>
        {service.up ? `${formatNumber(errPct, 2)}%` : "—"}
      </td>
    </tr>
  );
}

function formatNumber(n: number, digits: number): string {
  if (!Number.isFinite(n)) return "—";
  return n.toLocaleString("en-US", {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  });
}
