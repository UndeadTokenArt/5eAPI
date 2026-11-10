# NeoTokyo Guide Data

This folder contains the data used to render the NeoTokyo guide. Each category
has its own folder. The application reads every `.json` file in a category
folder in filename order and adds its entries to that category.

You can add a single JSON object or an array of objects in a new file, for
example `spells/my-homebrew-spells.json`. Changes are picked up within a couple
of seconds without a restart or rebuild.

Editor users can also create custom categories such as History, Personas, or
Corporations from the editor page. The category setup lets you define each
field's label, JSON key, input type, and display style. Those definitions are
saved in `_categories/<category-slug>.json`; entries are saved in the matching
category folder. The available display styles are paragraph, heading, quote,
callout, and badge. Each field can also set small, normal, or large text, bold,
and italic. The saved settings are used by both the editor form and guide page.
Custom categories and their entries also appear in the guide menu and search.
The editor supports single-line text, long text, and comma-separated list
inputs. The entry `name` is the displayed heading and is supplied separately
from the custom fields.

To change a custom category, choose it under **Existing custom category** on
the editor page, update its title, fields, or display settings, and save. Its
URL and storage folder keep the original slug if the title changes. Schema
changes that would make existing entries invalid are rejected; edit or remove
those entries before retrying. In the category's additions list, choose
**Edit** to update an entry or **Remove** to delete it. The editor only changes
entries in `zz-editor-additions.json`; bundled guide data must be edited in its
JSON files.

## Classes

Each class is its own file in `classes/`, named after the class (for example
`classes/SuperStud.json`). The file holds one class object (`name`, `features`,
`subclasses`). Drop in a new file and the class appears in the guide and the
character sheet. Class resources and spell lists for it can be added in the
`_reference` files. If two files define the same class name, the later file
(alphabetically) wins; an old combined `classes.json` is always overridden.

## Categories

The category folder names are:

- `abilities`
- `backgrounds`
- `species`
- `classes`
- `feats`
- `adventuring-gear-and-containers`
- `tools-kits-and-consumables`
- `ammunition`
- `weapons`
- `armor-and-shields`
- `mounts-vehicles-and-harnesses`
- `foci-and-instruments`
- `spells`
- `conditions`
- `non-player-entities`

## JSON Format

- Use valid JSON: double-quoted keys and strings, commas between properties
  and items, and no trailing commas. Escape quotes and backslashes inside
  strings.
- Field names use lowercase `snake_case`. Unknown fields are ignored, and
  optional fields can be omitted.
- All entries require a `name`.
- Use `description` for a short summary and `content` for longer text.
  `details` and `features` are arrays; keep nested features under the feature
  or subclass they describe so the guide renders them at the right level.
- Any entry or nested entry can include an optional `image` string containing
  an image URL. The guide displays it as a small icon beside that entry's name;
  omit the field or leave it as `""` until an image is available. For example:
  `"image": "https://example.com/images/operative.png"`.
- Category fields are listed below. Fields not listed for a category are not
  used by that category's guide renderer.

### Category Fields

| Category | Additional fields |
| --- | --- |
| `abilities` | `features`: array of objects with `name`, `example`, `description`, and `content` |
| `backgrounds` | `description`, `content`, `skill_proficiencies`, `starting_equipment`, `background_feature`, `tool_proficiencies`, `tool_language_proficiencies` |
| `species` | `origin`, `appearance`, `cultural_fit`, `trait`: array of objects with `name` and `description` |
| `classes` | `description`, `content`, `details` (string array), `focus`, `defensive_flourish`, `slashing_flourish`, `mobile_flourish`, `features`, `subclasses` |
| `feats` | Group entries by `name`; `features`: array of objects with `name`, `prerequisite`, `description`, `content`, and `details` (string array) |
| Gear, ammunition, armor, mounts, and foci categories | `price`, `description` |
| `weapons` | `price`, `damage`, `damage_type`, `attribute`, `tag`, `description` |
| `spells` | `level`, `school`, `casting_time`, `range`, `duration`, `components` (string array), `description`, `mechanics` |
| `conditions` | `description`, `content`, `details` (string array), and optional `level_1` through `level_6` |
| `non-player-entities` | SRD monster-style stat block: `size`, `type`, `alignment`, `armor_class`, `hit_points`, `hit_points_roll`, `speed`, the six ability scores (`strength`…`charisma`), `senses`, `languages`, `challenge_rating`, `xp`, `special_abilities`, `actions`, `reactions`, `legendary_actions` and more |

For class features, `features` is an array of objects with `name`,
`description`, `content`, and optional nested `features` (objects with `name`
and `description`). Subclasses use the same fields as classes.

## Examples

Save these as `.json` files in the matching category folder.

### Weapon

```json
{
  "name": "Pulse Baton",
  "price": "25 gp",
  "damage": "1d6",
  "damage_type": "lightning",
  "attribute": "DEX",
  "tag": "Finesse",
  "description": "A compact shock weapon."
}
```

### Spell

```json
{
  "name": "Signal Veil",
  "level": "1",
  "school": "Illusion",
  "casting_time": "1 Action",
  "range": "Self",
  "duration": "10 minutes",
  "components": ["V", "S"],
  "description": "A brief summary.",
  "mechanics": "The full rules text."
}
```

### Class with a subclass and nested features

```json
{
  "name": "Example Operative",
  "description": "Class summary.",
  "features": [
    {
      "name": "Combat Protocol (Level 1)",
      "description": "Feature rules."
    }
  ],
  "subclasses": [
    {
      "name": "Ghost Path",
      "description": "Subclass summary.",
      "features": [
        {
          "name": "Phase Step (Level 3)",
          "description": "Subclass feature rules."
        }
      ]
    }
  ]
}
```

## Classes and Subclasses

Each class can contain its subclasses in a `subclasses` array. To add a
subclass to an existing class, either edit that class or add an entry to
`subclasses/<filename>.json`. The separate-file format is a subclass object
with an additional `class` field containing the exact parent class name, as in
this example:

```json
{
  "class": "Example Operative",
  "name": "Ghost Path",
  "description": "Subclass summary."
}
```

The editor portal's Subclasses category uses this format. Older flat files
(separate `"... Subclasses"` entries) can be converted with
`scripts/migrate_subclasses.py <classes.json>`.

## Reference Data and Renaming

The `_reference/` folder contains the class name mapping and class spell lists
used for spell filtering and class pages.

Class entries may also contain a `uid` and a `levels` array. Each level entry
has a numeric `level`, an optional `hitdie` such as `1d6`, and `skills` using
the same nested feature format as the class's existing `features`. The guide
and character-options API expose these entries. Missing level entries mean the
progression has not been authored; they do not mean the class gains nothing.
Existing `features` and `subclasses` remain supported. This format does not yet
define machine-readable choices. The current level-up dialog is preview-only;
applying both single-class and multiclass levels is disabled until the
progression and choice effects can be handled safely.

For preparing fuller class data, see
[`class-progression-template.json`](_reference/classes/class-progression-template.json)
and the [class progression authoring guide](../../../developernotes/CLASS_PROGRESSION.md).
It is a **proposed authoring worksheet**, not an app-supported schema. Keep it
outside the live `classes/` folder: the application will ignore unsupported
fields, and the proposed `levels[].features` differs from the currently
supported `levels[].skills`. Filling it in does not enable multiclassing.

[`classesv2/`](classesv2/) contains draft worksheets based on the supplied
2024-style reference classes. They are not active class entries; the loader
reads JSON in the folder but does not use it as the `classes` category. The
new worksheets defer subclasses and retain unresolved rules and choices for
review. Awakened Mutant has a 2024-style single-class preview and spellcasting
pilot, but its choices are still manual. The multiclass eligibility and slot
calculator use 2024-style rules; applying mixed builds on the sheet is not
yet supported.

Copy the worksheet outside the active guide data directory for each class:

1. Replace the class ID, name and original class name, and note the rules edition. Keep IDs stable;
   worksheet IDs are proposed identifiers, not current app lookups.
2. Record the hit die and multiclass ability prerequisites for entering and
   leaving the class. Each `any_of` item is an alternative; every ability in
   its `all_of` must be satisfied.
3. Separate starting-class proficiencies and equipment from proficiencies gained
   when taking the class later. Record fixed grants, choices, eligible options,
   how many to choose, and restrictions on duplicates.
4. Describe casting ability, spell list, casting kind, preparation, replacement
   rules and any restrictions. In each level's spell progression, `*_total`
   fields are totals at that class level; `spellbook_spells_gained` is newly
   gained at that level. `class_spell_slots` is an array of nine counts for
   spell levels 1–9. Pact slots are separate. Mystic Arcanum and similar
   features are choices, not ordinary slots. The shared multiclass slot table
   is a separate calculation, not the sum of those class slot arrays.
5. For all class levels 1–20, record every feature gained or improved, including
   repeated gains, subclass selection, resources and choices. Describe each
   choice's options, count, prerequisites, replacement rules and effects.
   Feature descriptions alone are not machine-readable rules.
6. Describe each subclass's selection level and level-specific features,
   choices and spellcasting overrides. Record resource maximum and recharge
   rules, specifying class level, total level, ability modifiers or other inputs.
7. Leave unknown values as `null` and levels marked `unfilled`. Use empty
   arrays only after confirming there are no grants or choices; mark a level
   `complete` only after review. Remove sample placeholders that do not apply.

The worksheet's choice kinds, effects, expressions and subclass progression
are proposals; validation and migration must be implemented before the app
can use them.

When renaming a class, also update its `reskinned_class` in
`_reference/class-and-spell-reskins.json`. Character sheet spellcasting and
class-resource lookups depend on this mapping. If you have separate subclass
entries in `subclasses/*.json`, update their `class` value to the new exact
class name so they remain attached.

Saved characters using the old class name may need to be updated. Class page
links and anchors based on the old name can change, and mentions or links in
other guide content are not rewritten automatically.

Renaming a subclass can change its page link or anchor and may require
updating saved characters or manual references. Subclass spellcasting behavior
may also depend on the subclass name.

## Reloading and Resetting

If a JSON file is invalid, it is skipped for that reload and the last good
data stays live; the error is written to the server log.

This folder is seeded from the built-in copy only when it is empty. To reset
to defaults, delete the folder's contents and restart the application.
