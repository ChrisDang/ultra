import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { Nav } from "@/components/nav";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "VibeCloud — Deploy your vibe-coded apps in minutes",
  description:
    "One CLI to deploy your AI-built apps. VibeCloud wraps Vercel, Supabase, and Expo so you can go live in minutes, not hours.",
  openGraph: {
    title: "VibeCloud — Deploy your vibe-coded apps in minutes",
    description:
      "One CLI to deploy your AI-built apps. VibeCloud wraps Vercel, Supabase, and Expo so you can go live in minutes, not hours.",
    type: "website",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col">
        <Nav />
        {children}
      </body>
    </html>
  );
}
