"use client";

import { Button, Spinner } from "@heroui/react";
import type { ReactNode } from "react";

type Variant = "primary" | "secondary" | "ghost" | "danger";
type Size = "sm" | "md" | "lg";

interface GlassButtonProps {
  variant?: Variant;
  size?: Size;
  isLoading?: boolean;
  isDisabled?: boolean;
  icon?: ReactNode;
  children?: ReactNode;
  className?: string;
  onPress?: () => void;
  type?: "button" | "submit" | "reset";
  "aria-label"?: string;
}

const variantStyles: Record<Variant, string> = {
  primary: [
    "bg-[var(--accent)] text-[var(--accent-foreground)]",
    "shadow-[0_4px_16px_var(--accent-glow)]",
    "hover:shadow-[0_6px_24px_var(--accent-glow)]",
    "hover:-translate-y-[1px]",
    "active:translate-y-0",
    "transition-all duration-200",
  ].join(" "),
  secondary: [
    "bg-[var(--glass-bg-elevated)] text-foreground",
    "border border-[var(--glass-border-subtle)]",
    "backdrop-blur-sm",
    "hover:bg-[var(--glass-bg-solid)]",
    "transition-all duration-200",
  ].join(" "),
  ghost: [
    "bg-transparent text-foreground",
    "hover:bg-[var(--accent-subtle)]",
    "transition-all duration-200",
  ].join(" "),
  danger: [
    "bg-[var(--glass-danger)] text-white",
    "hover:opacity-90",
    "transition-all duration-200",
  ].join(" "),
};

const sizeStyles: Record<Size, string> = {
  sm: "h-8 px-3 text-sm rounded-lg",
  md: "h-10 px-5 text-sm rounded-xl",
  lg: "h-12 px-6 text-base rounded-xl",
};

export function GlassButton({
  variant = "primary",
  size = "md",
  isLoading = false,
  isDisabled,
  icon,
  children,
  className = "",
  onPress,
  type = "button",
  "aria-label": ariaLabel,
}: GlassButtonProps) {
  return (
    <Button
      isDisabled={isDisabled || isLoading}
      className={`font-medium ${variantStyles[variant]} ${sizeStyles[size]} ${className}`}
      onPress={onPress}
      type={type}
      aria-label={ariaLabel}
    >
      {isLoading && <Spinner size="sm" className="mr-2" />}
      {!isLoading && icon && <span className="mr-2 flex items-center">{icon}</span>}
      {children}
    </Button>
  );
}
