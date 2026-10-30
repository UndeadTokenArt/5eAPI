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

## Campaigns

Campaigns let players share a live view of their characters. Open `/campaigns` to create one; the creator gets a join link (`/campaigns/join/<token>`) that asks which saved characters to add.

- **Shared fields:** name, species, class, current/max HP, AC, initiative (0 = not rolled; sorts the party), conditions, cyber strain, portrait URL, and sheet dice rolls.
- **Editing:** a character's owner or the campaign creator (GM) can edit HP, conditions, strain, initiative and portrait from the campaign screen. Edits are written back to the character sheet, and sheet saves update the campaign.
- **Live sync:** `/ws/campaigns/{id}` sends the full campaign state on connect and after every change; clients reconnect with backoff and refetch state, so a missed message cannot desync.
- **Retention:** state is stored in `CAMPAIGN_DATA_FILE` (default `data/campaigns.json`, `/app/data/campaigns.json` in Docker, inside the existing data volume). After 48 hours without activity the shared info is regathered from each member's sheet and the dice history is cleared. Campaigns idle for 30 days are deleted. The dice history keeps the latest 100 rolls.
- **Mobile:** swipe left on the character sheet to open the campaign, swipe right to return. A reverse proxy must pass WebSocket upgrades (Caddy does by default).
- **Inventory and deliveries:** character sheets label equipment as Inventory and track item quantities. A sheet that is a campaign member can trade only items in that Inventory and Yen it owns. Incoming campaign or player deliveries appear as an accept/deny dialog; acceptance adds items and Yen, while a declined GM delivery returns to the campaign dropbox and a declined player trade returns to its sender.
- **GM dropbox:** the campaign creator can add custom items and Yen, generate random equipment/weapon loot by CR, and send the box to a character. A pending delivery is limited to one per recipient character. Campaign member URL aliases (`user1`, `user2`, ...) are assigned by character join order and remain stable if other members leave.

### Campaign screen extras

- The dice history shows the **character** name, not the account name.
- The creator can add **non-player entities** (name, HP) to the initiative list, then track their HP, initiative and conditions live. They persist until removed.
- Conditions are chosen from a fixed list in `static/js/campaign.js` (`CONDITIONS`); the character sheet's suggestion list in `templates/characters.tmpl` should match.
- **Roll dice** tray: click d4–d20 icons to build a hand, pick which character to roll as, then Roll. Results go to the shared history.
- Changing HP on a saved character sheet updates the campaign within a moment, without needing to save the whole sheet.
