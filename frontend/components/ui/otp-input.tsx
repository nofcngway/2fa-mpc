"use client";

import { InputOTP, REGEXP_ONLY_DIGITS } from "@heroui/react";

interface OTPInputProps {
  length?: number;
  value: string;
  onChange: (value: string) => void;
  onComplete?: (value: string) => void;
  error?: string;
  isDisabled?: boolean;
  autoFocus?: boolean;
}

export function OTPInput({
  length = 6,
  value,
  onChange,
  onComplete,
  error,
  isDisabled = false,
  autoFocus = true,
}: OTPInputProps) {
  const halfLength = Math.ceil(length / 2);

  const handleChange = (val: string) => {
    onChange(val);
    if (val.length === length && onComplete) {
      // Defer to avoid calling async operations inside React's onChange cycle
      setTimeout(() => onComplete(val), 0);
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <InputOTP
        maxLength={length}
        value={value}
        onChange={handleChange}
        pattern={REGEXP_ONLY_DIGITS}
        isDisabled={isDisabled}
        autoFocus={autoFocus}
      >
        <InputOTP.Group>
          {Array.from({ length: halfLength }, (_, i) => (
            <InputOTP.Slot
              key={i}
              index={i}
              className={`glass-otp-slot ${error ? "!border-[var(--glass-danger)]" : ""}`}
            />
          ))}
        </InputOTP.Group>
        <InputOTP.Separator />
        <InputOTP.Group>
          {Array.from({ length: length - halfLength }, (_, i) => (
            <InputOTP.Slot
              key={halfLength + i}
              index={halfLength + i}
              className={`glass-otp-slot ${error ? "!border-[var(--glass-danger)]" : ""}`}
            />
          ))}
        </InputOTP.Group>
      </InputOTP>
      {error && (
        <p className="text-xs text-[var(--glass-danger)] text-center">{error}</p>
      )}
    </div>
  );
}
