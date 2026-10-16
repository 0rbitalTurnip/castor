// Test-only Connect server. Never part of the production image or real playback.
import { createServer } from "node:http";
import { setTimeout as delay } from "node:timers/promises";
import { connectNodeAdapter } from "@connectrpc/connect-node";
import { Code, ConnectError } from "@connectrpc/connect";
import type { MessageInitShape } from "@bufbuild/protobuf";
import { CastService, type CastRequest, type WatchResponseSchema } from "../../src/gen/castor/v1/cast_pb";
import { DeviceService } from "../../src/gen/castor/v1/device_pb";

let lastRequest: CastRequest | undefined;
let stopped = false;
let active = false;
let sequence = 0;
let watchCount = 0;
let mode = "normal";
const screen = { id: "screen-1", name: "Test living room", type: 1, address: "192.0.2.10" };
const second = { id: "screen-2", name: "Test bedroom", type: 2, address: "192.0.2.11" };
const adapter = connectNodeAdapter({
  routes(router) {
    router.service(DeviceService, {
      listDevices() {
        if (mode === "offline") throw new ConnectError("Mock API unavailable", Code.Unavailable);
        return { devices: mode === "empty" ? [] : [screen, second] };
      },
    });
    router.service(CastService, {
      cast(req) {
        lastRequest = req;
        if (mode === "cast-error") throw new ConnectError("Mock screen refused the cast", Code.FailedPrecondition);
        stopped = false; active = true; sequence++; watchCount = 0;
        return { castId: `test-cast-${sequence}` };
      },
      listCasts() {
        if (mode === "offline") throw new ConnectError("Mock API unavailable", Code.Unavailable);
        return { casts: active ? [{ id: `test-cast-${sequence}`, device: screen,
          status: { state: { case: "casting" as const, value: { streams: 1, castable: 1, attempt: 1 } } } }] : [] };
      },
      stop() { stopped = true; active = false; return {}; },
      async *watch(_req, context): AsyncGenerator<MessageInitShape<typeof WatchResponseSchema>> {
        watchCount++;
        yield { update: { case: "status" as const, value: { state: { case: "connecting" as const, value: {} } } } };
        await delay(100);
        yield { update: { case: "status" as const, value: { state: { case: "extracting" as const, value: {} } } } };
        await delay(100);
        yield { update: { case: "status" as const, value: { state: { case: "measuring" as const, value: { streams: 1, castable: 1 } } } } };
        await delay(100);
        yield { update: { case: "status" as const, value: { state: { case: "casting" as const, value: { streams: 1, castable: 1, attempt: 1 } } } } };
        if (mode === "disconnect" && watchCount === 1) return;
        if (mode === "completed") {
          active = false;
          yield { update: { case: "ended" as const, value: { result: { case: "completed" as const, value: {} } } } };
          return;
        }
        while (!stopped && !context.signal.aborted) await delay(50);
        if (stopped) yield { update: { case: "ended" as const, value: { result: { case: "stopped" as const, value: {} } } } };
      },
    });
  },
  interceptors: [(next) => async (req) => {
    if (req.header.get("authorization") !== "Bearer mock-api-token-only-for-tests") {
      throw new ConnectError("Wrong server token", Code.Unauthenticated);
    }
    return next(req);
  }],
});
createServer((req, res) => {
  if (req.url?.startsWith("/__test/")) {
    if (req.url.startsWith("/__test/mode/")) mode = req.url.slice("/__test/mode/".length);
    res.setHeader("content-type", "application/json");
    res.end(JSON.stringify({ mode, lastRequest, stopped, active, watchCount }));
    return;
  }
  adapter(req, res);
}).listen(18411, "127.0.0.1", () => console.log("Mock Connect API listening on 18411"));
