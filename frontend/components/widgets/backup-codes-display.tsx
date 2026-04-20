"use client";

import { GlassCard } from "@/components/ui/glass-card";
import { GlassButton } from "@/components/ui/glass-button";
import { copyToClipboard, downloadAsFile } from "@/lib/utils";
import { useTranslations } from "@/lib/i18n";
import { toast } from "@heroui/react";
import { Copy, Download, AlertTriangle } from "lucide-react";

interface BackupCodesDisplayProps {
  codes: string[];
  onDone: () => void;
}

export function BackupCodesDisplay({ codes, onDone }: BackupCodesDisplayProps) {
  const t = useTranslations();

  const handleCopy = async () => {
    try {
      await copyToClipboard(codes.join("\n"));
      toast(t.setup.backupCodesCopied, { variant: "success" });
    } catch {
      toast(t.common.copyFailed, { variant: "danger" });
    }
  };

  const handleDownload = () => {
    const content = [
      t.setup.backupFileTitle,
      "====================",
      "",
      t.setup.backupFileWarning,
      t.setup.backupFileNote,
      "",
      ...codes.map((code, i) => `${i + 1}. ${code}`),
      "",
      `Generated: ${new Date().toISOString()}`,
    ].join("\n");
    downloadAsFile(content, t.setup.backupFileName);
  };

  return (
    <div className="flex flex-col gap-6 items-center">
      <GlassCard variant="flat" className="p-4 w-full max-w-sm">
        <div className="flex items-start gap-3">
          <AlertTriangle size={20} className="text-[var(--glass-warning)] flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-sm font-medium">{t.setup.saveBackupCodes}</p>
            <p className="text-xs text-muted mt-1">{t.setup.backupWarning}</p>
          </div>
        </div>
      </GlassCard>

      <GlassCard className="p-6 w-full max-w-sm">
        <div className="grid grid-cols-2 gap-2">
          {codes.map((code, i) => (
            <div key={i} className="flex items-center gap-2 px-3 py-2 rounded-xl bg-[var(--glass-bg-elevated)] border border-[var(--glass-border-subtle)]">
              <span className="text-xs text-muted w-4">{i + 1}.</span>
              <code className="text-sm font-mono font-medium">{code}</code>
            </div>
          ))}
        </div>
      </GlassCard>

      <div className="flex gap-3 w-full max-w-sm">
        <GlassButton variant="secondary" size="md" onPress={handleCopy} icon={<Copy size={16} />} className="flex-1">
          {t.setup.copyAll}
        </GlassButton>
        <GlassButton variant="secondary" size="md" onPress={handleDownload} icon={<Download size={16} />} className="flex-1">
          {t.setup.download}
        </GlassButton>
      </div>

      <GlassButton variant="primary" size="lg" onPress={onDone} className="w-full max-w-sm">
        {t.setup.savedCodes}
      </GlassButton>
    </div>
  );
}
