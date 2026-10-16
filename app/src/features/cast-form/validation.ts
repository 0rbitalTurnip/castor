export class InputError extends Error {}

export function videoUrl(value: string): string {
  const input = value.trim();
  if (!input || input.length > 8192 || /[\s\\\u0000-\u001f\u007f]/.test(input) || !/^https?:\/\/[^/?#]/i.test(input)) {
    throw new InputError("Use one complete http:// or https:// URL without spaces (maximum 8192 characters).");
  }
  let url: URL;
  try { url = new URL(input); } catch {
    throw new InputError("Enter a valid video URL.");
  }
  if (!url.hostname || url.username || url.password) {
    throw new InputError("URLs must have a hostname and cannot include a username or password.");
  }
  // Do not reserialize signed query strings.
  return input.replace(/^https?:/i, (scheme) => scheme.toLowerCase());
}

export function formValues(f: FormData) {
  const text = (key: string) => {
    const values = f.getAll(key);
    if (values.length > 1 || (values.length && typeof values[0] !== "string")) {
      throw new InputError(`Invalid ${key}.`);
    }
    return String(values[0] ?? "").trim();
  };
  const deviceId = text("deviceId");
  if (!deviceId || deviceId.length > 2048 || /[\u0000-\u001f\u007f]/.test(deviceId)) {
    throw new InputError("Choose a screen before casting.");
  }
  const kind = text("kind");
  if (kind !== "pages" && kind !== "stream") throw new InputError("Choose a video page or direct stream.");
  const url = videoUrl(text("url"));
  const delivery = text("delivery");
  if (!["", "1", "2"].includes(delivery)) throw new InputError("Choose a valid delivery mode.");
  const height = text("maxHeight");
  if (height && (!/^\d+$/.test(height) || Number(height) < 2 || Number(height) > 4320)) {
    throw new InputError("Maximum height must be a whole number from 2 to 4320.");
  }
  const subtitles = text("subtitles");
  if (subtitles && !["off", "auto"].includes(subtitles) && !/^[a-z]{2,3}$/.test(subtitles)) {
    throw new InputError("Choose off, automatic subtitles, or a two- or three-letter language code.");
  }
  return { deviceId, kind, url, delivery: delivery ? Number(delivery) : undefined,
    maxHeight: height ? Number(height) : undefined, subtitles };
}
