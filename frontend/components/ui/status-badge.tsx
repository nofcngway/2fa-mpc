import { Chip } from "@heroui/react";
import { ShieldCheck, ShieldOff, Loader } from "lucide-react";

type Status = "enabled" | "disabled" | "pending";
type Size = "sm" | "md";

interface StatusBadgeProps {
  status: Status;
  size?: Size;
}

const statusConfig: Record<
  Status,
  { label: string; color: string; icon: typeof ShieldCheck }
> = {
  enabled: {
    label: "Enabled",
    color: "text-[var(--glass-success)]",
    icon: ShieldCheck,
  },
  disabled: {
    label: "Disabled",
    color: "text-muted",
    icon: ShieldOff,
  },
  pending: {
    label: "Pending",
    color: "text-[var(--glass-warning)]",
    icon: Loader,
  },
};

export function StatusBadge({ status, size = "md" }: StatusBadgeProps) {
  const cfg = statusConfig[status];
  const Icon = cfg.icon;
  const iconSize = size === "sm" ? 14 : 16;

  return (
    <Chip
      className={`
        bg-[var(--glass-bg-elevated)] border border-[var(--glass-border-subtle)]
        ${cfg.color} ${size === "sm" ? "text-xs" : "text-sm"}
      `}
    >
      <Icon size={iconSize} className="inline mr-1" />
      {cfg.label}
    </Chip>
  );
}
