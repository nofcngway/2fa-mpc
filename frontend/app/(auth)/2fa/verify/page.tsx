"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { GlassCard } from "@/components/ui/glass-card";
import { OTPVerifyForm } from "@/components/widgets/otp-verify-form";
import { LoadingSpinner } from "@/components/ui/loading-spinner";
import { getStoredUser, getPending2FA, setPending2FA } from "@/lib/auth";
import { apiRequest, ApiRequestError } from "@/lib/api";
import { mapApiErrorCode } from "@/lib/utils";
import { useTranslations } from "@/lib/i18n";
import type { Verify2FAResponse } from "@/lib/types";

export default function TwoFAVerifyPage() {
  const router = useRouter();
  const t = useTranslations();
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (!getPending2FA()) {
      router.replace("/dashboard");
      return;
    }
    setReady(true);
  }, [router]);

  const handleVerify = async (code: string) => {
    const user = getStoredUser();
    if (!user) { router.replace("/login"); return; }

    setIsLoading(true);
    setError(null);
    try {
      const result = await apiRequest<Verify2FAResponse>("/api/v1/2fa/verify", {
        method: "POST",
        body: { userId: user.id, otpCode: code },
      });
      if (result.valid) {
        setPending2FA(false);
        router.replace("/dashboard");
      } else {
        setError(t.setup.invalidCode);
      }
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setError(mapApiErrorCode(err.code, t));
      } else {
        setError(t.setup.verificationFailed);
      }
    } finally {
      setIsLoading(false);
    }
  };

  if (!ready) return <LoadingSpinner fullPage />;

  return (
    <GlassCard variant="elevated" className="p-8">
      <div className="text-center mb-6">
        <h1 className="text-2xl font-semibold">{t.verifyLogin.title}</h1>
        <p className="text-sm text-muted mt-1">{t.verifyLogin.subtitle}</p>
      </div>
      <OTPVerifyForm
        onVerify={handleVerify}
        isLoading={isLoading}
        error={error}
        showModeToggle
        title=""
        description={t.verifyLogin.description}
      />
    </GlassCard>
  );
}
