# Character Sheets

Character sheets are available at `/characters`. When Authelia supplies the authenticated username in `Remote-User`, each sheet is stored under that identity and all reads, edits, and deletes are owner-scoped.

The supplied Compose service temporarily sets `CHARACTER_GUEST_ACCESS: "true"`. If `Remote-User` is absent, the app creates a random, HttpOnly browser cookie and uses it as an isolated guest identity. Guest sheets are available only in that browser profile; clearing cookies loses access to them. Disable this setting after Authelia is correctly configured.

## Multiclass groundwork

The character API accepts an optional `classes` array containing `{ "name", "subclass", "level" }` entries. The first entry must match the existing `class` and `subclass` fields, and class levels must add up to the character's total `level`. Existing sheets without `classes` remain valid; saving a single-class sheet in the current editor records its current class and level without inventing a level-up history.

Spells may include `spell_class` to record their source. The optional `pact_slots` object uses the same `level`, `current` and `max` fields as an ordinary spell slot but is stored separately from `spell_slots`. Existing slot data is not automatically divided into normal and pact slots, because manually changed or expended slots cannot be reliably reconstructed.

`POST /api/character-spellcasting` accepts `{"classes":[{"name":"Cyber-Mage","level":3},{"name":"AI Host","level":5}]}` and returns 2024-style per-class casting rules, shared multiclass Spellcasting slots and a separate Pact Magic pool. It requires the same identity and same-origin `Origin` header as character writes. Mystic Arcanum unlocks are returned separately, not as slots. One Spellcasting class uses its own table; multiple Spellcasting classes use the multiclass table, rounding each Paladin and Ranger contribution up. Casting subclasses are rejected until their 2024 progressions are authored. Other single-class sheet and guide spellcasting displays still need conversion.

`POST /api/multiclass-eligibility` accepts `classes`, `new_class`, and
`ability_scores`. It checks the 2024 primary-ability requirements of every
current class and the new class. The level-up preview displays the result and
required scores. Missing or invalid prerequisite data is an error, not
permission to multiclass. This check is informational: it does not apply or
save a level. The requirements follow the primary abilities in the supplied
class references and the
[2024 multiclass rules](https://www.dndbeyond.com/sources/dnd/br-2024/creating-a-character#Multiclassing).

The **Level up** button previews the draft 2024 worksheet for the selected class,
but applying a level is disabled. All worksheet levels are currently marked
unfilled and the app cannot yet apply their choices, subclass features, or
other effects safely. The previous manual single-class application has also
been disabled so it cannot suggest that an incomplete level-up is complete.
Existing sheet fields remain editable; this restriction is not server-side
enforcement of character creation or manual edits.

The dialog can preview another class but **cannot apply a multiclass level**. The existing sheet cannot edit multiple classes or a separate pact pool, and disables saving for sheets containing either to prevent data loss. Draft previews do not include subclass progression and may omit other features or choices.

`GET /api/class-progression-preview?class=...` returns the draft base-class
features for a mapped class. It requires a character identity and does not
apply or validate a level-up. It supports the twelve supplied worksheets,
including Cyber-Berserker (Barbarian) and Corporate Enforcer
(Fighter), and Shrine Priest (Cleric), though the class selector still comes
from the existing guide.

The Cyber-Mage worksheet also authors Neon Bladesinger, System Forensics
Diviner, and Ghost Protocol Necromancer from the supplied Wizard subclass
references. It records their features, resources, and choices, including
recurring Savant spellbook additions. These subclass entries are not returned
by the preview or applied to sheets; the current guide's older subclasses
are unchanged. The source does not settle whether a Savant also gains an
extra spell when its level-3 feature and level-2 slots are gained together,
so that interaction needs review before automation.

The Vow-Bound Operative worksheet authors the Aegis Enforcer, Chrome Samurai,
and Civic Sentinel vows from the supplied Paladin references. Their fixed oath
spells are recorded at each unlock as always prepared, separately from the
class preparation limit. Civic Sentinel adapts the 2014 Oath of the Crown to
the 2024 class's shared Channel Divinity pool, while retaining the source
features and capstone recharge. The supplied Crown text does not specify a
maximum duration for Champion Challenge, so that requires review before
automation. Compelled Duel, Aura of Vitality, and Circle of Power are absent
from the current spell-reskin reference and spell data and will need entries
before the Crown spell list can be connected. Like the Cyber-Mage subclasses,
these entries are draft data:
the preview and character sheet do not consume them, and older guide
subclasses are unchanged.

The Shrine Priest worksheet records nine reskinned domains from the supplied
Cleric references, including their fixed domain spells, choices, shared
Channel Divinity effects, and limited-use resources. These entries are also
draft-only: the preview and sheet do not return or apply subclass effects.
Some domain spell names have no exact match in the existing reskin mapping;
the worksheet's source notes list the names and likely aliases to review before
integration. The supplied Grave reference truncates the final Keeper of Souls
recovery sentence; the worksheet records that recovery as applying to Keeper
of Souls.

### Awakened Mutant 2024 pilot

The Awakened Mutant worksheet now records six reskinned origins from the
supplied Sorcerer references, including fixed spells, choices, resources, and
the Wild Signal Surge table. These are **draft data only**: neither progression
endpoint returns subclass entries, and the sheet does not apply or track their
effects. The existing guide's origins remain older rules. Evard's Black
Tentacles and Rary's Telepathic Bond need matching to the shorter names in the
spell mapping. The mapping lacks Summon Construct, Chromatic Orb, Dragon's
Breath, Charm Monster, Summon Dragon, and Aura of Vitality; these need matching
spell data before integration.

Awakened Mutant uses the supplied 2024-style Sorcerer prepared-spell totals,
including on existing sheets. Its level-up preview reads its draft worksheet
and shows features and choice prompts; its effects are **not** automatically applied.
Metamagic, spell selections and replacements, feats, subclass features, and hit
dice still require manual updates. The current spell list is the existing
reference mapping and has not been verified as a complete 2024 list. The pilot
tracks Innate Sorcery and mutation points, but spending points on alternate
uses and the level-5 short-rest recovery must be handled manually.

Existing spells, subclass selections, and slot usage are not migrated or
reconstructed. Review them against the new rules before saving. If slot
maximums were customized or reflect older data, review them manually; reset
slots only if a full reset is intended. Existing subclasses remain older guide
content, not 2024 subclass automation.

Multiclass application remains blocked, including entering or leaving this
pilot class, until the sheet can safely edit and save separate classes, pact
slots, and resources. The 2024 prerequisite and slot calculations do not
automate class features, proficiency grants, subclass casting, or spell choices.
The other single-class sheet and guide spellcasting displays still use earlier
rules.

For an existing `NEOTOKYO_DATA_DIR` volume, deploy the updated
`_reference/class-resources.json` there to see the new Innate Sorcery tracker.
Deploy the updated `classesv2/` worksheets there for 2024 prerequisites and
slot calculations; existing volume files override bundled worksheets.
The Barbarian and Fighter worksheets are now named
`Cyber-Berserker.json` and `Corporate Enforcer.json`; older files
under their previous names will not override these worksheets.
The Cleric worksheet is now named `Shrine Priest.json`; its previous filename
will not override the renamed worksheet either.
If the v2 worksheet is absent from that volume, the preview uses the bundled
copy; a present but invalid worksheet produces an error instead of silently
falling back.

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
