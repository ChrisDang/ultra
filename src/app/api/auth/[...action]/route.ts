import { NextRequest, NextResponse } from "next/server";
import { cookies } from "next/headers";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
const IS_PROD = process.env.NODE_ENV === "production";

function cookieOpts(maxAge: number) {
  return {
    httpOnly: true,
    secure: IS_PROD,
    sameSite: "lax" as const,
    path: "/",
    maxAge,
  };
}

async function proxyAuth(
  action: string,
  body: unknown
): Promise<NextResponse> {
  if (action === "logout") {
    const res = NextResponse.json({ success: true });
    res.cookies.delete("access_token");
    res.cookies.delete("refresh_token");
    return res;
  }

  const backendRes = await fetch(`${API_URL}/api/v1/auth/${action}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });

  const data = await backendRes.json();

  if (!backendRes.ok) {
    return NextResponse.json(data, { status: backendRes.status });
  }

  const tokenData = data.data ?? data;
  const { access_token, refresh_token, expires_in } = tokenData;

  const res = NextResponse.json({ access_token, expires_in });

  if (access_token) {
    res.cookies.set("access_token", access_token, cookieOpts(expires_in ?? 900));
  }
  if (refresh_token) {
    res.cookies.set(
      "refresh_token",
      refresh_token,
      cookieOpts(7 * 24 * 3600)
    );
  }

  return res;
}

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ action: string[] }> }
): Promise<NextResponse> {
  const { action } = await params;
  const actionName = action[0];

  if (!["login", "register", "refresh", "logout"].includes(actionName)) {
    return NextResponse.json({ error: "Not found" }, { status: 404 });
  }

  let body: unknown = {};
  if (actionName !== "logout") {
    try {
      body = await request.json();
    } catch {
      return NextResponse.json(
        { error: { message: "Invalid JSON" } },
        { status: 400 }
      );
    }
  }

  // For refresh: inject httpOnly cookie if body doesn't have refresh_token
  if (actionName === "refresh") {
    const cookieStore = await cookies();
    const cookieToken = cookieStore.get("refresh_token")?.value;
    if (cookieToken && !(body as Record<string, unknown>).refresh_token) {
      body = { ...(body as object), refresh_token: cookieToken };
    }
  }

  return proxyAuth(actionName, body);
}
