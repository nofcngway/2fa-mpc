"use client";

import { useState } from "react";
import { TextField, Label, FieldError } from "@heroui/react";
import { Eye, EyeOff } from "lucide-react";

interface PasswordInputProps {
  label?: string;
  placeholder?: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
  isDisabled?: boolean;
  autoComplete?: string;
}

export function PasswordInput({
  label = "Password",
  placeholder = "Enter your password",
  value,
  onChange,
  error,
  isDisabled = false,
  autoComplete = "current-password",
}: PasswordInputProps) {
  const [visible, setVisible] = useState(false);

  return (
    <TextField
      isInvalid={!!error}
      isDisabled={isDisabled}
      onChange={(v) => onChange(v)}
      value={value}
    >
      <Label>{label}</Label>
      <div className="relative">
        <input
          type={visible ? "text" : "password"}
          placeholder={placeholder}
          autoComplete={autoComplete}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          disabled={isDisabled}
          className="glass-input rounded-xl w-full h-10 px-3 pr-10 text-sm text-foreground placeholder:text-muted outline-none"
        />
        <button
          type="button"
          tabIndex={-1}
          className="absolute right-2 top-1/2 -translate-y-1/2 text-muted hover:text-foreground transition-colors p-1 cursor-pointer"
          onClick={() => setVisible(!visible)}
          aria-label={visible ? "Hide password" : "Show password"}
        >
          {visible ? <EyeOff size={18} /> : <Eye size={18} />}
        </button>
      </div>
      {error && <FieldError>{error}</FieldError>}
    </TextField>
  );
}
