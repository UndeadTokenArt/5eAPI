# Class Progression Authoring Guide

This guide explains how to fill in
[class-progression-template.json](../static/data/neotokyo/_reference/classes/class-progression-template.json)
for a NeoTokyo class. The goal is to collect enough information for a future
level-up and multiclass system to know **what a character gains, what they must
choose, and how the choice affects their sheet**.

> **Important:** The template is a proposed worksheet, not an implemented
> application schema. Filling it out does **not** make multiclassing work.
> Keep completed worksheets outside the live `static/data/neotokyo/classes/`
> folder until the application has a validated schema and a migration path.
> In particular, the worksheet's `levels[].features` is not the same as the
> currently supported `levels[].skills`. Unknown fields may be silently ignored
> if a worksheet is placed in the live folder.

The current app previews levels and has a multiclass spellcasting calculator,
but it does not yet apply levels from the sheet. Some spellcasting data also lives in Go and separate reference files,
not in the class JSON. This worksheet is intended to prepare the data for
replacing or connecting those pieces later.

## 1. Make a working copy

For example, from the repository root:

```bash
mkdir -p developernotes/class-worksheets
cp static/data/neotokyo/_reference/classes/class-progression-template.json \
  developernotes/class-worksheets/cyber-mage.json
```

Edit the copy, not the template or the live class file. Start with one class,
review its existing guide entry and reference files, and fill in the worksheet
using the chosen **2024-style** rules, with NeoTokyo names and any documented
setting-specific changes. The multiclass eligibility and slot calculator use
the 2024 primary abilities and class spell tables, but the character sheet
cannot yet apply multiclass levels.

The worksheets in [`classesv2/`](../static/data/neotokyo/classesv2/) instead
use the supplied **2024-style** class reference files. They are drafts for
review and are not yet a complete app-supported multiclass schema. Cyber-Berserker
/ Aug-Brute is the Barbarian worksheet and Corporate Enforcer is the Fighter
worksheet. Subclass automation remains incomplete. Artificer is omitted because its source material was not supplied.
The Cleric worksheet uses the mapping's theme because no corresponding live
reskin file was supplied.
The level-up dialog previews worksheets for available guide classes, but does
not apply levels or choice effects while the implementation is incomplete. See
[Character Sheets](../CHARACTER_SHEETS.md#awakened-mutant-2024-pilot) for
the current behavior and limitations.

Examples below are **fragments**: replace the matching portion of your worksheet,
not the entire file. Replace example IDs, names, options and rules with the
ones that actually apply to the class. They illustrate how to record
information; they are not a complete rules reference or a guarantee that a
particular class has those options.

## 2. JSON basics

- Text is in double quotes: `"name": "Cyber-Mage"`.
- Numbers are not quoted: `"level": 2`.
- `null` means **unknown or not filled in yet**.
- `[]` means **reviewed and confirmed to have no entries**. It is not the
  same as `null`.
- Lists use `[...]`; objects use `{...}`. Put commas between items, but not
  after the last item in a list or object.
- A `null` choice or subclass list means it has not yet been structured, not
  that there are no choices or subclasses.
- JSON does not allow comments. Use the template's `description` or `notes`
  fields for explanations.
- For a field that does not apply, explain that in `notes` and use the empty
  or neutral value appropriate to the field. The worksheet has no implemented
  validator yet, so the final representation may change.

Initially leave each level's `"status": "unfilled"`. Change it to `"complete"`
only after checking the entire level, including choices and spell progression.
These are authoring markers, not app-enforced statuses.

You can check that your file is valid JSON:

```bash
python3 -m json.tool developernotes/class-worksheets/cyber-mage.json >/dev/null
```

No output means the syntax is valid. This does **not** check whether the rules
are correct or whether the app understands the fields.

## 3. Class identity and hit die

Fill in the first fields. The example uses the die's number of sides as an
integer: `6` represents **1d6**; do not enter a rolled hit point result here.

```json
{
  "schema_status": "draft_authoring_template_not_supported_by_app",
  "schema_version": 1,
  "uid": "class-cyber-mage",
  "name": "Cyber-Mage",
  "original_class": "Wizard",
  "description": "A practitioner who uses a cyberdeck to work digital magic.",
  "hit_die": 6
}
```

- `uid`: choose a unique, stable identifier. Do not reuse it for another class;
  do not change it just because the display name changes.
- `name`: the exact name used by the current class entry.
- `original_class`: the class used as its rules basis, if applicable. Identify
  the rules edition in your source notes. If
  the class is custom and has no direct equivalent, explain that rather than
  forcing an incorrect mapping.
- `hit_die`: die sides per class level, not total hit dice or maximum HP.
  Hit points also depend on Constitution and the player's average-or-roll
  choice. The current app gets hit dice from code rather than this worksheet.

Worksheet IDs are proposed identifiers; current character and spell lookups
still depend on class names and mappings.

## 4. Multiclass prerequisites

Use `any_of` for **alternative** ways to qualify and `all_of` for requirements
that must be met **together**. This hypothetical example means either
Intelligence 13 **or** both Dexterity 13 and Wisdom 13:

```json
"multiclass_prerequisites": {
  "description": "Meet either of the listed alternatives.",
  "any_of": [
    { "all_of": [{ "ability": "intelligence", "minimum": 13 }] },
    { "all_of": [
      { "ability": "dexterity", "minimum": 13 },
      { "ability": "wisdom", "minimum": 13 }
    ] }
  ]
}
```

Use ability names such as `strength`, `dexterity`, `constitution`,
`intelligence`, `wisdom`, and `charisma`. Verify the actual requirements for
each class; the example is not a Cyber-Mage rule. Under the 2024
rules, eligibility may depend on both the class being left and the class being
entered. Record each class's own requirements; the current preview checks them
across the character, but does not apply the level. Do not assume a missing requirement
means no prerequisite.

## 5. Starting versus multiclass grants

There are two different situations:

1. **Starting class:** the character begins at character level 1 in this class.
2. **Multiclass entry:** an existing character takes their first level in this
   class later.

Fill `starting_proficiencies` for the first situation and
`multiclass_proficiencies` for the second. Record armor, weapons, tools, saving
throws, skills and choices separately. Do not copy all starting proficiencies
into the multiclass section without checking.

Illustrative fragments:

```json
"starting_proficiencies": {
  "armor": [],
  "weapons": ["Example weapon"],
  "tools": [],
  "saving_throws": ["intelligence", "wisdom"],
  "skills": [],
  "choices": [{
    "id": "starting-skills",
    "name": "Choose starting skills",
    "kind": "skill",
    "count": 2,
    "allowed_options": ["Cyber-Arcana", "Data Archives", "System Forensics"],
    "eligibility_rule": "Choose distinct skills; verify the complete class list.",
    "selection_effect": "Mark the selected skills proficient."
  }]
}
```

```json
"multiclass_proficiencies": {
  "armor": [],
  "weapons": [],
  "tools": [],
  "skills": [],
  "choices": []
}
```

These are example values, **not an authoritative grant list**. The structure
used for choices in these arrays is a proposed convention. Spell out what
happens when a character already has a proficiency instead of guessing that
they receive a replacement.

For `starting_equipment`, record fixed items and alternatives separately.
Starting equipment is not automatically granted to a character taking the
class later. The worksheet does not yet define a formal structure for item
quantities or alternative bundles: describe them clearly in `description` and
`choices` until the schema supports them.

## 6. Record level-by-level features and choices

The `levels` list represents **levels in this class**, not the character's
combined level. A character with 3 levels in one class and 2 in another is
character level 5, but uses the class-level 3 and class-level 2 entries
respectively.

For each class level, list what is **new or improved at that level**. If a
feature improves at several levels, include an entry at each relevant level
and describe the improvement. Include subclass selection and ability-score
improvements where applicable; do not assume the app will derive them from
prose or from the first time a feature appears.

An illustrative feature with a choice:

```json
{
  "level": 2,
  "status": "unfilled",
  "features": [
    {
      "id": "example-subroutine",
      "name": "Example Subroutine",
      "description": "Choose one specialization for this class.",
      "when": "both",
      "choices": [
        {
          "id": "example-subroutine-choice",
          "name": "Choose a specialization",
          "kind": "subclass",
          "count": 1,
          "allowed_options": ["Example Path A", "Example Path B"],
          "eligibility_rule": "Choose one path available to this class.",
          "selection_effect": "Record the chosen subclass and its level-2 feature."
        }
      ]
    }
  ],
  "spell_progression": null,
  "notes": "Verify this entry against the intended class rules."
}
```

`when` is proposed as `starting`, `multiclass`, or `both`: use it to identify
whether a grant applies only when the class is the character's first class,
only on entering it later, or in both cases. For later class levels, `both`
normally describes a feature that applies regardless of how the class was
acquired. Check each rule rather than relying on that assumption.

For each choice, record:

- A stable `id` and a clear prompt in `name`.
- Its proposed `kind`, such as `skill`, `subclass`, `spell`, `feat`, or
  `ability_score`. These labels are examples, not implemented enum values.
- `count`, the number selected, and `allowed_options`, the actual eligible
  values or a clearly described source.
- Requirements, duplicates, replacement or swap rules, and whether choices
  depend on earlier selections.
- `selection_effect`: what should be saved or changed on the sheet. Include
  granted proficiencies, uses, scores, or other effects where relevant.

Some choices need alternatives rather than a single fixed count—for example,
an ability-score improvement *or* a feat. Describe each branch and its
constraints in the rule text for now. A formal, machine-readable alternative
and effects format has not been implemented.

## 7. Spellcasting and slots

Fill in the class-level `spellcasting` object first:

- `kind`: use `none`, `full`, `half`, `third`, or `pact` as worksheet labels
  when they fit. Describe exceptions. A subclass may provide or change
  spellcasting even if its parent class does not cast.
- `ability`: the casting ability, or `null` for a noncaster.
- `starts_at_class_level`: the class level at which this progression starts.
- `spell_list_id`: identify the intended source list and make sure its entries
  can be matched to the guide's spell data. IDs here are not wired up yet.
- `casting_mode`, `preparation_rule`, `ritual_casting_rule`, and `focus_rule`:
  describe how this class learns, prepares and casts its spells. If the class
  uses a spellbook, explain both the book and preparation.
- `multiclass_slot_rule`: describe how the class contributes to shared slots,
  including rounding and any exception. Pact Magic is a separate pool.

At each class level, fill `spell_progression`. Example for an illustrative
single-class caster at class level 3:

```json
"spell_progression": {
  "cantrips_known_total": 3,
  "spells_known_total": null,
  "prepared_spells_total": null,
  "spellbook_spells_gained": 2,
  "max_class_spell_level": 2,
  "class_spell_slots": [4, 2, 0, 0, 0, 0, 0, 0, 0],
  "pact_slot_count": null,
  "pact_slot_level": null,
  "choices_and_replacements": "Record which spells can be added, prepared, or replaced."
}
```

`*_total` means the **total at that class level**, not the number newly
gained. `spellbook_spells_gained` means the number gained **at that level**.
`prepared_spells_total` is the number prepared from the class feature, distinct
from spells known and from spells that are always prepared without counting
against that limit. The supplied 2024-style class tables use prepared totals.
The nine slot counts are for spell levels 1 through 9, respectively. Use
`null` for an unknown or not-applicable number and explain which in `notes`;
use `0` when the value is known to be zero.

The class's own spell slots are useful for its class progression, but a
multiclass character's shared Spellcasting slots are calculated separately.
Do **not** add the slot arrays together. Pact slots remain separate and recover
according to their own rules. A high-level special spell such as Mystic
Arcanum is a separate choice/use, not an ordinary spell slot. Also distinguish
**slot level** from the highest spell level the character may choose for a
particular class.

When recording spell choices, include class list, spell level, how many can be
selected, whether it is known, prepared, or in a spellbook, and replacement
rules. The app already has reference data in
[`_reference/`](../static/data/neotokyo/_reference/) and spell entries in
[`spells/`](../static/data/neotokyo/spells/); the worksheet does not yet
automatically connect to them.

## 8. Resources and subclasses

A resource describes something tracked in limited uses or points. For example:

```json
"resources": [{
  "id": "example-charge",
  "name": "Example Charge",
  "description": "A limited-use class feature.",
  "recharge": "long",
  "maximum_rule": "Maximum equals this class's level; regain all uses on a long rest."
}]
```

Specify whether a maximum uses **class level**, **total character level**, an
ability modifier, proficiency bonus, or a fixed table. Also record minimums,
what spending does, when it recharges and whether an increase restores spent
uses. `maximum_rule` is explanatory text today, not an executable formula.
Existing tracked resources currently come from a separate reference file.

For every subclass, record a stable ID, display name, summary, the class level
at which it is chosen, features and choices by class level, and any changes to
spells or lists. For example:

```json
"subclasses": [{
  "id": "example-path-a",
  "name": "Example Path A",
  "description": "An example specialization.",
  "selected_at_class_level": 2,
  "spellcasting_override": null,
  "levels": [
    {
      "level": 2,
      "status": "unfilled",
      "features": [{
        "id": "example-path-a-feature",
        "name": "Example Feature",
        "description": "Describe what this subclass grants at class level 2.",
        "when": "both",
        "choices": []
      }]
    }
  ]
}]
```

This subclass `levels` shape is an illustrative proposal, not current app
behavior. Record future subclass features at the class levels where they are
gained, not all at the selection level. If a feature grants additional spells,
specify whether they are always prepared, count against limits, or use a
particular class's casting ability.

## 9. Review checklist

- [ ] A unique class ID, exact current name, rules basis, and hit die are recorded.
- [ ] Entry and exit multiclass prerequisites are researched.
- [ ] Starting proficiencies and equipment are separate from multiclass grants.
- [ ] All class levels 1–20 are reviewed; unknowns remain `null`/`unfilled`.
- [ ] Repeated improvements, subclass gains and ability-score/feat choices are
      recorded at each applicable class level.
- [ ] Choices state count, options, eligibility, replacement and sheet effects.
- [ ] Spell totals, newly gained spells, ordinary slots and pact slots are not
      confused; prepared, known and spellbook spells are distinguished.
- [ ] Resources specify their maximum and recovery rules.
- [ ] Subclass selection, later features and spellcasting exceptions are covered.
- [ ] JSON syntax is valid, and placeholders have been replaced or removed.

After review, keep the worksheet as an authoring document. The next development
step is to agree on and implement the formal schema and validation, migrate
the worksheets, and build the multiclass sheet and level-up behavior. A valid, complete
worksheet by itself will not enable level application.
