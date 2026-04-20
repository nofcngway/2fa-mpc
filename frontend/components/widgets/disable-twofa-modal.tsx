"use client";

import { useState } from "react";
import { Modal } from "@heroui/react";
import { OTPInput } from "@/components/ui/otp-input";
import { BackupCodeInput } from "@/components/ui/backup-code-input";
import { GlassButton } from "@/components/ui/glass-button";
import { use2FA } from "@/hooks/use-2fa";
import { ApiRequestError } from "@/lib/api";
import { mapApiErrorMessage } from "@/lib/utils";
import { toast } from "@heroui/react";

type Mode = "otp" | "backup";

interface DisableTwoFAModalProps {
  isOpen: boolean;
  onClose: () => void;
  userId: string;
  onDisabled: () => void;
}

export function DisableTwoFAModal({
  isOpen,
  onClose,
  userId,
  onDisabled,
}: DisableTwoFAModalProps) {
  const { disable } = use2FA();
  const [mode, setMode] = useState<Mode>("otp");
  const [otpValue, setOtpValue] = useState("");
  const [backupValue, setBackupValue] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const reset = () => {
    setMode("otp");
    setOtpValue("");
    setBackupValue("");
    setError(null);
    setIsSubmitting(false);
  };

  const handleClose = () => {
    reset();
    onClose();
  };

  const handleSubmit = async (directCode?: string) => {
    const code = directCode || (mode === "otp" ? otpValue : backupValue.trim());
    if (!code) {
      setError(mode === "otp" ? "Enter your 6-digit code" : "Enter your backup code");
      return;
    }

    setIsSubmitting(true);
    setError(null);
    try {
      await disable(userId, code);
      toast("Two-factor authentication disabled", { variant: "success" });
      reset();
      onDisabled();
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setError(mapApiErrorMessage(err.code, err.message));
      } else {
        setError("Failed to disable 2FA. Try again.");
      }
      if (mode === "otp") setOtpValue("");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onOpenChange={(open) => !open && handleClose()}>
      <Modal.Backdrop>
        <Modal.Container size="md">
          <Modal.Dialog className="glass-card-elevated overflow-hidden">
            <Modal.CloseTrigger />
            <Modal.Header>
              <Modal.Heading>Disable Two-Factor Authentication</Modal.Heading>
            </Modal.Header>
            <Modal.Body>
              <div className="flex flex-col gap-5">
                <p className="text-sm text-muted">
                  {mode === "otp"
                    ? "Enter the 6-digit code from your authenticator app."
                    : "Enter one of your backup codes."}
                </p>

                {mode === "otp" ? (
                  <div className="flex justify-center">
                    <OTPInput
                      value={otpValue}
                      onChange={(v) => {
                        setOtpValue(v);
                        setError(null);
                      }}
                      onComplete={handleSubmit}
                      error={error ?? undefined}
                      isDisabled={isSubmitting}
                    />
                  </div>
                ) : (
                  <BackupCodeInput
                    value={backupValue}
                    onChange={(v) => {
                      setBackupValue(v);
                      setError(null);
                    }}
                    error={error ?? undefined}
                    isDisabled={isSubmitting}
                  />
                )}

                <GlassButton
                  variant="danger"
                  size="md"
                  isLoading={isSubmitting}
                  onPress={handleSubmit}
                  className="w-full"
                >
                  Disable 2FA
                </GlassButton>

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
                      setError(null);
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
              </div>
            </Modal.Body>
          </Modal.Dialog>
        </Modal.Container>
      </Modal.Backdrop>
    </Modal>
  );
}
