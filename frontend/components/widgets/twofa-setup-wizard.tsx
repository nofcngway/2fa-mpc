"use client";

import { useState, useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import { GlassCard } from "@/components/ui/glass-card";
import { LoadingSpinner } from "@/components/ui/loading-spinner";
import { QRCodeDisplay } from "@/components/widgets/qr-code-display";
import { OTPVerifyForm } from "@/components/widgets/otp-verify-form";
import { BackupCodesDisplay } from "@/components/widgets/backup-codes-display";
import { useAuth } from "@/hooks/use-auth";
import { use2FA } from "@/hooks/use-2fa";
import { ApiRequestError } from "@/lib/api";
import { mapApiErrorCode } from "@/lib/utils";
import { useTranslations } from "@/lib/i18n";
import { toast } from "@heroui/react";

type Step = "loading" | "qr" | "verify" | "backup";

export function TwoFASetupWizard() {
  const router = useRouter();
  const { user } = useAuth();
  const { setup, verify } = use2FA();
  const t = useTranslations();
  const [step, setStep] = useState<Step>("loading");
  const [provisioningUri, setProvisioningUri] = useState("");
  const [backupCodes, setBackupCodes] = useState<string[]>([]);
  const [verifyError, setVerifyError] = useState<string | null>(null);
  const [isVerifying, setIsVerifying] = useState(false);
  const setupCalled = useRef(false);

  useEffect(() => {
    if (!user || setupCalled.current) return;
    setupCalled.current = true;

    const init = async () => {
      try {
        const data = await setup(user.id, user.email);
        setProvisioningUri(data.provisioningUri);
        setBackupCodes(data.backupCodes);
        setStep("qr");
      } catch (err) {
        if (err instanceof ApiRequestError) {
          toast(mapApiErrorCode(err.code, t), { variant: "danger" });
        } else {
          toast(t.common.error, { variant: "danger" });
        }
        router.replace("/dashboard");
      }
    };

    init();
  }, [user, setup, router, t]);

  const handleVerify = async (code: string) => {
    if (!user) return;
    setIsVerifying(true);
    setVerifyError(null);
    try {
      const result = await verify(user.id, code);
      if (result.valid) {
        toast(t.setup.enabled, { variant: "success" });
        setStep("backup");
      } else {
        setVerifyError(t.setup.invalidCode);
      }
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setVerifyError(mapApiErrorCode(err.code, t));
      } else {
        setVerifyError(t.setup.verificationFailed);
      }
    } finally {
      setIsVerifying(false);
    }
  };

  const handleDone = () => {
    router.push("/dashboard");
  };

  const steps = [t.setup.stepScan, t.setup.stepVerify, t.setup.stepBackup];
  const currentStep = step === "qr" ? 0 : step === "verify" ? 1 : step === "backup" ? 2 : -1;

  return (
    <div className="flex flex-col gap-6">
      {currentStep >= 0 && (
        <div className="flex items-center justify-center gap-2">
          {steps.map((label, i) => (
            <div key={label} className="flex items-center gap-2">
              <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium transition-all duration-300 ${
                i <= currentStep
                  ? "bg-[var(--accent)] text-[var(--accent-foreground)]"
                  : "bg-[var(--glass-bg-elevated)] text-muted border border-[var(--glass-border-subtle)]"
              }`}>
                {i + 1}
              </div>
              <span className={`text-sm hidden sm:inline ${i <= currentStep ? "text-foreground font-medium" : "text-muted"}`}>
                {label}
              </span>
              {i < steps.length - 1 && (
                <div className={`w-8 h-px mx-1 ${i < currentStep ? "bg-[var(--accent)]" : "bg-[var(--glass-border-subtle)]"}`} />
              )}
            </div>
          ))}
        </div>
      )}

      <GlassCard variant="elevated" className="p-8">
        {step === "loading" && <LoadingSpinner label={t.setup.settingUp} />}
        {step === "qr" && <QRCodeDisplay provisioningUri={provisioningUri} onNext={() => setStep("verify")} />}
        {step === "verify" && <OTPVerifyForm onVerify={handleVerify} isLoading={isVerifying} error={verifyError} />}
        {step === "backup" && <BackupCodesDisplay codes={backupCodes} onDone={handleDone} />}
      </GlassCard>
    </div>
  );
}
