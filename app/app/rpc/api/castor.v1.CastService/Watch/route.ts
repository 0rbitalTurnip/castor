import { env } from "@/lib/env";
import { proxyWatch } from "@/lib/watch-proxy";
import { mutationOriginAllowed, webAccess, webCredentials } from "@/lib/web-auth";

// Watch is the only call a browser makes; everything else runs in server actions.
export async function POST(req: Request) {
  const refusal = webAccess(req.headers, webCredentials());
  if (refusal) return refusal;
  if (!mutationOriginAllowed(req.headers)) return new Response("Forbidden", { status: 403 });
  return proxyWatch(req, env.api);
}
