"use client";

import { checkPassword, getPasswordStrength } from "@/lib/utils";
import { Check, X } from "lucide-react";

interface PasswordStrengthProps {
  password: string;
}

const requirements = [
  { key: "minLength" as const, label: "At least 12 characters" },
  { key: "hasLowercase" as const, label: "One lowercase letter" },
  { key: "hasUppercase" as const, label: "One uppercase letter" },
  { key: "hasDigit" as const, label: "One digit" },
  { key: "hasSpecial" as const, label: "One special character" },
  { key: "noSequences" as const, label: "No 4+ character sequences" },
];

function getStrengthColor(strength: number): string {
  if (strength < 35) return "bg-[var(--glass-danger)]";
  if (strength < 65) return "bg-[var(--glass-warning)]";
  if (strength < 100) return "bg-[var(--accent)]";
  return "bg-[var(--glass-success)]";
}

function getStrengthLabel(strength: number): string {
  if (strength < 35) return "Weak";
  if (strength < 65) return "Fair";
  if (strength < 100) return "Good";
  return "Strong";
}

export function PasswordStrength({ password }: PasswordStrengthProps) {
  if (!password) return null;

  const checks = checkPassword(password);
  const strength = getPasswordStrength(password);

  return (
    <div className="flex flex-col gap-2 px-1">
      {/* Strength bar */}
      <div className="flex items-center gap-2">
        <div className="flex-1 h-1.5 rounded-full bg-[var(--glass-border-subtle)] overflow-hidden">
          <div
            className={`h-full rounded-full transition-all duration-300 ${getStrengthColor(strength)}`}
            style={{ width: `${strength}%` }}
          />
        </div>
        <span className="text-xs text-muted font-medium min-w-[40px]">
          {getStrengthLabel(strength)}
        </span>
      </div>

      {/* Requirements checklist */}
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
