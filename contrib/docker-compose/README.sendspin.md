# Drop-in Docker image (Sendspin jukebox)

This fork publishes a container that is a **drop-in replacement** for
[`deluan/navidrome`](https://hub.docker.com/r/deluan/navidrome/): same volumes,
same env vars, same UI port — plus Sendspin on **8927**.

## Swap an existing stack

In your `docker-compose.yml` (or equivalent), change only the image and add the
Sendspin port:

```diff
  navidrome:
-   image: deluan/navidrome:latest
+   image: ghcr.io/shrippen/navidrome:sendspin
    ports:
      - "4533:4533"
+     - "8927:8927"
    volumes:
      - "./data:/data"
      - "/path/to/music:/music:ro"
```

Then enable jukebox/Sendspin (env **or** `navidrome.toml` in `/data`):

```yaml
environment:
  ND_JUKEBOX_ENABLED: "true"
  ND_JUKEBOX_SENDSPIN_ENABLED: "true"
  ND_JUKEBOX_SENDSPIN_PORT: "8927"
  ND_JUKEBOX_SENDSPIN_NAME: "Navidrome"
  ND_JUKEBOX_SENDSPIN_ENABLEMDNS: "true"
```

Or in `/data/navidrome.toml`:

```toml
[Jukebox]
Enabled = true

[Jukebox.Sendspin]
Enabled = true
Port = 8927
Name = "Navidrome"
EnableMDNS = true
```

Recreate the container:

```bash
docker compose pull navidrome   # if using GHCR
docker compose up -d navidrome
```

Your library DB and music mounts stay as-is. In the UI, open the user avatar →
**Sendspin devices** once the feature is enabled.

Connect players to:

```text
ws://<host>:8927/sendspin
```

## Build locally

From the repo root (needs Docker Buildx; first build is slow — UI + Go + CGO):

```bash
docker build \
  --target final \
  --build-arg GIT_SHA="$(git rev-parse --short HEAD)" \
  --build-arg GIT_TAG=sendspin \
  -t navidrome-sendspin:local \
  .
```

Then point compose at `image: navidrome-sendspin:local`.

Full example: [`docker-compose.sendspin.yml`](./docker-compose.sendspin.yml).

## Image tags (GHCR)

| Tag | Meaning |
|-----|---------|
| `ghcr.io/shrippen/navidrome:sendspin` | Current Sendspin fork image |
| `ghcr.io/shrippen/navidrome:sendspin-latest` | Same, explicit latest |
| `ghcr.io/shrippen/navidrome:sendspin-<sha>` | Immutable git SHA build |

Workflow: [`.github/workflows/docker-sendspin.yml`](../../.github/workflows/docker-sendspin.yml).

## Notes

- **ffmpeg** and **mpv** remain in the image (same as upstream).
- **libopus** is included for Sendspin encoding.
- If mDNS discovery fails across Docker networks, use the manual WebSocket URL.
- Rollback: switch `image` back to `deluan/navidrome:latest` and remove port `8927`.
