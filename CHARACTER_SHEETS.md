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

## Editable NeoTokyo guide data

The guide data lives in a folder instead of one big file: `static/data/neotokyo/` (see its `README.txt`).
Each category has its own subfolder (`spells/`, `classes/`, `weapons/`, ...) and every `*.json` file in it is
merged in filename order. A file may hold an array of entries or a single entry. `_reference/` holds the class
name mapping and class spell lists.

In Docker, `NEOTOKYO_DATA_DIR=/app/data/neotokyo` sits inside the existing `./data/5etools:/app/data` volume, so
the files appear on the host at `./data/5etools/neotokyo/`. On first start with an empty folder the built-in
copy is seeded; existing files are never overwritten. Changes (edits, new files) are picked up within about a
second without a restart; an invalid JSON file is logged and the last good data stays live.
Entry IDs are position based, so append new entries rather than inserting in the middle to keep deep links stable.

## Editor portal (`/editor`)

Members of the Authelia group `editor` (override with `EDITOR_GROUP`) can add and remove guide entries at `/editor`.
The app trusts the `Remote-User` and `Remote-Groups` headers, so the reverse proxy must forward both
(Caddy: `copy_headers Remote-User Remote-Groups`). Additions go to `zz-editor-additions.json` inside each category
folder of the data volume, so bundled files are never modified and existing entry IDs don't shift. Simple categories
(items, weapons, spells, conditions, backgrounds) have forms; abilities, species, classes and feats use a JSON editor.

Because a bypass rule doesn't forward identity headers, protect the portal in Authelia:

    - domain: undeadtoken.com
      resources: ['^/editor', '^/api/editor']
      policy: one_factor
      subject: 'group:editor'

placed above the bypass rule, and make sure Caddy sends `/editor` and `/api/editor` through `forward_auth`.
