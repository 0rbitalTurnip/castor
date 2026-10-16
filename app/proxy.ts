import { NextResponse, type NextRequest } from "next/server";
import { mutationOriginAllowed, webAccess, webCredentials } from "./src/lib/web-auth";

export function proxy(request: NextRequest) {
  // Liveness only: never returns configuration, devices, or upstream responses.
  if (request.nextUrl.pathname === "/healthz" && ["GET", "HEAD"].includes(request.method)) {
    return NextResponse.next();
  }
  const refusal = webAccess(request.headers, webCredentials());
  if (refusal) return refusal;
  if (!["GET", "HEAD", "OPTIONS"].includes(request.method) && !mutationOriginAllowed(request.headers)) {
    return new Response("Cross-origin requests are not allowed.", { status: 403 });
  }
  return NextResponse.next();
}
