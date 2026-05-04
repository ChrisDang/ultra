"use client";

import { useState } from "react";
import { api } from "@/lib/auth/api";

export function DeviceCode() {
  const [code, setCode] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function generate() {
    setLoading(true);
    setError("");
    try {
      const res = await api.post("/api/v1/auth/device-code");
      const data = res.data?.data ?? res.data;
      setCode(data.code);
    } catch {
      setError("Failed to generate code. Try again.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div>
      {code && (
        <div className="mb-6 p-6 rounded-xl bg-indigo-950/30 border border-indigo-500/20">
          <p className="text-sm text-indigo-300 mb-3">
            Enter this code in your terminal:
          </p>
          <p className="text-4xl font-mono font-bold tracking-[0.3em] text-white text-center py-4">
            {code}
          </p>
          <p className="text-xs text-zinc-500 text-center">
            Expires in 5 minutes
          </p>
        </div>
      )}

      {error && <p className="text-red-400 text-sm mb-4">{error}</p>}

      <button
        onClick={generate}
        disabled={loading}
        className="px-4 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 text-sm font-medium text-zinc-300 transition-colors disabled:opacity-50"
      >
        {loading ? "Generating..." : code ? "Generate new code" : "Generate CLI code"}
      </button>
    </div>
  );
}
