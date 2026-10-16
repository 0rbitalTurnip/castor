# Security

Castor collects no telemetry.

## Keeping secrets out of git

Use git-ignored `config.local.yaml` (overlays `config.yaml`):

```yaml
tmdb:
  api_key: "<key>"
```

Environment variables such as `CASTOR_TMDB__API_KEY` override both files. Revoke accidentally committed keys. The TMDB key is used only by the interactive browser and sent only to [TMDB](https://www.themoviedb.org/settings/api).

## API, media, and scraping services

- **Authenticate RPCs:** set independent `api.token`, `server.token`, and `scraping.token`; use `Authorization: Bearer <token>`. Health checks are exempt. Default ports `:8411`, `:8410`, and `:8412` bind all interfaces and warn if exposed without tokens.
- **Restrict network access:** unauthenticated callers can start casts or make scraping visit URLs, including its local network. Media accepts streams, not webpage inputs.
- **Device playback is open:** `/media` cannot require authentication. Anyone who can reach media and knows a cast URL can fetch it; `server.advertise` defines the device-reachable address.
- **Protect credentials in transit:** RPCs use plain HTTP by default and carry tokens, stream URLs, and replay headers. Use HTTPS beyond trusted networks. IP-bound streams may require scraping and media to share an outbound IP.
- **Keep replay headers private:** they reach media but are stripped from public cast listings.

## Headless Chrome

Only scraping runs Chrome and reads `browser`/`capture` settings. It uses an isolated profile and fresh randomized fingerprint, not your browser's cookies or passwords. It visits supplied pages and their loaded resources, following only HTTP(S) frames.

## Browser interface

The self-hosted web app requires a separate HTTP Basic login and fails closed when credentials are missing or the password is shorter than 32 characters. Authentication is checked both at the web boundary and before application actions/data access. API credentials stay server-side; the browser login does not grant direct access to backend RPC ports.

The Compose UI and API bind loopback by default. Use an SSH tunnel or HTTPS reverse proxy for browser access, and preserve the original Host header through that proxy. HTTP Basic is not encrypted on plain HTTP. Keep this administration interface off the public internet.

Only trusted operators should have access. HTTP(S) URL validation intentionally permits LAN media and does not prevent SSRF through redirects, DNS changes, or page subresources. The backend release image runs Chrome without its sandbox; use a dedicated VM and restrictive network access when processing untrusted pages. See [SELF_HOSTING.md](SELF_HOSTING.md) for deployment boundaries and limitations.

## Image provenance

Release images include an SPDX SBOM and SLSA provenance from the [delivery workflow](.github/workflows/_delivery.yml):

```sh
docker buildx imagetools inspect ghcr.io/stupside/castor:latest --format '{{ json .SBOM }}'
docker buildx imagetools inspect ghcr.io/stupside/castor:latest --format '{{ json .Provenance }}'
```

## Reporting a vulnerability

Open a [GitHub issue](https://github.com/stupside/castor/issues) marked **[security]**, or for sensitive reports, email the maintainer (address on their GitHub profile).
