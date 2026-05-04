import axios from "axios";

// In-memory access token (never in localStorage)
let accessToken: string | null = null;
let refreshPromise: Promise<string | null> | null = null;

async function refreshTokenOnce(): Promise<string | null> {
  if (!refreshPromise) {
    refreshPromise = (async () => {
      try {
        const res = await fetch("/api/auth/refresh", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: "{}",
        });
        if (!res.ok) return null;
        const data = await res.json();
        accessToken = data.access_token ?? null;
        return accessToken;
      } catch {
        return null;
      } finally {
        refreshPromise = null;
      }
    })();
  }
  return refreshPromise;
}

let initPromise: Promise<void> | null = null;

export function initialize(): Promise<void> {
  if (!initPromise) {
    initPromise = (async () => {
      try {
        const res = await fetch("/api/auth/token");
        if (res.ok) {
          const data = await res.json();
          accessToken = data.access_token ?? null;
        }
      } catch {
        accessToken = null;
      }
    })();
  }
  return initPromise;
}

export const api = axios.create({
  baseURL: "",
  headers: { "Content-Type": "application/json" },
});

api.interceptors.request.use(async (config) => {
  await initialize();
  if (accessToken) {
    config.headers.Authorization = `Bearer ${accessToken}`;
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;
      const newToken = await refreshTokenOnce();
      if (newToken) {
        originalRequest.headers.Authorization = `Bearer ${newToken}`;
        return api(originalRequest);
      }
      accessToken = null;
      if (typeof window !== "undefined") {
        window.location.href = "/login";
      }
    }
    return Promise.reject(error);
  }
);

export function getAccessToken(): string | null {
  return accessToken;
}

export function setAccessToken(token: string | null): void {
  accessToken = token;
}

export function clearAccessToken(): void {
  accessToken = null;
  initPromise = null;
}
