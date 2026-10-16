import "server-only";
import { headers } from "next/headers";
import { mutationOriginAllowed, webAccess, webCredentials } from "./web-auth";

// Check at the data boundary too, not just at the page/proxy boundary.
export async function requireAccess(mutation = false) {
  const incoming = await headers();
  if (webAccess(incoming, webCredentials()) || (mutation && !mutationOriginAllowed(incoming))) {
    throw new Error("Unauthorized");
  }
}
