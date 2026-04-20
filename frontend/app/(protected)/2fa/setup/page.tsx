"use client";

import Link from "next/link";
import { TwoFASetupWizard } from "@/components/widgets/twofa-setup-wizard";
import { ArrowLeft } from "lucide-react";

export default function TwoFASetupPage() {
  return (
    <div className="flex flex-col gap-6 max-w-lg mx-auto">
      <Link
        href="/dashboard"
        className="inline-flex items-center gap-1.5 text-sm text-muted hover:text-foreground transition-colors w-fit"
      >
        <ArrowLeft size={16} />
        Back to Dashboard
      </Link>
      <h1 className="text-2xl font-semibold text-center">
        Setup Two-Factor Authentication
      </h1>
      <TwoFASetupWizard />
    </div>
  );
}
