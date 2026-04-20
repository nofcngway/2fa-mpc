"use client";

import Link from "next/link";
import { TwoFASetupWizard } from "@/components/widgets/twofa-setup-wizard";
import { useTranslations } from "@/lib/i18n";
import { ArrowLeft } from "lucide-react";

export default function TwoFASetupPage() {
  const t = useTranslations();

  return (
    <div className="flex flex-col gap-6 max-w-lg mx-auto">
      <Link
        href="/dashboard"
        className="inline-flex items-center gap-1.5 text-sm text-muted hover:text-foreground transition-colors w-fit"
      >
        <ArrowLeft size={16} />
        {t.common.backToDashboard}
      </Link>
      <h1 className="text-2xl font-semibold text-center">{t.setup.title}</h1>
      <TwoFASetupWizard />
    </div>
  );
}
