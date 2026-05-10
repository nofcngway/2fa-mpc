"use client";

import { useRouter, usePathname } from "next/navigation";
import { Logo } from "@/components/ui/logo";
import { ThemeToggle } from "@/components/ui/theme-toggle";
import { LocaleToggle } from "@/components/ui/locale-toggle";
import { GlassButton } from "@/components/ui/glass-button";
import { useAuth } from "@/hooks/use-auth";
import { useTranslations } from "@/lib/i18n";
import { LogOut, Activity, LayoutDashboard } from "lucide-react";

export function Navbar() {
  const router = useRouter();
  const pathname = usePathname();
  const { user, logout } = useAuth();
  const t = useTranslations();

  const isDashboard = pathname?.startsWith("/dashboard") ?? false;
  const isMonitoring = pathname?.startsWith("/monitoring") ?? false;

  return (
    <header className="absolute top-0 left-0 right-0 z-50">
      <div className="flex items-center justify-between px-6 py-4">
        <div className="flex items-center gap-6">
          <Logo size="sm" />
          {user && (
            <nav className="flex items-center gap-1">
              <GlassButton
                variant={isDashboard ? "secondary" : "ghost"}
                size="sm"
                onPress={() => router.push("/dashboard")}
                icon={<LayoutDashboard size={14} />}
              >
                <span className="hidden sm:inline">{t.dashboard.title}</span>
              </GlassButton>
              <GlassButton
                variant={isMonitoring ? "secondary" : "ghost"}
                size="sm"
                onPress={() => router.push("/monitoring")}
                icon={<Activity size={14} />}
              >
                <span className="hidden sm:inline">{t.navMonitoring}</span>
              </GlassButton>
            </nav>
          )}
        </div>

        <div className="flex items-center gap-3">
          {user && (
            <span className="text-sm text-muted hidden md:block">
              {user.email}
            </span>
          )}
          <LocaleToggle size="sm" />
          <ThemeToggle size="sm" />
          <GlassButton
            variant="ghost"
            size="sm"
            onPress={logout}
            icon={<LogOut size={16} />}
            aria-label={t.navbar.logout}
          >
            <span className="hidden sm:inline">{t.navbar.logout}</span>
          </GlassButton>
        </div>
      </div>
    </header>
  );
}
