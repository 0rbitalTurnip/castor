import "server-only";
import { unstable_cache } from "next/cache";
import { devices } from "@/lib/server";
import { env } from "@/lib/env";
import { requireAccess } from "@/lib/require-access";

const cachedDevices = unstable_cache(
  async () => (await devices.listDevices({}, { timeoutMs: 15_000 })).devices,
  ["castor-devices", env.api.url],
  { revalidate: 15, tags: ["devices"] },
);

export async function listDevices() {
  await requireAccess();
  return cachedDevices();
}
