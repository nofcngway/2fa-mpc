"use client";

import { TextField, Label, InputGroup, FieldError } from "@heroui/react";
import type { ReactNode } from "react";

interface GlassInputProps {
  label?: string;
  type?: string;
  placeholder?: string;
  value?: string;
  onChange?: (value: string) => void;
  error?: string;
  isDisabled?: boolean;
  autoComplete?: string;
  size?: "sm" | "md" | "lg";
  className?: string;
  suffix?: ReactNode;
}

const sizeClasses: Record<string, string> = {
  sm: "text-sm",
  md: "text-base",
  lg: "text-lg",
};

export function GlassInput({
  label,
  type = "text",
  placeholder,
  value,
  onChange,
  error,
  isDisabled = false,
  autoComplete,
  size = "md",
  className = "",
  suffix,
}: GlassInputProps) {
  return (
    <div className={sizeClasses[size]}>
      <TextField
        isInvalid={!!error}
        isDisabled={isDisabled}
        onChange={(v) => onChange?.(v)}
        value={value}
        className={className}
      >
        {label && <Label>{label}</Label>}
        <InputGroup>
          <InputGroup.Input
            type={type}
            placeholder={placeholder}
            autoComplete={autoComplete}
            className="glass-input rounded-xl"
          />
          {suffix && <InputGroup.Suffix>{suffix}</InputGroup.Suffix>}
        </InputGroup>
        {error && <FieldError>{error}</FieldError>}
      </TextField>
    </div>
  );
}
