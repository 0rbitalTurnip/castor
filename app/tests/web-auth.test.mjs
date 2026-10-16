import assert from "node:assert/strict";
import test from "node:test";
import { mutationOriginAllowed, webAccess } from "../src/lib/web-auth.ts";

const credentials = { username: "castor", password: "a-long-local-test-password-of-32-characters" };
const authorization = (username = credentials.username, password = credentials.password) =>
  new Headers({ authorization: `Basic ${Buffer.from(`${username}:${password}`).toString("base64")}` });

test("browser login fails closed when missing or weakly configured", () => {
  for (const config of [{}, { username: "castor" }, { ...credentials, password: "short" }, { ...credentials, username: "bad:user" }]) {
    assert.equal(webAccess(authorization(), config).status, 503);
  }
});
test("valid browser credentials work; missing, malformed and wrong credentials fail", () => {
  assert.equal(webAccess(authorization(), credentials), null);
  for (const headers of [new Headers(), new Headers({ authorization: "Basic !!!" }),
    new Headers({ authorization: "Bearer server-token" }), authorization("wrong"), authorization("castor", "wrong")]) {
    const response = webAccess(headers, credentials);
    assert.equal(response.status, 401);
    assert.equal(response.headers.get("cache-control"), "no-store");
  }
});
test("cross-site mutations and null origins are rejected, same-origin and authenticated CLI allowed", () => {
  assert.equal(mutationOriginAllowed(new Headers({ host: "castor.local:3000", origin: "http://castor.local:3000", "sec-fetch-site": "same-origin" })), true);
  for (const extra of [{origin: "https://evil.test"}, {origin: "null"}, {"sec-fetch-site": "cross-site"}, {"sec-fetch-site": "same-site"}]) {
    assert.equal(mutationOriginAllowed(new Headers({ host: "castor.local:3000", ...extra })), false);
  }
  assert.equal(mutationOriginAllowed(new Headers()), true);
});
