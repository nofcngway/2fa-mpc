"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { GlassCard } from "@/components/ui/glass-card";
import { OTPVerifyForm } from "@/components/widgets/otp-verify-form";
import { LoadingSpinner } from "@/components/ui/loading-spinner";
import { getStoredUser, getPending2FA, setPending2FA } from "@/lib/auth";
import { apiRequest, ApiRequestError } from "@/lib/api";
import { mapApiErrorMessage } from "@/lib/utils";
import type { Verify2FAResponse } from "@/lib/types";

export default function TwoFAVerifyPage() {
  const router = useRouter();
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
    if (!user) {
      router.replace("/login");
      return;
    }

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
        setError("Invalid code. Please try again.");
      }
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setError(mapApiErrorMessage(err.code, err.message));
      } else {
        setError("Verification failed. Try again.");
      }
    } finally {
      setIsLoading(false);
    }
  };

  if (!ready) {
    return <LoadingSpinner fullPage />;
  }

  return (
    <GlassCard variant="elevated" className="p-8">
      <div className="text-center mb-6">
        <h1 className="text-2xl font-semibold">Two-Factor Verification</h1>
        <p className="text-sm text-muted mt-1">
          Your account has 2FA enabled. Enter the code to continue.
        </p>
      </div>
      <OTPVerifyForm
        onVerify={handleVerify}
        isLoading={isLoading}
        error={error}
        showModeToggle
        title=""
        description="Enter the 6-digit code from your authenticator app"
      />
    </GlassCard>
  );
}
