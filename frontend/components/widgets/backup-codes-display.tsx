"use client";

import { GlassCard } from "@/components/ui/glass-card";
import { GlassButton } from "@/components/ui/glass-button";
import { copyToClipboard, downloadAsFile } from "@/lib/utils";
import { toast } from "@heroui/react";
import { Copy, Download, AlertTriangle } from "lucide-react";

interface BackupCodesDisplayProps {
  codes: string[];
  onDone: () => void;
}

export function BackupCodesDisplay({ codes, onDone }: BackupCodesDisplayProps) {
  const handleCopy = async () => {
    try {
      await copyToClipboard(codes.join("\n"));
      toast("Backup codes copied", { variant: "success" });
    } catch {
      toast("Failed to copy", { variant: "danger" });
    }
  };

  const handleDownload = () => {
    const content = [
      "MPC-2FA Backup Codes",
      "====================",
      "",
      "Keep these codes in a safe place.",
      "Each code can only be used once.",
      "",
      ...codes.map((code, i) => `${i + 1}. ${code}`),
      "",
      `Generated: ${new Date().toISOString()}`,
    ].join("\n");
    downloadAsFile(content, "mpc-2fa-backup-codes.txt");
  };

  return (
    <div className="flex flex-col gap-6 items-center">
      {/* Warning */}
      <GlassCard variant="flat" className="p-4 w-full max-w-sm">
        <div className="flex items-start gap-3">
          <AlertTriangle size={20} className="text-[var(--glass-warning)] flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-sm font-medium">Save your backup codes</p>
            <p className="text-xs text-muted mt-1">
              These codes won&apos;t be shown again. Store them in a safe place.
              Each code can only be used once.
            </p>
          </div>
        </div>
      </GlassCard>

      {/* Codes grid */}
      <GlassCard className="p-6 w-full max-w-sm">
        <div className="grid grid-cols-2 gap-2">
          {codes.map((code, i) => (
            <div
              key={i}
              className="flex items-center gap-2 px-3 py-2 rounded-xl bg-[var(--glass-bg-elevated)] border border-[var(--glass-border-subtle)]"
            >
              <span className="text-xs text-muted w-4">{i + 1}.</span>
              <code className="text-sm font-mono font-medium">{code}</code>
            </div>
          ))}
        </div>
      </GlassCard>

      {/* Actions */}
      <div className="flex gap-3 w-full max-w-sm">
        <GlassButton
          variant="secondary"
          size="md"
          onPress={handleCopy}
          icon={<Copy size={16} />}
          className="flex-1"
        >
          Copy all
        </GlassButton>
        <GlassButton
          variant="secondary"
          size="md"
          onPress={handleDownload}
          icon={<Download size={16} />}
          className="flex-1"
        >
          Download
        </GlassButton>
      </div>

      <GlassButton
        variant="primary"
        size="lg"
        onPress={onDone}
        className="w-full max-w-sm"
      >
        I&apos;ve saved my codes
      </GlassButton>
    </div>
  );
}
