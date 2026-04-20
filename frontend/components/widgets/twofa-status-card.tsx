"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { GlassCard } from "@/components/ui/glass-card";
import { GlassButton } from "@/components/ui/glass-button";
import { StatusBadge } from "@/components/ui/status-badge";
import { LoadingSpinner } from "@/components/ui/loading-spinner";
import { DisableTwoFAModal } from "@/components/widgets/disable-twofa-modal";
import { useAuth } from "@/hooks/use-auth";
import { use2FA } from "@/hooks/use-2fa";
import { useTranslations, useLocale } from "@/lib/i18n";
import { formatDate } from "@/lib/utils";
import { ShieldCheck, ShieldPlus } from "lucide-react";

export function TwoFAStatusCard() {
  const router = useRouter();
  const { user } = useAuth();
  const { status, isLoading, fetchStatus } = use2FA();
  const t = useTranslations();
  const { locale } = useLocale();
  const [showDisableModal, setShowDisableModal] = useState(false);

  useEffect(() => {
    if (user) {
      fetchStatus(user.id).catch(() => {});
    }
  }, [user, fetchStatus]);

  if (isLoading && !status) {
    return (
      <GlassCard className="p-6">
        <LoadingSpinner size="sm" label={t.twofa.loadingStatus} />
      </GlassCard>
    );
  }

  const isEnabled = status?.isEnabled ?? false;

  return (
    <>
      <GlassCard className="p-6">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-4">
            <div
              className={`w-12 h-12 rounded-2xl flex items-center justify-center ${
                isEnabled ? "bg-[var(--glass-success)]/10" : "bg-[var(--glass-bg-elevated)]"
              }`}
            >
              {isEnabled ? (
                <ShieldCheck size={24} className="text-[var(--glass-success)]" />
              ) : (
                <ShieldPlus size={24} className="text-muted" />
              )}
            </div>
            <div>
              <div className="flex items-center gap-3">
                <h2 className="text-lg font-semibold">{t.twofa.title}</h2>
                <StatusBadge status={isEnabled ? "enabled" : "disabled"} size="sm" />
              </div>
              {isEnabled && status?.createdAt && (
                <p className="text-sm text-muted mt-0.5">
                  {t.twofa.enabledSince} {formatDate(status.createdAt, locale)}
                </p>
              )}
              {!isEnabled && (
                <p className="text-sm text-muted mt-0.5">{t.twofa.addSecurity}</p>
              )}
            </div>
          </div>

          {isEnabled ? (
            <GlassButton variant="danger" size="sm" onPress={() => setShowDisableModal(true)}>
              {t.twofa.disable}
            </GlassButton>
          ) : (
            <GlassButton variant="primary" size="sm" onPress={() => router.push("/2fa/setup")}>
              {t.twofa.enable}
            </GlassButton>
          )}
        </div>
      </GlassCard>

      {user && showDisableModal && (
        <DisableTwoFAModal
          isOpen={showDisableModal}
          onClose={() => setShowDisableModal(false)}
          userId={user.id}
          onDisabled={() => {
            setShowDisableModal(false);
            if (user) fetchStatus(user.id).catch(() => {});
          }}
        />
      )}
    </>
  );
}
