"use client";

import { useEffect } from "react";
import { useAuthStore } from "@/lib/auth/store";
import { DeviceCode } from "./device-code";
import { TierToggle } from "./tier-toggle";

export default function DashboardPage() {
  const { user, isLoading, initialize } = useAuthStore();

  useEffect(() => {
    initialize();
  }, [initialize]);

  if (isLoading) {
    return (
      <div className="min-h-screen bg-[#09090b] flex items-center justify-center">
        <p className="text-zinc-500">Loading...</p>
      </div>
    );
  }

  if (!user) return null;

  return (
    <div className="min-h-screen bg-[#09090b] text-white">
      <div className="max-w-3xl mx-auto px-6 py-24">
        <h1 className="text-3xl font-bold mb-2">Dashboard</h1>
        <p className="text-zinc-400 mb-8">
          {user.email}{" "}
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-zinc-800 text-zinc-300 ml-1">
            {user.tier}
          </span>
        </p>

        <section className="mb-12">
          <h2 className="text-xl font-semibold mb-4">CLI Authentication</h2>
          <p className="text-zinc-400 text-sm mb-6">
            Generate a code to authenticate the VibeCloud CLI. Run{" "}
            <code className="text-zinc-300 bg-zinc-800 px-1.5 py-0.5 rounded text-xs font-mono">
              vibecloud auth login
            </code>{" "}
            and enter the code when prompted.
          </p>
          <DeviceCode />
        </section>

        <section>
          <h2 className="text-xl font-semibold mb-4">Plan</h2>
          <TierToggle currentTier={user.tier} />
        </section>
      </div>
    </div>
  );
}
