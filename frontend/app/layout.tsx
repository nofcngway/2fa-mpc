import type { Metadata } from "next";
import { Onest, Golos_Text, JetBrains_Mono } from "next/font/google";
import { Providers } from "@/app/providers";
import "./globals.css";

const onest = Onest({
  subsets: ["cyrillic", "latin"],
  variable: "--font-onest",
  display: "swap",
});

const golosText = Golos_Text({
  subsets: ["cyrillic", "latin"],
  variable: "--font-golos",
  display: "swap",
});

const jetbrainsMono = JetBrains_Mono({
  subsets: ["cyrillic", "latin"],
  variable: "--font-jetbrains",
  display: "swap",
});

export const metadata: Metadata = {
  title: "MPC-2FA",
  description: "Two-factor authentication with distributed secret storage",
  icons: {
    icon: "/favicon.svg",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body
        className={`${onest.variable} ${golosText.variable} ${jetbrainsMono.variable} bg-background text-foreground antialiased`}
      >
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
