"use client";

import { useState } from "react";
import { api } from "@/lib/auth/api";
import { useAuthStore } from "@/lib/auth/store";

export function TierToggle({ currentTier }: { currentTier: string }) {
  const [tier, setTier] = useState(currentTier);
  const [loading, setLoading] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const initialize = useAuthStore((s) => s.initialize);

  const isPremium = tier === "premium";
  const targetTier = isPremium ? "free" : "premium";

  async function toggleTier() {
    setLoading(true);
    try {
      const res = await api.patch("/api/v1/tier", { tier: targetTier });
      const user = res.data?.data ?? res.data;
      setTier(user.tier);
      setShowConfirm(false);
    } catch {
      // Failed
    } finally {
      setLoading(false);
    }
  }

  function handleClick() {
    if (targetTier === "premium") {
      setShowConfirm(true);
    } else {
      toggleTier();
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between p-4 rounded-xl border border-zinc-800 bg-zinc-900/30">
        <div>
          <p className="font-medium text-white">
            {isPremium ? "Premium" : "Free"}
          </p>
          <p className="text-sm text-zinc-500">
            {isPremium
              ? "Unlimited deploys, production deployments, safety rails"
              : "15 deploys/month, preview deployments only"}
          </p>
        </div>
        <button
          onClick={handleClick}
          disabled={loading}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors disabled:opacity-50 ${
            isPremium
              ? "bg-zinc-800 hover:bg-zinc-700 text-zinc-300"
              : "bg-indigo-600 hover:bg-indigo-500 text-white"
          }`}
        >
          {loading
            ? "..."
            : isPremium
              ? "Downgrade to Free"
              : "Upgrade to Premium"}
        </button>
      </div>

      {showConfirm && (
        <div className="mt-4 p-4 rounded-xl border border-amber-500/20 bg-amber-950/20">
          <p className="text-sm text-amber-200 mb-3">
            Premium is in alpha preview. When premium launches with billing,
            your tier will revert to free. Your snapshots and configuration
            will be preserved.
          </p>
          <div className="flex gap-3">
            <button
              onClick={toggleTier}
              disabled={loading}
              className="px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-sm font-medium text-white transition-colors disabled:opacity-50"
            >
              {loading ? "Upgrading..." : "Enable Premium Alpha"}
            </button>
            <button
              onClick={() => setShowConfirm(false)}
              className="px-4 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-sm font-medium text-zinc-300 transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
