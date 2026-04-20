"use client";

import { Chip } from "@heroui/react";
import { ShieldCheck, ShieldOff, Loader } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

type Status = "enabled" | "disabled" | "pending";
type Size = "sm" | "md";

interface StatusBadgeProps {
  status: Status;
  size?: Size;
}

const statusIcons: Record<Status, typeof ShieldCheck> = {
  enabled: ShieldCheck,
  disabled: ShieldOff,
  pending: Loader,
};

const statusColors: Record<Status, string> = {
  enabled: "text-[var(--glass-success)]",
  disabled: "text-muted",
  pending: "text-[var(--glass-warning)]",
};

export function StatusBadge({ status, size = "md" }: StatusBadgeProps) {
  const t = useTranslations();
  const Icon = statusIcons[status];
  const color = statusColors[status];
  const label = t.twofa[status];
  const iconSize = size === "sm" ? 14 : 16;

  return (
    <Chip
      className={`
        bg-[var(--glass-bg-elevated)] border border-[var(--glass-border-subtle)]
        ${color} ${size === "sm" ? "text-xs" : "text-sm"}
      `}
    >
      <Icon size={iconSize} className="inline mr-1" />
      {label}
    </Chip>
  );
}
