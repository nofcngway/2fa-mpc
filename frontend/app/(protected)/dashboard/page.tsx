"use client";

import { UserInfoCard } from "@/components/widgets/user-info-card";
import { TwoFAStatusCard } from "@/components/widgets/twofa-status-card";

export default function DashboardPage() {
  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold">Dashboard</h1>
      <div className="flex flex-col gap-4">
        <UserInfoCard />
        <TwoFAStatusCard />
      </div>
    </div>
  );
}
