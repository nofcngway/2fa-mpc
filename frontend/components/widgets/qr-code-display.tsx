"use client";

import { useState } from "react";
import { QRCodeSVG } from "qrcode.react";
import { GlassCard } from "@/components/ui/glass-card";
import { GlassButton } from "@/components/ui/glass-button";
import { copyToClipboard } from "@/lib/utils";
import { useTranslations } from "@/lib/i18n";
import { toast } from "@heroui/react";
import { Copy, QrCode } from "lucide-react";

interface QRCodeDisplayProps {
  provisioningUri: string;
  onNext: () => void;
}

export function QRCodeDisplay({ provisioningUri, onNext }: QRCodeDisplayProps) {
  const t = useTranslations();
  const [showManual, setShowManual] = useState(false);

  const handleCopyUri = async () => {
    try {
      await copyToClipboard(provisioningUri);
      toast(t.common.copied, { variant: "success" });
    } catch {
      toast(t.common.copyFailed, { variant: "danger" });
    }
  };

  return (
    <div className="flex flex-col gap-6 items-center">
      <div className="flex items-center gap-3">
        <div className="w-10 h-10 rounded-2xl bg-[var(--accent-subtle)] flex items-center justify-center">
          <QrCode size={20} className="text-[var(--accent)]" />
        </div>
        <div>
          <h2 className="text-lg font-semibold">{t.setup.scanTitle}</h2>
          <p className="text-sm text-muted">{t.setup.scanDescription}</p>
        </div>
      </div>

      <GlassCard variant="flat" className="p-6">
        <div className="bg-white rounded-2xl p-4">
          <QRCodeSVG value={provisioningUri} size={200} level="M" marginSize={0} />
        </div>
      </GlassCard>

      {!showManual ? (
        <button type="button" onClick={() => setShowManual(true)} className="text-sm text-[var(--accent)] hover:underline cursor-pointer">
          {t.setup.cantScan}
        </button>
      ) : (
        <div className="w-full max-w-sm">
          <div className="glass-card-flat p-4 flex items-center gap-2">
            <code className="text-xs text-muted break-all flex-1 font-mono">{provisioningUri}</code>
            <button type="button" onClick={handleCopyUri} className="text-[var(--accent)] hover:text-foreground transition-colors p-1 cursor-pointer flex-shrink-0" aria-label="Copy">
              <Copy size={16} />
            </button>
          </div>
        </div>
      )}

      <GlassButton variant="primary" size="lg" onPress={onNext} className="w-full max-w-sm">
        {t.setup.scanned}
      </GlassButton>
    </div>
  );
}
