import assert from "node:assert/strict";
import test from "node:test";
import { formValues, videoUrl } from "../src/features/cast-form/validation.ts";

const form = (overrides = {}) => {
  const f = new FormData();
  for (const [key, value] of Object.entries({ deviceId: "screen-1", kind: "pages", url: "https://example.test/video", ...overrides })) f.set(key, value);
  return f;
};

test("URLs preserve signed query bytes, normalize scheme and allow trusted LAN media", () => {
  assert.equal(videoUrl(" HTTPS://example.test/a?sig=a%2Fb+X&x=1 "), "https://example.test/a?sig=a%2Fb+X&x=1");
  assert.equal(videoUrl("http://192.168.1.10:8080/video.mp4"), "http://192.168.1.10:8080/video.mp4");
});

test("reject malformed, credential-bearing, multiple and non-HTTP URLs", () => {
  for (const input of ["file:///etc/passwd", "javascript:alert(1)", "//example.test/a", "http:///a",
    "https://user:pass@example.test", "https://example.test\\@evil.test/a", "https://a.test https://b.test",
    "https://a.test/\nfoo", "http://", "https://a.test/" + "a".repeat(8192)]) {
    assert.throws(() => videoUrl(input), undefined, input);
  }
});

test("blank settings inherit defaults; explicit settings are typed", () => {
  const defaults = formValues(form());
  assert.equal(defaults.delivery, undefined);
  assert.equal(defaults.maxHeight, undefined);
  assert.equal(defaults.subtitles, "");
  assert.deepEqual(formValues(form({ kind: "stream", delivery: "2", maxHeight: "2160", subtitles: "off" })),
    { deviceId: "screen-1", kind: "stream", url: "https://example.test/video", delivery: 2, maxHeight: 2160, subtitles: "off" });
});

test("tampered form fields are rejected before RPC", () => {
  for (const overrides of [{deviceId: ""}, {kind: "unknown"}, {delivery: "0"}, {delivery: "NaN"},
    {maxHeight: "Infinity"}, {maxHeight: "1080.5"}, {maxHeight: "-1"}, {maxHeight: "99999"}, {subtitles: "<script>"}]) {
    assert.throws(() => formValues(form(overrides)));
  }
  const duplicates = form(); duplicates.append("kind", "stream");
  assert.throws(() => formValues(duplicates));
  const upload = form(); upload.set("url", new Blob(["bad"]));
  assert.throws(() => formValues(upload));
});
