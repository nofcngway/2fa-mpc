"use client";

import { useState } from "react";
import { GlassCard } from "@/components/ui/glass-card";
import { GlassButton } from "@/components/ui/glass-button";
import { useAuth } from "@/hooks/use-auth";
import { toast } from "@heroui/react";
import { User, LogOut } from "lucide-react";

export function UserInfoCard() {
  const { user, logoutAll } = useAuth();
  const [isLoggingOut, setIsLoggingOut] = useState(false);

  const handleLogoutAll = async () => {
    setIsLoggingOut(true);
    try {
      await logoutAll();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Failed to logout", {
        variant: "danger",
      });
    } finally {
      setIsLoggingOut(false);
    }
  };

  if (!user) return null;

  return (
    <GlassCard className="p-6">
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-4">
          <div className="w-12 h-12 rounded-2xl bg-[var(--accent-subtle)] flex items-center justify-center">
            <User size={24} className="text-[var(--accent)]" />
          </div>
          <div>
            <h2 className="text-lg font-semibold">Account</h2>
            <p className="text-sm text-muted">{user.email}</p>
          </div>
        </div>

        <GlassButton
          variant="ghost"
          size="sm"
          isLoading={isLoggingOut}
          onPress={handleLogoutAll}
          icon={<LogOut size={14} />}
        >
          Logout all devices
        </GlassButton>
      </div>
    </GlassCard>
  );
}
