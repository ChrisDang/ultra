import { NextResponse } from "next/server";
import { cookies } from "next/headers";

export async function GET(): Promise<NextResponse> {
  const cookieStore = await cookies();
  const token = cookieStore.get("access_token")?.value ?? null;
  return NextResponse.json({ access_token: token });
}
