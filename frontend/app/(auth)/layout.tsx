import { Logo } from "@/components/ui/logo";
import { ThemeToggle } from "@/components/ui/theme-toggle";
import { LocaleToggle } from "@/components/ui/locale-toggle";

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="auth-background relative">
      <div className="absolute top-0 left-0 right-0 flex items-center justify-between px-6 py-4">
        <Logo size="sm" />
        <div className="flex items-center gap-2">
          <LocaleToggle size="sm" />
          <ThemeToggle size="sm" />
        </div>
      </div>

      <main className="min-h-screen flex items-center justify-center px-4">
        <div className="w-full max-w-md">{children}</div>
      </main>
    </div>
  );
}
