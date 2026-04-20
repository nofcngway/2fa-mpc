"use client";

import { useState } from "react";
import Link from "next/link";
import { GlassCard } from "@/components/ui/glass-card";
import { GlassInput } from "@/components/ui/glass-input";
import { PasswordInput } from "@/components/ui/password-input";
import { PasswordStrength } from "@/components/ui/password-strength";
import { GlassButton } from "@/components/ui/glass-button";
import { useAuth } from "@/hooks/use-auth";
import { ApiRequestError } from "@/lib/api";
import { validateEmail, validatePassword, mapApiErrorCode } from "@/lib/utils";
import { useTranslations } from "@/lib/i18n";

export function RegisterForm() {
  const { register } = useAuth();
  const t = useTranslations();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [emailError, setEmailError] = useState<string | null>(null);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [confirmError, setConfirmError] = useState<string | null>(null);
  const [serverError, setServerError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setServerError(null);

    const eErr = validateEmail(email, t);
    const pErr = validatePassword(password, t);
    const cErr = !confirmPassword
      ? t.validation.confirmRequired
      : password !== confirmPassword
        ? t.validation.confirmMismatch
        : null;

    setEmailError(eErr);
    setPasswordError(pErr);
    setConfirmError(cErr);

    if (eErr || pErr || cErr) return;

    setIsSubmitting(true);
    try {
      await register(email, password);
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setServerError(mapApiErrorCode(err.code, t));
      } else {
        setServerError(t.apiErrors.generic);
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <GlassCard variant="elevated" className="p-8">
      <form onSubmit={handleSubmit} className="flex flex-col gap-5">
        <div className="text-center mb-2">
          <h1 className="text-2xl font-semibold">{t.auth.createAccount}</h1>
          <p className="text-sm text-muted mt-1">{t.auth.createAccountSubtitle}</p>
        </div>

        {serverError && (
          <div className="bg-[var(--glass-danger)]/10 border border-[var(--glass-danger)]/20 rounded-xl px-4 py-3 text-sm text-[var(--glass-danger)]">
            {serverError}
          </div>
        )}

        <GlassInput
          label={t.auth.email}
          type="email"
          placeholder={t.auth.emailPlaceholder}
          value={email}
          onChange={(v) => { setEmail(v); setEmailError(null); }}
          error={emailError ?? undefined}
          isDisabled={isSubmitting}
          autoComplete="email"
        />

        <div className="flex flex-col gap-2">
          <PasswordInput
            label={t.auth.password}
            placeholder={t.auth.passwordPlaceholder}
            value={password}
            onChange={(v) => { setPassword(v); setPasswordError(null); }}
            error={passwordError ?? undefined}
            isDisabled={isSubmitting}
            autoComplete="new-password"
          />
          <PasswordStrength password={password} />
        </div>

        <PasswordInput
          label={t.auth.confirmPassword}
          placeholder={t.auth.confirmPasswordPlaceholder}
          value={confirmPassword}
          onChange={(v) => { setConfirmPassword(v); setConfirmError(null); }}
          error={confirmError ?? undefined}
          isDisabled={isSubmitting}
          autoComplete="new-password"
        />

        <GlassButton type="submit" variant="primary" size="lg" isLoading={isSubmitting} className="w-full mt-1">
          {t.auth.createAccount}
        </GlassButton>

        <p className="text-center text-sm text-muted">
          {t.auth.hasAccount}{" "}
          <Link href="/login" className="text-[var(--accent)] hover:underline font-medium">
            {t.auth.signInLink}
          </Link>
        </p>
      </form>
    </GlassCard>
  );
}
