"use client";

import { Logo } from "@/components/ui/logo";
import { ThemeToggle } from "@/components/ui/theme-toggle";
import { GlassButton } from "@/components/ui/glass-button";
import { useAuth } from "@/hooks/use-auth";
import { LogOut } from "lucide-react";

export function Navbar() {
  const { user, logout } = useAuth();

  return (
    <header className="glass-navbar sticky top-0 z-50">
      <div className="max-w-5xl mx-auto flex items-center justify-between px-6 py-3">
        <Logo size="sm" />

        <div className="flex items-center gap-3">
          {user && (
            <span className="text-sm text-muted hidden sm:block">
              {user.email}
            </span>
          )}
          <ThemeToggle size="sm" />
          <GlassButton
            variant="ghost"
            size="sm"
            onPress={logout}
            icon={<LogOut size={16} />}
            aria-label="Logout"
          >
            <span className="hidden sm:inline">Logout</span>
          </GlassButton>
        </div>
      </div>
    </header>
  );
}
