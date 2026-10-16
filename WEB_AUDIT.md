# Castor workflow and security audit

Reviewed October 15, 2026, America/Chicago. Baseline: `0rbitalTurnip/castor`, commit `3c491f1`. The frontend workflow passes the checks described here after local repairs, but this is not an unconditional security certification or proof of successful TV playback.

## Verdict

The earlier response overstated completion. The inherited workspace contained the conversation but no implementation files, archive, or build evidence. GitHub contained only `main`, no GUI branch, and no pull request. The actual repository already had a Next.js GUI, but lacked the claimed web Docker image, web Compose service, environment example, self-hosting instructions, and visible playback settings.

The review therefore started from the actual repository, reproduced its baseline checks, and made a new, auditable local implementation. The baseline production build, TypeScript check, lint check, and eight existing tests passed. Passing those checks did not establish deployment readiness or browser authentication.

The corrected code has not been pushed to GitHub or installed on the media server. No real device was contacted and no actual media was cast during this review.

## Findings and repairs

| Priority | Finding | Disposition |
| --- | --- | --- |
| High | A browser could reach server actions and the Watch proxy without its own login. Keeping the backend token on the server did not protect the web interface. | Added separate HTTP Basic browser credentials, fail-closed configuration, checks in the request proxy and at actions/data-access boundaries, plus cross-origin mutation rejection. |
| High | The advertised self-hosted web deployment did not exist in the available code. | Added a multi-stage Node 24 standalone image, a web Compose service, liveness endpoint, safe configuration generation, and deployment documentation. |
| High | The API bound all interfaces; the proposed web address implicitly exposed a control interface over unencrypted HTTP. | Compose now binds API and UI to loopback by default. Documentation covers SSH tunneling, HTTPS reverse proxying, explicit trusted-LAN opt-in, and firewall boundaries. |
| Medium | Form parsing trusted source type and freely coerced numeric preferences. Browser validation could be bypassed. | Added server-side validation for one HTTP(S) URL, source type, device ID, delivery mode, resolution, subtitle mode, duplicate fields, and uploaded-file fields. Rejects embedded URL credentials, control characters, backslashes, unsupported schemes, and excessive length. |
| Medium | Preferences existed in the server parser but had no visible controls. | Added optional delivery, resolution, and subtitle controls. Confirmed their exact values at the mock Connect API. Blank fields still inherit server defaults. |
| Medium | The base Compose stack required `/dev/dri`, including on CPU-only hosts. | Moved GPU passthrough to an optional Compose override. Actual GPU behavior remains untested. |
| Medium | Watch requests could forward an arbitrarily large request body and wait indefinitely for upstream headers. | Added a 16 KiB input limit, a five-second input-read deadline, and a 15-second upstream-header deadline. Established response streams are not given a fixed total lifetime. |
| Medium | The baseline stack shared a backend cache volume across services. | Split media and scraping cache volumes to reduce unnecessary cross-service access. |
| Low | Clipboard failure was silently ignored, particularly confusing on LAN HTTP. | Added an explicit keyboard/touch-paste fallback message. |
| Low | Button hover translation could oscillate at the pointer edge and prevent a stable retry click. | Removed movement from button hit targets. Mouse retry now passes the browser check. |
| Low | Development setup did not explain the new required browser credentials. | Updated README, CONTRIBUTING, SECURITY, and SELF_HOSTING documentation. |

The original backend release Dockerfile and restrictive root `.dockerignore` are preserved. A Dockerfile-specific allowlist supplies only the frontend build context, excluding `.env` and other runtime credentials. The web runtime is non-root, drops capabilities, and enables no-new-privileges in Compose.

## Verification completed

### Automated checks

| Check | Result | Scope |
| --- | --- | --- |
| Immutable Yarn install | Pass | Original dependency lockfile retained; existing upstream peer warnings remain |
| Production Next.js build | Pass | Node 24, standalone output including request proxy |
| TypeScript | Pass | Application and TypeScript test fixture |
| ESLint | Pass | Compatible existing ESLint 9 toolchain |
| Unit and HTTP regression tests | 16 passed, 0 failed | Eight original tests plus validation, browser-auth, origin, and body-limit coverage |
| Production dependency audit | Pass | No reported production dependency advisories at review time; not a guarantee against undiscovered issues |
| Base Compose validation | Pass | `config --quiet` with dummy test configuration |
| Optional GPU Compose merge | Pass | Both Compose files validate together |
| Setup shell syntax | Pass | `sh -n` |
| Setup permissions | Pass | Generated `.env` has mode `0600` |
| Existing-secret protection | Pass | Re-running setup refuses to overwrite `.env`; file hash unchanged |
| Diff whitespace check | Pass | `git diff --check` |
| Production server without credentials | Pass | Home and Watch requests return HTTP 401 |
| Cross-origin authenticated mutation | Pass | Returns HTTP 403 |
| Liveness route | Pass | Returns HTTP 200 and no private configuration |
| Request-proxy bypass-header attempt | Pass | Unauthenticated request remained HTTP 401 |
| Client asset inspection | Pass | Test API token and browser test password absent from client static assets |

The streaming regression suite checks header filtering, server-side credential injection, redirect refusal, decompressed HTTP headers, cancellation of quiet upstream streams, sanitized connection failure, premature EOF handling, terminal events, and oversized request rejection.

### Real browser against a mock Connect API

The browser ran the production standalone Next.js server, not a static mockup. Only the upstream Castor API was simulated. The fixture uses the repository's generated protobuf service descriptors and the real Connect server adapter, and rejects calls without the expected server-only test token.

- Screen selection: selected both mock devices and verified the requested device ID.
- Direct stream: submitted a signed-query-style URL and verified the exact preserved URL and `stream` source variant.
- Video page: verified the `pages` source variant with one URL.
- Invalid input: a `file://` URL could not be submitted through the UI.
- Preferences: verified relay mode `2`, height `720`, and explicit disabled subtitles at the API; separately verified blank defaults remain unset.
- Cast lifecycle: observed connecting, extracting, measuring, casting, and stopped states.
- Active casts: returned home, found the active cast, requested stop, and confirmed removal.
- Recovery: a clean EOF without an `Ended` event showed connection loss, not success; the Reconnect button restored status.
- Discovery failure: tested empty device discovery and API-unavailable feedback, then refreshed successfully.
- Cast failure: tested a refused cast, visible retry, and successful mouse retry after the button fix.
- Completion: verified the message that a direct hand-off can finish Castor's work while the TV continues playing.
- Clipboard: denied clipboard permission and confirmed the manual-paste fallback.
- Mobile: used a 375 × 812 viewport; opened settings, started a cast, viewed progress, and stopped it.
- Desktop: used a 1440 × 1000 viewport; checked settings, failure/retry, and completion.

Desktop and mobile screenshots were inspected. The tested states had no horizontal overflow or obstructed primary controls, and no JavaScript page errors were recorded in the desktop test context. This was a targeted usability check, not a comprehensive accessibility audit.

## Outstanding risks and unverified behavior

### Dependency/toolchain findings

The full development dependency audit reports `braces@3.0.3`, a high-severity stack-exhaustion issue with no patched version listed at review time ([GitHub advisory GHSA-vfj7-8cjw-p6xm](https://github.com/advisories/GHSA-vfj7-8cjw-p6xm)). In this checkout it is a development-tool dependency through `micromatch` and `fast-glob`; it was absent from the inspected standalone runtime, and the production-only audit did not report it.

The full audit also reports ESLint 9.39.5 as deprecated. An attempted ESLint 10 upgrade broke the current React plugin (`contextOrFilename.getFilename is not a function`), so the upgrade was reverted rather than leaving a failing lint toolchain. Keep development inputs trusted and revisit both findings when compatible upstream updates are available.

### Deployment and backend limits

- Docker runtime: no Docker daemon was available in the sandbox. Compose schema and merge checks passed, and the equivalent standalone web server ran, but the image was not actually built or started in Docker here.
- CI: added Compose-validation and web-image-build steps. Those new GitHub Actions steps have not run because the changes have not been pushed.
- Go/Whisper: native backend build, backend Go tests, FFmpeg integration tests, and model downloads were not executed.
- Backend version: Compose still uses an upstream image selected by `CASTOR_IMAGE`. A tested image digest has not been established. The default `latest` tag can drift from this frontend's generated contract; validate and pin a compatible digest before calling the deployment reproducible.
- Devices and network: no real DLNA, Chromecast, or Roku discovery, multicast routing, audio/video output, subtitle rendering, transcoding, or GPU acceleration was verified.
- Restart behavior: web liveness is not backend readiness. Backend state is not promised to survive restarts.
- Direct hand-off: completion is not necessarily playback completion, and a finished backend cast may no longer expose a stop action.

### Security boundaries

- Basic authentication needs HTTPS, an SSH tunnel, or another encrypted transport. Plain LAN HTTP exposes credentials to network observers.
- The default UI is deliberately loopback-only. `http://MEDIA-SERVER-IP:3000` will not work until the operator explicitly selects LAN binding or configures a reverse proxy.
- The login is for trusted household operators, not public multi-tenancy. There is no per-user authorization, MFA, login throttling, or session revocation.
- The application intentionally accepts private/LAN HTTP URLs. URL syntax checks are not SSRF protection against redirects, DNS changes, or webpage subresources.
- The upstream backend container runs Chrome without its sandbox, and backend services have broader privileges than the web container. Use a dedicated VM and restrict network access for untrusted processing.
- TV media URLs cannot require the same browser/API authentication. Restrict the media port to intended devices and treat cast URLs as sensitive.
- The reviewed UI retains the repository's existing dark visual design. The earlier promise of a light/dark theme switch was not present and is not being claimed as delivered.

## Required final acceptance on the media server

Use the reviewed code and follow `SELF_HOSTING.md`. Do not run the earlier abbreviated deployment instructions against unchanged `main`.

1. Configure the server's real LAN advertisement address and generate separate credentials.
2. Build and start the Docker stack, then inspect container state and logs.
3. Verify wrong or missing browser credentials cannot reveal devices or control casts.
4. Cast a known-good authorized direct media URL to each intended device type. Confirm picture and sound on the physical screen.
5. Test an authorized webpage, stop, reconnect, and the required playback settings.
6. Test CPU mode before enabling optional GPU passthrough.
7. Test HTTPS/phone access and a full service restart.
8. Pin the successfully tested backend image digest and retain a rollback reference.

Only after these steps pass should the complete media-server-to-TV outcome be marked attained. The evidence here supports the repaired frontend and mocked API workflow, not an unqualified claim that all code is correct or that the entire deployment is secure.
