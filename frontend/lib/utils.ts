import type { Translations } from "@/lib/i18n/ru";

export function copyToClipboard(text: string): Promise<void> {
  return navigator.clipboard.writeText(text);
}

export function downloadAsFile(content: string, filename: string): void {
  const blob = new Blob([content], { type: "text/plain" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

export function formatDate(dateStr: string, locale: string = "ru"): string {
  try {
    return new Intl.DateTimeFormat(locale === "ru" ? "ru-RU" : "en-US", {
      year: "numeric",
      month: "long",
      day: "numeric",
    }).format(new Date(dateStr));
  } catch {
    return dateStr;
  }
}

// --- Validation ---

const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function validateEmail(email: string, t: Translations): string | null {
  if (!email.trim()) return t.validation.emailRequired;
  if (!EMAIL_REGEX.test(email)) return t.validation.emailInvalid;
  return null;
}

const SEQUENCES = "abcdefghijklmnopqrstuvwxyz0123456789qwertyuiopasdfghjklzxcvbnm";

function hasSequence(password: string): boolean {
  const lower = password.toLowerCase();
  for (let i = 0; i <= lower.length - 4; i++) {
    const chunk = lower.slice(i, i + 4);
    if (SEQUENCES.includes(chunk)) return true;
    const reversed = chunk.split("").reverse().join("");
    if (SEQUENCES.includes(reversed)) return true;
  }
  return false;
}

export interface PasswordCheck {
  minLength: boolean;
  hasLowercase: boolean;
  hasUppercase: boolean;
  hasDigit: boolean;
  hasSpecial: boolean;
  noSequences: boolean;
}

export function checkPassword(password: string): PasswordCheck {
  return {
    minLength: password.length >= 12,
    hasLowercase: /[a-z]/.test(password),
    hasUppercase: /[A-Z]/.test(password),
    hasDigit: /[0-9]/.test(password),
    hasSpecial: /[^a-zA-Z0-9]/.test(password),
    noSequences: !hasSequence(password),
  };
}

export function isPasswordValid(password: string): boolean {
  const checks = checkPassword(password);
  return Object.values(checks).every(Boolean);
}

export function validatePassword(password: string, t: Translations): string | null {
  if (!password) return t.validation.passwordRequired;
  if (!isPasswordValid(password)) return t.validation.passwordInvalid;
  return null;
}

export function getPasswordStrength(password: string): number {
  if (!password) return 0;
  const checks = checkPassword(password);
  const passed = Object.values(checks).filter(Boolean).length;
  return Math.round((passed / 6) * 100);
}

export function mapApiErrorCode(code: number, t: Translations): string {
  switch (code) {
    case 3: return t.apiErrors.invalidInput;
    case 5: return t.apiErrors.notFound;
    case 6: return t.apiErrors.alreadyExists;
    case 9: return t.apiErrors.preconditionFailed;
    case 16: return t.apiErrors.unauthenticated;
    default: return t.apiErrors.generic;
  }
}
