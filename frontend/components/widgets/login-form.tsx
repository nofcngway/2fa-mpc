"use client";

import { useState } from "react";
import Link from "next/link";
import { GlassCard } from "@/components/ui/glass-card";
import { GlassInput } from "@/components/ui/glass-input";
import { PasswordInput } from "@/components/ui/password-input";
import { GlassButton } from "@/components/ui/glass-button";
import { useAuth } from "@/hooks/use-auth";
import { ApiRequestError } from "@/lib/api";
import { validateEmail, mapApiErrorMessage } from "@/lib/utils";

export function LoginForm() {
  const { login } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [emailError, setEmailError] = useState<string | null>(null);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [serverError, setServerError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setServerError(null);

    const eErr = validateEmail(email);
    const pErr = !password ? "Password is required" : null;
    setEmailError(eErr);
    setPasswordError(pErr);

    if (eErr || pErr) return;

    setIsSubmitting(true);
    try {
      await login(email, password);
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
          <h1 className="text-2xl font-semibold">Welcome back</h1>
          <p className="text-sm text-muted mt-1">
            Sign in to your account
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

        <PasswordInput
          value={password}
          onChange={(v) => {
            setPassword(v);
            setPasswordError(null);
          }}
          error={passwordError ?? undefined}
          isDisabled={isSubmitting}
          autoComplete="current-password"
        />

        <GlassButton
          type="submit"
          variant="primary"
          size="lg"
          isLoading={isSubmitting}
          className="w-full mt-1"
        >
          Sign in
        </GlassButton>

        <p className="text-center text-sm text-muted">
          Don&apos;t have an account?{" "}
          <Link
            href="/register"
            className="text-[var(--accent)] hover:underline font-medium"
          >
            Create one
          </Link>
        </p>
      </form>
    </GlassCard>
  );
}
