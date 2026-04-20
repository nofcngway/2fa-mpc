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
import {
  validateEmail,
  validatePassword,
  mapApiErrorMessage,
} from "@/lib/utils";

export function RegisterForm() {
  const { register } = useAuth();
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

    const eErr = validateEmail(email);
    const pErr = validatePassword(password);
    const cErr =
      !confirmPassword
        ? "Please confirm your password"
        : password !== confirmPassword
          ? "Passwords do not match"
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
        setServerError(mapApiErrorMessage(err.code, err.message));
      } else {
        setServerError("Something went wrong. Please try again.");
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <GlassCard variant="elevated" className="p-8">
      <form onSubmit={handleSubmit} className="flex flex-col gap-5">
        <div className="text-center mb-2">
          <h1 className="text-2xl font-semibold">Create account</h1>
          <p className="text-sm text-muted mt-1">
            Get started with MPC-2FA
          </p>
        </div>

        {serverError && (
          <div className="bg-[var(--glass-danger)]/10 border border-[var(--glass-danger)]/20 rounded-xl px-4 py-3 text-sm text-[var(--glass-danger)]">
            {serverError}
          </div>
        )}

        <GlassInput
          label="Email"
          type="email"
          placeholder="you@example.com"
          value={email}
          onChange={(v) => {
            setEmail(v as unknown as string);
            setEmailError(null);
          }}
          error={emailError ?? undefined}
          isDisabled={isSubmitting}
          autoComplete="email"
        />

        <div className="flex flex-col gap-2">
          <PasswordInput
            value={password}
            onChange={(v) => {
              setPassword(v);
              setPasswordError(null);
            }}
            error={passwordError ?? undefined}
            isDisabled={isSubmitting}
            autoComplete="new-password"
          />
          <PasswordStrength password={password} />
        </div>

        <PasswordInput
          label="Confirm password"
          placeholder="Repeat your password"
          value={confirmPassword}
          onChange={(v) => {
            setConfirmPassword(v);
            setConfirmError(null);
          }}
          error={confirmError ?? undefined}
          isDisabled={isSubmitting}
          autoComplete="new-password"
        />

        <GlassButton
          type="submit"
          variant="primary"
          size="lg"
          isLoading={isSubmitting}
          className="w-full mt-1"
        >
          Create account
        </GlassButton>

        <p className="text-center text-sm text-muted">
          Already have an account?{" "}
          <Link
            href="/login"
            className="text-[var(--accent)] hover:underline font-medium"
          >
            Sign in
          </Link>
        </p>
      </form>
    </GlassCard>
  );
}
