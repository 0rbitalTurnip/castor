# Self-hosting the Castor web interface

This configuration is intended for a Linux Docker host on the same reachable network as your TVs. It is a trusted-household administration tool, not a public multi-user streaming service.

## What runs where

| Service | Binding | Purpose |
| --- | --- | --- |
| Web | `127.0.0.1:3000` by default, host network | Browser UI with separate HTTP Basic login |
| API | `127.0.0.1:8411`, host network | Device discovery and cast lifecycle |
| Scraping | Host loopback `8412` mapped into a bridge container | Chrome-based webpage extraction |
| Media | Host network `8410` | Media API and device-reachable playback URLs |

The backend uses the upstream `CASTOR_IMAGE`; building the web image does not compile the Go backend. The root Dockerfile expects prebuilt Go binaries and is not a source-build Dockerfile.

## First install

Use the reviewed checkout that includes `app/Dockerfile`, `scripts/setup-web.sh`, and this document. These instructions do not apply to an unchanged upstream checkout.

```bash
./scripts/setup-web.sh
nano .env
```

Set `CASTOR_SERVER__ADVERTISE` to the media server's actual LAN address, for example `http://192.168.1.50:8410`. Do not use `localhost`, a Docker bridge address, or a VPN address your TV cannot reach.

The setup script generates four independent 256-bit random values and writes `.env` with owner-only permissions. It refuses to overwrite existing configuration. The browser username defaults to `castor`; its password is `CASTOR_WEB_PASSWORD` in `.env`, not any of the API tokens. Never commit or share `.env`.

Then validate and start:

```bash
docker compose config --quiet
docker compose pull
docker compose up -d --build
docker compose ps
docker compose logs --tail=100 web api-server media-server scraping-server
```

Do not use plain `docker compose config` when sharing output; it prints interpolated secrets. Treat logs as potentially sensitive because source URLs can contain signed access tokens.

## Open the UI safely

The safe default binds the UI only to the server's loopback address. From your workstation, open an SSH tunnel:

```bash
ssh -N -L 3000:127.0.0.1:3000 YOUR_SSH_USER@MEDIA_SERVER_IP
```

Open `http://localhost:3000` and sign in using the web credentials from `.env`. Replace the uppercase placeholders with your actual SSH username and server address.

For convenient phone access, use an HTTPS reverse proxy reachable only over your trusted LAN or VPN. Proxy to `127.0.0.1:3000` from the host network, preserve the original `Host` and `Authorization` headers, disable response buffering on `/rpc/`, and permit long-running streaming responses. A bridge-network proxy container cannot reach the host's loopback interface directly; use a host-network proxy or deliberately configure a reachable private binding.

If you explicitly accept unencrypted trusted-LAN access, set `CASTOR_WEB_BIND=0.0.0.0`, recreate `web`, and open `http://MEDIA_SERVER_IP:3000`:

```bash
docker compose up -d web
```

HTTP Basic credentials are not encrypted over plain HTTP. Do not forward ports 3000, 8410, 8411, or 8412 to the internet. Restrict TCP 8410 to the TV/client subnet with your host firewall. Host-network services do not use Docker bridge port isolation.

Clipboard reading may be unavailable over LAN HTTP. The Paste button explains the fallback; ordinary keyboard/touch-menu pasting still works.

## Optional GPU acceleration

The default stack does not require `/dev/dri` and works on CPU-only hosts. If the Linux host has supported VA-API hardware and `/dev/dri`, use both Compose files for every lifecycle command:

```bash
docker compose -f docker-compose.yml -f compose.gpu.yaml up -d --build
```

Passing the device through does not prove the selected codec/driver supports hardware encoding. Verify logs and playback before relying on GPU acceleration.

## First-cast acceptance checklist

- Sign-in: no credentials or a wrong password must not reveal device names or permit casting.
- Discovery: refresh screens; confirm the selected TV is actually yours and reachable.
- Direct stream: cast a known-good, authorized MP4/HLS URL and verify both picture and sound on the TV.
- Webpage: repeat with an authorized non-DRM webpage. Capture can fail on unsupported sites.
- Status: verify progress, stop, home navigation, and active-cast listing.
- Recovery: disconnect and reconnect the status view. Do not blindly resubmit a cast after a timeout; it may already have started.
- Settings: verify relay mode, a lower resolution, and optional subtitles on a supported device.
- Restart: restart the stack and confirm the UI recovers; in-memory casts are not promised to survive a backend restart.
- Phone: verify the same workflow through your chosen HTTPS/VPN access path.

Completion can mean a successful direct hand-off; playback may continue on the TV after Castor finishes. A stop control is not guaranteed after the backend ends that hand-off. Subtitles require model downloads and additional processing, and do not apply to every device path.

## Operational safety and limits

The web runtime runs as a non-root user with dropped Linux capabilities and no-new-privileges. Only the API token is passed to the web container; media and scraping tokens are not exposed to the browser. Runtime credentials are not build arguments.

The upstream backend image still runs Chrome with `--no-sandbox` and has broader privileges than the web container. The media/API containers use host networking for discovery and TV reachability. Treat arbitrary webpage/media processing as untrusted workload: prefer a dedicated VM, do not mount personal media libraries or the Docker socket, use only trusted operators, and keep the upstream image patched.

Input validation allows private/LAN HTTP media intentionally. It is not an SSRF sandbox: authenticated users and fetched pages can cause requests from the server network. Use network egress restrictions if your threat model requires protection of internal services. The UI has one shared household login, not per-user permissions, MFA, rate limiting, or session revocation; use an authentication proxy for broader access.

`/healthz` checks web-process liveness only. It does not prove discovery, backend authentication, or playback works. `depends_on` controls startup order, not backend readiness; the UI offers refresh/retry when the backend is unavailable.

`CASTOR_IMAGE=...:latest` is a moving tag. After validating compatibility with this frontend, pin a tested digest in `.env` and use the same digest for all backend services. Pulling a newer backend can break the unstable RPC contract. The review did not establish a tested backend image digest.

## Developer checks

Run from the repository root with Node 24 or newer:

```bash
node .yarn/releases/yarn-4.18.1.cjs install --immutable
node .yarn/releases/yarn-4.18.1.cjs workspace castor-app test
node .yarn/releases/yarn-4.18.1.cjs workspace castor-app typecheck
node .yarn/releases/yarn-4.18.1.cjs workspace castor-app lint
node .yarn/releases/yarn-4.18.1.cjs workspace castor-app build
node .yarn/releases/yarn-4.18.1.cjs npm audit --all --recursive --environment production
```

Native Go/Whisper builds and device integration tests are separate from frontend tests. Passing a frontend build or mock API test must not be represented as successful real-device playback.
