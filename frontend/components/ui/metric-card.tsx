"use client";

import type { ReactNode } from "react";
import { GlassCard } from "@/components/ui/glass-card";

type Tone = "neutral" | "success" | "warning" | "danger" | "accent";

interface MetricCardProps {
  label: string;
  value: string;
  unit?: string;
  hint?: string;
  icon?: ReactNode;
  tone?: Tone;
  className?: string;
}

const toneAccentBg: Record<Tone, string> = {
  neutral: "var(--glass-bg-elevated)",
  success: "color-mix(in oklch, var(--glass-success) 12%, transparent)",
  warning: "color-mix(in oklch, var(--glass-warning) 12%, transparent)",
  danger: "color-mix(in oklch, var(--glass-danger) 12%, transparent)",
  accent: "var(--accent-subtle)",
};

const toneIconColor: Record<Tone, string> = {
  neutral: "var(--accent)",
  success: "var(--glass-success)",
  warning: "var(--glass-warning)",
  danger: "var(--glass-danger)",
  accent: "var(--accent)",
};

/**
 * MetricCard — atomic single-metric tile used by monitoring widgets.
 *
 * Composition: GlassCard frame + optional icon badge + label + large numeric
 * value (mono font for stable digit width) + optional unit + optional hint.
 */
export function MetricCard({
  label,
  value,
  unit,
  hint,
  icon,
  tone = "neutral",
  className = "",
}: MetricCardProps) {
  return (
    <GlassCard className={`p-5 ${className}`}>
      <div className="flex items-start gap-4">
        {icon && (
          <div
            className="w-10 h-10 rounded-xl flex items-center justify-center shrink-0"
            style={{ background: toneAccentBg[tone], color: toneIconColor[tone] }}
          >
            {icon}
          </div>
        )}
        <div className="flex-1 min-w-0">
          <p className="text-xs uppercase tracking-wider text-muted">{label}</p>
          <p className="mt-1 flex items-baseline gap-1">
            <span className="text-2xl font-semibold font-mono">{value}</span>
            {unit && <span className="text-sm text-muted">{unit}</span>}
          </p>
          {hint && <p className="text-xs text-muted mt-0.5 truncate">{hint}</p>}
        </div>
      </div>
    </GlassCard>
  );
}
