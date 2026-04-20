"use client";

import { useState } from "react";
import { OTPInput } from "@/components/ui/otp-input";
import { BackupCodeInput } from "@/components/ui/backup-code-input";
import { GlassButton } from "@/components/ui/glass-button";
import { ShieldCheck } from "lucide-react";

type Mode = "otp" | "backup";

interface OTPVerifyFormProps {
  onVerify: (code: string) => Promise<void>;
  isLoading: boolean;
  error: string | null;
  showModeToggle?: boolean;
  title?: string;
  description?: string;
}

export function OTPVerifyForm({
  onVerify,
  isLoading,
  error,
  showModeToggle = false,
  title = "Verify your code",
  description = "Enter the 6-digit code from your authenticator app",
}: OTPVerifyFormProps) {
  const [mode, setMode] = useState<Mode>("otp");
  const [otpValue, setOtpValue] = useState("");
  const [backupValue, setBackupValue] = useState("");

  const handleSubmit = () => {
    const code = mode === "otp" ? otpValue : backupValue.trim();
    if (!code) return;
    onVerify(code);
  };

  const handleOtpComplete = (code: string) => {
    onVerify(code);
  };

  return (
    <div className="flex flex-col gap-6 items-center">
      <div className="flex items-center gap-3">
        <div className="w-10 h-10 rounded-2xl bg-[var(--accent-subtle)] flex items-center justify-center">
          <ShieldCheck size={20} className="text-[var(--accent)]" />
        </div>
        <div>
          <h2 className="text-lg font-semibold">{title}</h2>
          <p className="text-sm text-muted">
            {mode === "otp" ? description : "Enter one of your backup codes"}
          </p>
        </div>
      </div>

      {mode === "otp" ? (
        <OTPInput
          value={otpValue}
          onChange={(v) => setOtpValue(v)}
          onComplete={handleOtpComplete}
          error={error ?? undefined}
          isDisabled={isLoading}
        />
      ) : (
        <div className="w-full max-w-sm">
          <BackupCodeInput
            value={backupValue}
            onChange={(v) => setBackupValue(v)}
            error={error ?? undefined}
            isDisabled={isLoading}
          />
        </div>
      )}

      <GlassButton
        variant="primary"
        size="lg"
        isLoading={isLoading}
        onPress={handleSubmit}
        className="w-full max-w-sm"
      >
        Verify
      </GlassButton>

      {showModeToggle && (
        <div className="text-center">
          <div className="flex items-center gap-3 mb-3">
            <div className="flex-1 h-px bg-[var(--glass-border-subtle)]" />
            <span className="text-xs text-muted">or</span>
            <div className="flex-1 h-px bg-[var(--glass-border-subtle)]" />
          </div>
          <button
            type="button"
            onClick={() => {
              setMode(mode === "otp" ? "backup" : "otp");
              setOtpValue("");
              setBackupValue("");
            }}
            className="text-sm text-[var(--accent)] hover:underline cursor-pointer"
          >
            {mode === "otp"
              ? "Use a backup code instead"
              : "Use authenticator app"}
          </button>
        </div>
      )}
    </div>
  );
}
