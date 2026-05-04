"use client";

import { Suspense, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useAuthStore } from "@/lib/auth/store";

function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const redirect = searchParams.get("redirect") || "/dashboard";

  const { login, register } = useAuthStore();

  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      if (mode === "login") {
        await login(email, password);
      } else {
        await register(email, password);
      }
      router.push(redirect);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="w-full max-w-sm">
      <div className="bg-zinc-900/50 border border-zinc-800 rounded-xl p-8">
        <h1 className="text-xl font-semibold text-white mb-1">
          {mode === "login" ? "Sign in" : "Create account"}
        </h1>
        <p className="text-sm text-zinc-400 mb-6">
          {mode === "login"
            ? "Sign in to your VibeCloud account"
            : "Get started with VibeCloud"}
        </p>

        {error && (
          <div className="mb-4 px-3 py-2 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label
              htmlFor="email"
              className="block text-sm font-medium text-zinc-300 mb-1.5"
            >
              Email
            </label>
            <input
              id="email"
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.com"
              className="w-full px-3 py-2 rounded-lg bg-zinc-800 border border-zinc-700 text-white placeholder-zinc-500 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:border-indigo-500 transition-colors"
            />
          </div>

          <div>
            <label
              htmlFor="password"
              className="block text-sm font-medium text-zinc-300 mb-1.5"
            >
              Password
            </label>
            <input
              id="password"
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Enter your password"
              className="w-full px-3 py-2 rounded-lg bg-zinc-800 border border-zinc-700 text-white placeholder-zinc-500 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:border-indigo-500 transition-colors"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-2 px-4 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading
              ? "Loading..."
              : mode === "login"
                ? "Sign in"
                : "Create account"}
          </button>
        </form>

        <div className="mt-6 text-center">
          <button
            type="button"
            onClick={() => {
              setMode(mode === "login" ? "register" : "login");
              setError(null);
            }}
            className="text-sm text-zinc-400 hover:text-white transition-colors"
          >
            {mode === "login"
              ? "Don't have an account? Create one"
              : "Already have an account? Sign in"}
          </button>
        </div>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return (
    <div className="flex-1 flex items-center justify-center px-4 pt-14">
      <Suspense
        fallback={
          <div className="w-full max-w-sm">
            <div className="bg-zinc-900/50 border border-zinc-800 rounded-xl p-8 animate-pulse">
              <div className="h-6 bg-zinc-800 rounded w-24 mb-4" />
              <div className="h-4 bg-zinc-800 rounded w-48 mb-6" />
              <div className="space-y-4">
                <div className="h-10 bg-zinc-800 rounded" />
                <div className="h-10 bg-zinc-800 rounded" />
                <div className="h-10 bg-zinc-800 rounded" />
              </div>
            </div>
          </div>
        }
      >
        <LoginForm />
      </Suspense>
    </div>
  );
}
