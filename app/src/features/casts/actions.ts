"use server";

import { revalidatePath } from "next/cache";
import { describe } from "@/lib/errors";
import { casts } from "@/lib/server";
import { requireAccess } from "@/lib/require-access";

export type StopState = { error?: string; stopped?: boolean };

export async function stopAction(castId: string, _: StopState): Promise<StopState> {
  try {
    await requireAccess(true);
    if (typeof castId !== "string" || !castId || castId.length > 2048) return { error: "Invalid cast ID." };
    await casts.stop({ castId }, { timeoutMs: 10_000 });
  } catch (e) {
    return { error: describe(e) };
  }
  revalidatePath("/");
  return { stopped: true };
}
