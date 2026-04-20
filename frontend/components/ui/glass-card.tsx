import type { ComponentPropsWithoutRef } from "react";

type Variant = "default" | "elevated" | "flat";

interface GlassCardProps extends ComponentPropsWithoutRef<"div"> {
  variant?: Variant;
}

const variantClasses: Record<Variant, string> = {
  default: "glass-card",
  elevated: "glass-card-elevated",
  flat: "glass-card-flat",
};

export function GlassCard({
  variant = "default",
  className = "",
  children,
  ...props
}: GlassCardProps) {
  return (
    <div className={`${variantClasses[variant]} ${className}`} {...props}>
      {children}
    </div>
  );
}
