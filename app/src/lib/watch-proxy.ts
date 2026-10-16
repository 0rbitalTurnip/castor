const protocolHeaders = ["content-type", "connect-protocol-version", "connect-timeout-ms", "connect-content-encoding", "connect-accept-encoding"];

function copyProtocolHeaders(from: Headers) {
  const headers = new Headers();
  for (const name of protocolHeaders) {
    const value = from.get(name);
    if (value !== null) headers.set(name, value);
  }
  return headers;
}

export async function proxyWatch(req: Request, api: { url: string; token: string }): Promise<Response> {
  const abort = new AbortController();
  const cancel = () => abort.abort(req.signal.reason);
  req.signal.addEventListener("abort", cancel, { once: true });
  if (req.signal.aborted) cancel();
  const release = () => req.signal.removeEventListener("abort", cancel);
  const headers = copyProtocolHeaders(req.headers);
  // Fetch decodes HTTP compression; never advertise the browser's encodings or relay its credentials.
  headers.set("accept-encoding", "identity");
  if (api.token) headers.set("authorization", `Bearer ${api.token}`);

  // Watch contains only a cast ID. Bound both declared and chunked input before
  // forwarding; the response remains a streaming body of unlimited duration.
  const input = req.body?.getReader();
  let inputTimedOut = false;
  const inputTimer = setTimeout(() => {
    inputTimedOut = true;
    void input?.cancel().catch(() => {});
  }, 5_000);
  const chunks: Uint8Array[] = [];
  let size = 0;
  try {
    if (Number(req.headers.get("content-length")) > 16_384) {
      await input?.cancel();
      return new Response("Watch request is too large.", { status: 413 });
    }
    if (input) {
      for (;;) {
        const { value, done } = await input.read();
        if (done) break;
        size += value.byteLength;
        if (size > 16_384) {
          await input.cancel();
          return new Response("Watch request is too large.", { status: 413 });
        }
        chunks.push(value);
      }
    }
    if (inputTimedOut || req.signal.aborted) return new Response("Watch request timed out.", { status: 408 });
  } catch {
    return new Response("Could not read watch request.", { status: 400 });
  } finally {
    clearTimeout(inputTimer);
    input?.releaseLock();
    // The request abort handler is reattached below once input has been read.
    release();
  }
  req.signal.addEventListener("abort", cancel, { once: true });
  if (req.signal.aborted) cancel();
  const requestBody = new Uint8Array(size);
  let offset = 0;
  for (const chunk of chunks) { requestBody.set(chunk, offset); offset += chunk.length; }
  // Only waiting for response headers is timed out, not an active status stream.
  const headerTimer = setTimeout(() => abort.abort(), 15_000);
  try {
    const init: RequestInit & { duplex: "half" } = {
      method: "POST", headers, body: requestBody, duplex: "half",
      signal: abort.signal, cache: "no-store", redirect: "error",
    };
    const upstream = await fetch(`${api.url.replace(/\/+$/, "")}/castor.v1.CastService/Watch`, init);
    clearTimeout(headerTimer);
    const responseHeaders = copyProtocolHeaders(upstream.headers);
    responseHeaders.set("cache-control", "no-store");
    responseHeaders.set("x-accel-buffering", "no");
    if (!upstream.body) {
      release();
      return new Response(null, { status: upstream.status, headers: responseHeaders });
    }
    const reader = upstream.body.getReader();
    const body = new ReadableStream<Uint8Array>({
      async pull(controller) {
        try {
          const { done, value } = await reader.read();
          if (done) {
            release();
            controller.close();
          } else controller.enqueue(value);
        } catch (e) {
          release();
          controller.error(e);
        }
      },
      async cancel(reason) {
        release();
        try {
          await reader.cancel(reason);
        } finally {
          abort.abort(reason);
        }
      },
    });
    return new Response(body, { status: upstream.status, headers: responseHeaders });
  } catch {
    release();
    return Response.json({ code: "unavailable", message: "Can’t reach the Castor API. Try reconnecting." }, {
      status: 502, headers: { "cache-control": "no-store" },
    });
  } finally {
    clearTimeout(headerTimer);
  }
}
