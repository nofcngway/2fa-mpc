"use client";

type Health = "up" | "down" | "degraded";
type Size = "sm" | "md";

interface ServiceStatusDotProps {
  health: Health;
  size?: Size;
  pulse?: boolean;
  ariaLabel?: string;
}

const colorByHealth: Record<Health, string> = {
  up: "var(--glass-success)",
  down: "var(--glass-danger)",
  degraded: "var(--glass-warning)",
};

const sizePx: Record<Size, number> = {
  sm: 8,
  md: 10,
};

/**
 * ServiceStatusDot — atomic indicator for a single service's health.
 *
 * Style follows the project's Liquid Glass tokens (--glass-success/danger/
 * warning). The optional `pulse` adds a subtle animated glow to draw the eye
 * to "down" services without being distracting.
 */
export function ServiceStatusDot({
  health,
  size = "md",
  pulse = false,
  ariaLabel,
}: ServiceStatusDotProps) {
  const px = sizePx[size];
  const color = colorByHealth[health];

  return (
    <span
      role="status"
      aria-label={ariaLabel ?? health}
      style={{
        display: "inline-block",
        width: px,
        height: px,
        borderRadius: "9999px",
        background: color,
        boxShadow: pulse ? `0 0 0 0 ${color}` : `0 0 4px ${color}80`,
        animation: pulse ? "service-status-pulse 2s ease-in-out infinite" : undefined,
      }}
    />
  );
}
