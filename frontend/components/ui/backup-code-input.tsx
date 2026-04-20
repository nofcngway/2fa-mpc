"use client";

import { TextField, Label, Input, FieldError } from "@heroui/react";

interface BackupCodeInputProps {
  value: string;
  onChange: (value: string) => void;
  error?: string;
  placeholder?: string;
  isDisabled?: boolean;
}

export function BackupCodeInput({
  value,
  onChange,
  error,
  placeholder = "e.g. a1b2c3d4",
  isDisabled = false,
}: BackupCodeInputProps) {
  return (
    <TextField
      isInvalid={!!error}
      isDisabled={isDisabled}
      onChange={(v) => onChange(v)}
      value={value}
    >
      <Label>Backup code</Label>
      <Input
        placeholder={placeholder}
        autoComplete="off"
        className="glass-input rounded-xl font-mono"
      />
      {error && <FieldError>{error}</FieldError>}
    </TextField>
  );
}
