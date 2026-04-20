import { cookies } from "next/headers";
import { NextResponse } from "next/server";

const COOKIE_NAME = "refresh_token";
const MAX_AGE = 7 * 24 * 60 * 60; // 7 days

export async function POST(request: Request) {
  const body = await request.json();
  const { refreshToken } = body;

  if (!refreshToken || typeof refreshToken !== "string") {
    return NextResponse.json({ error: "Missing refresh token" }, { status: 400 });
  }

  const cookieStore = await cookies();
  cookieStore.set(COOKIE_NAME, refreshToken, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: MAX_AGE,
  });

  return NextResponse.json({ ok: true });
}

export async function GET() {
  const cookieStore = await cookies();
  const token = cookieStore.get(COOKIE_NAME);

  if (!token?.value) {
    return NextResponse.json({ refreshToken: null }, { status: 401 });
  }

  return NextResponse.json({ refreshToken: token.value });
}

export async function DELETE() {
  const cookieStore = await cookies();
  cookieStore.delete(COOKIE_NAME);

  return NextResponse.json({ ok: true });
}
