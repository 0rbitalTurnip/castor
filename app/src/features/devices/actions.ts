"use server";

import { revalidatePath, updateTag } from "next/cache";
import { requireAccess } from "@/lib/require-access";

export async function refreshDevicesAction() {
  await requireAccess(true);
  updateTag("devices");
  revalidatePath("/");
}
