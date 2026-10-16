"use server";

import { redirect } from "next/navigation";
import { describe } from "@/lib/errors";
import { casts } from "@/lib/server";
import { preferences, source } from "./request";
import { formValues, InputError } from "./validation";
import { requireAccess } from "@/lib/require-access";

export type FormState = { error?: string; deviceId?: string; kind?: string; url?: string };

export async function submitAction(_: FormState, f: FormData): Promise<FormState> {
  let castId: string;
  try {
    await requireAccess(true);
    const values = formValues(f);
    ({ castId } = await casts.cast({
      target: { target: { case: "deviceId", value: values.deviceId } },
      source: source(values),
      preferences: preferences(values),
    }, { timeoutMs: 30_000 }));
    if (!castId) throw new Error("No cast ID returned");
  } catch (e) {
    return { error: e instanceof InputError ? e.message : describe(e), deviceId: String(f.get("deviceId") ?? ""), kind: String(f.get("kind") ?? ""), url: String(f.get("url") ?? "") };
  }
  redirect(`/casts/${encodeURIComponent(castId)}`);
}
