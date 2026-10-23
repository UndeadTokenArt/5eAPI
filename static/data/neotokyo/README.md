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
