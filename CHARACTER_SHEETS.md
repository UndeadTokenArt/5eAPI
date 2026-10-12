# Character Sheets

Character sheets are available at `/characters`. When Authelia supplies the authenticated username in `Remote-User`, each sheet is stored under that identity and all reads, edits, and deletes are owner-scoped.

The supplied Compose service temporarily sets `CHARACTER_GUEST_ACCESS: "true"`. If `Remote-User` is absent, the app creates a random, HttpOnly browser cookie and uses it as an isolated guest identity. Guest sheets are available only in that browser profile; clearing cookies loses access to them. Disable this setting after Authelia is correctly configured.

## Authelia and Proxy Requirements

Configure the reverse proxy to protect both `/characters` and `/api/characters` with Authelia forward authentication. Pass Authelia's authenticated `Remote-User` response header to this app. The proxy must remove any client-supplied `Remote-User` header before setting its trusted value. Do not publish port 8080 directly to the internet; only the trusted proxy should be able to reach the app container. If Caddy blocks the request before forwarding it, the app's guest mode cannot override that proxy policy; temporarily exempt these routes in Caddy or disable guest access and finish Authelia configuration.

Browser writes also require a same-origin `Origin` header, as an additional CSRF check.

## Persistent Compose Storage

This repository does not contain the server's Compose file. Add the environment variable and volume to the existing app service, retaining its current image/build and proxy networks:

```yaml
services:
  dnd5e:
    environment:
      CHARACTER_DATA_FILE: /app/data/characters.json
    volumes:
      - ./dnd5e-data:/app/data
    expose:
      - "8080"
```

Keep the bind-mounted `./dnd5e-data` directory on persistent host storage and include it in backups. The application creates the directory and file as needed. For local runs, the default path is `data/characters.json`.

The file store is designed for one app instance. If the service will run multiple replicas, use a shared database such as PostgreSQL instead of independent local files.
