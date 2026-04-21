"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Navbar } from "@/components/widgets/navbar";
import { LoadingSpinner } from "@/components/ui/loading-spinner";
import { useAuth } from "@/hooks/use-auth";
import { getPending2FA } from "@/lib/auth";

export default function ProtectedLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { isAuthenticated, isLoading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (isLoading) return;
    if (!isAuthenticated) {
      router.replace("/login");
    } else if (getPending2FA()) {
      router.replace("/2fa/verify");
    }
  }, [isLoading, isAuthenticated, router]);

  if (isLoading) {
    return <LoadingSpinner fullPage label="Loading..." />;
  }

  if (!isAuthenticated) {
    return null;
  }

  return (
    <div className="auth-background relative min-h-screen flex flex-col">
      <Navbar />
      <main className="flex-1 max-w-5xl w-full mx-auto px-4 sm:px-6 pt-20 pb-8">
        {children}
      </main>
    </div>
  );
}
