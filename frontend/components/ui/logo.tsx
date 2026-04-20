import { Shield } from "lucide-react";

interface LogoProps {
  size?: "sm" | "md" | "lg";
  withText?: boolean;
}

const sizeConfig: Record<string, { icon: number; text: string }> = {
  sm: { icon: 20, text: "text-base" },
  md: { icon: 24, text: "text-lg" },
  lg: { icon: 32, text: "text-2xl" },
};

export function Logo({ size = "md", withText = true }: LogoProps) {
  const cfg = sizeConfig[size];

  return (
    <div className="flex items-center gap-2">
      <Shield size={cfg.icon} className="text-[var(--accent)]" />
      {withText && (
        <span className={`font-semibold ${cfg.text} tracking-tight`}>
          MPC-2FA
        </span>
      )}
    </div>
  );
}
