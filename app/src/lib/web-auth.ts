import { createHash, timingSafeEqual } from "node:crypto";

export type WebCredentials = { username?: string; password?: string };
const digest = (value: string) => createHash("sha256").update(value).digest();

// A separate browser credential is essential: the API token is not a UI login.
export function webAccess(headers: Headers, credentials: WebCredentials): Response | null {
  const { username, password } = credentials;
  const responseHeaders = { "cache-control": "no-store" };
  if (!username || username.includes(":") || !password || password.length < 32) {
    return new Response("Set CASTOR_WEB_USERNAME and a CASTOR_WEB_PASSWORD of at least 32 characters.", {
      status: 503, headers: responseHeaders,
    });
  }
  const authorization = headers.get("authorization") ?? "";
  const encoded = /^Basic ([A-Za-z0-9+/]+=*)$/i.exec(authorization)?.[1];
  const decoded = encoded && encoded.length <= 4096 ? Buffer.from(encoded, "base64").toString("utf8") : "";
  const separator = decoded.indexOf(":");
  const userOK = timingSafeEqual(digest(decoded.slice(0, separator)), digest(username));
  const passwordOK = timingSafeEqual(digest(separator < 0 ? "" : decoded.slice(separator + 1)), digest(password));
  if (separator < 0 || !userOK || !passwordOK) {
    return new Response("Sign in to Castor.", {
      status: 401,
      headers: { ...responseHeaders, "www-authenticate": 'Basic realm="Castor", charset="UTF-8"' },
    });
  }
  return null;
}

export function mutationOriginAllowed(headers: Headers): boolean {
  const site = headers.get("sec-fetch-site");
  if (site && site !== "same-origin" && site !== "none") return false;
  const origin = headers.get("origin");
  if (!origin) return true; // Non-browser clients still need the full credential.
  try {
    const parsed = new URL(origin);
    return ["http:", "https:"].includes(parsed.protocol) && parsed.host === headers.get("host");
  } catch {
    return false;
  }
}

export const webCredentials = (): WebCredentials => ({
  username: process.env.CASTOR_WEB_USERNAME,
  password: process.env.CASTOR_WEB_PASSWORD,
});
