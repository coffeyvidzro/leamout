import { type NextRequest, NextResponse } from "next/server";

const SESSION_COOKIE_NAME = "lmt-session";

const protectedRoutes = [
  "/dashboard",
  "/customers",
  "/products",
  "/subscriptions",
  "/credits",
  "/dunning",
];

function isProtectedRoute(pathname: string) {
  return protectedRoutes.some(
    (route) => pathname === route || pathname.startsWith(`${route}/`),
  );
}

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;

  const hasSession = request.cookies.has(SESSION_COOKIE_NAME);

  if (isProtectedRoute(pathname) && !hasSession) {
    const loginUrl = new URL("/login", request.url);
    loginUrl.searchParams.set("next", pathname);

    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    "/dashboard/:path*",
    "/customers/:path*",
    "/products/:path*",
    "/subscriptions/:path*",
    "/credits/:path*",
    "/dunning/:path*",
  ],
};
