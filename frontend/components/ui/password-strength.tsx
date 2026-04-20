"use client";

import { checkPassword, getPasswordStrength } from "@/lib/utils";
import { useTranslations } from "@/lib/i18n";
import { Check, X } from "lucide-react";

interface PasswordStrengthProps {
  password: string;
}

function getStrengthColor(strength: number): string {
  if (strength < 35) return "bg-[var(--glass-danger)]";
  if (strength < 65) return "bg-[var(--glass-warning)]";
  if (strength < 100) return "bg-[var(--accent)]";
  return "bg-[var(--glass-success)]";
}

export function PasswordStrength({ password }: PasswordStrengthProps) {
  const t = useTranslations();

  if (!password) return null;

  const checks = checkPassword(password);
  const strength = getPasswordStrength(password);

  const strengthLabel =
    strength < 35 ? t.passwordStrength.weak
    : strength < 65 ? t.passwordStrength.fair
    : strength < 100 ? t.passwordStrength.good
    : t.passwordStrength.strong;

  const requirements = [
    { key: "minLength" as const, label: t.passwordStrength.minLength },
    { key: "hasLowercase" as const, label: t.passwordStrength.hasLowercase },
    { key: "hasUppercase" as const, label: t.passwordStrength.hasUppercase },
    { key: "hasDigit" as const, label: t.passwordStrength.hasDigit },
    { key: "hasSpecial" as const, label: t.passwordStrength.hasSpecial },
    { key: "noSequences" as const, label: t.passwordStrength.noSequences },
  ];

  return (
    <div className="flex flex-col gap-2 px-1">
      <div className="flex items-center gap-2">
        <div className="flex-1 h-1.5 rounded-full bg-[var(--glass-border-subtle)] overflow-hidden">
          <div
            className={`h-full rounded-full transition-all duration-300 ${getStrengthColor(strength)}`}
            style={{ width: `${strength}%` }}
          />
        </div>
        <span className="text-xs text-muted font-medium min-w-[55px]">{strengthLabel}</span>
      </div>

      <ul className="flex flex-col gap-0.5">
        {requirements.map((req) => {
          const passed = checks[req.key];
          return (
            <li
              key={req.key}
              className={`flex items-center gap-1.5 text-xs transition-colors duration-200 ${
                passed ? "text-[var(--glass-success)]" : "text-muted"
              }`}
            >
              {passed ? <Check size={12} /> : <X size={12} />}
              {req.label}
            </li>
          );
        })}
      </ul>
    </div>
  );
}
