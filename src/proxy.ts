import { NextRequest, NextResponse } from "next/server";

const protectedRoutes = ["/dashboard"];
const authRoutes = ["/login"];
const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
const IS_DEV = process.env.NODE_ENV === "development";

function isTokenExpired(token: string): boolean {
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    return payload.exp * 1000 < Date.now();
  } catch {
    return true;
  }
}

export async function proxy(request: NextRequest): Promise<NextResponse> {
  const { pathname } = request.nextUrl;
  const response = NextResponse.next();

  const accessToken = request.cookies.get("access_token")?.value;
  const refreshToken = request.cookies.get("refresh_token")?.value;

  let isAuthenticated = !!accessToken && !isTokenExpired(accessToken);

  // Proactive refresh
  if (!isAuthenticated && refreshToken) {
    try {
      const goRes = await fetch(`${API_URL}/api/v1/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });
      if (goRes.ok) {
        const json = await goRes.json();
        const data = json.data ?? json;
        response.cookies.set("access_token", data.access_token, {
          httpOnly: true,
          secure: !IS_DEV,
          sameSite: "lax",
          path: "/",
          maxAge: data.expires_in,
        });
        if (data.refresh_token) {
          response.cookies.set("refresh_token", data.refresh_token, {
            httpOnly: true,
            secure: !IS_DEV,
            sameSite: "lax",
            path: "/",
            maxAge: 7 * 24 * 3600,
          });
        }
        isAuthenticated = true;
      }
    } catch {
      // Refresh failed, continue unauthenticated
    }
  }

  // Protect dashboard
  if (protectedRoutes.some((route) => pathname.startsWith(route))) {
    if (!isAuthenticated) {
      const loginUrl = new URL("/login", request.url);
      loginUrl.searchParams.set("redirect", pathname);
      return NextResponse.redirect(loginUrl);
    }
  }

  // Redirect logged-in users away from login
  if (authRoutes.some((route) => pathname === route)) {
    if (isAuthenticated) {
      return NextResponse.redirect(new URL("/dashboard", request.url));
    }
  }

  return response;
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|icon.svg|api/).*)"],
};
