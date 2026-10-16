import type { NextConfig } from "next";
import path from "node:path";

// Phones and other machines open the dev server by LAN address; without this their pages render but never hydrate.
const config: NextConfig = {
  typedRoutes: true,
  output: "standalone",
  outputFileTracingRoot: path.join(process.cwd(), ".."),
  poweredByHeader: false,
  allowedDevOrigins: ["192.168.*.*", "10.*.*.*", "*.local"],
  async headers() {
    return [{ source: "/:path*", headers: [
      { key: "X-Content-Type-Options", value: "nosniff" },
      { key: "X-Frame-Options", value: "DENY" },
      { key: "Referrer-Policy", value: "no-referrer" },
      { key: "Content-Security-Policy", value: "frame-ancestors 'none'; base-uri 'self'; object-src 'none'" },
    ] }];
  },
};

export default config;
