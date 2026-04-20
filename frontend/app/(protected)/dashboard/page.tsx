"use client";

import { UserInfoCard } from "@/components/widgets/user-info-card";
import { TwoFAStatusCard } from "@/components/widgets/twofa-status-card";
import { useTranslations } from "@/lib/i18n";

export default function DashboardPage() {
  const t = useTranslations();

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold">{t.dashboard.title}</h1>
      <div className="flex flex-col gap-4">
        <UserInfoCard />
        <TwoFAStatusCard />
      </div>
    </div>
  );
}
