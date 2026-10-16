import "server-only";
import type { MessageInitShape } from "@bufbuild/protobuf";
import type { PreferencesSchema, SourceSchema } from "@/gen/castor/v1/cast_pb";
import type { formValues } from "./validation";

type Values = ReturnType<typeof formValues>;

export const source = (f: Values): MessageInitShape<typeof SourceSchema> =>
  f.kind === "pages"
    ? { source: { case: "pages", value: { urls: [f.url] } } }
    : { source: { case: "stream", value: { url: f.url } } };

// Blank inherits the server's configuration; explicit subtitle choices set the mode.
export const preferences = (f: Values): MessageInitShape<typeof PreferencesSchema> => {
  const subs = f.subtitles;
  return {
    delivery: f.delivery,
    maxHeight: f.maxHeight,
    subtitles: !subs ? undefined : {
      mode: subs === "off" ? { case: "disabled", value: {} }
        : subs === "auto" ? { case: "autoDetect", value: {} }
        : { case: "language", value: subs },
    },
  };
};
