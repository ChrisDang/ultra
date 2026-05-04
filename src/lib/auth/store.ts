import { create } from "zustand";
import {
  initialize as initApi,
  clearAccessToken,
  getAccessToken,
  setAccessToken,
} from "./api";

interface User {
  id: string;
  email: string;
  tier: string;
}

interface AuthState {
  user: User | null;
  isLoading: boolean;
  initialize: () => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isLoading: true,

  initialize: async () => {
    try {
      await initApi();
      const token = getAccessToken();
      if (!token) {
        set({ user: null, isLoading: false });
        return;
      }
      const payload = JSON.parse(atob(token.split(".")[1]));
      set({
        user: { id: payload.sub, email: payload.email, tier: payload.tier },
        isLoading: false,
      });
    } catch {
      set({ user: null, isLoading: false });
    }
  },

  login: async (email, password) => {
    const res = await fetch("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      throw new Error(data.error?.message || "Login failed");
    }
    const { access_token } = await res.json();
    setAccessToken(access_token);
    const payload = JSON.parse(atob(access_token.split(".")[1]));
    set({
      user: { id: payload.sub, email: payload.email, tier: payload.tier },
    });
  },

  register: async (email, password) => {
    const res = await fetch("/api/auth/register", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      throw new Error(data.error?.message || "Registration failed");
    }
    const { access_token } = await res.json();
    setAccessToken(access_token);
    const payload = JSON.parse(atob(access_token.split(".")[1]));
    set({
      user: { id: payload.sub, email: payload.email, tier: payload.tier },
    });
  },

  signOut: async () => {
    await fetch("/api/auth/logout", { method: "POST" });
    clearAccessToken();
    set({ user: null });
  },
}));
