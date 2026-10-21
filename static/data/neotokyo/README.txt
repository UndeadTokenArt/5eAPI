NeoTokyo guide data
===================

One folder per category. Every *.json file inside a category folder is read
(in filename order) and its entries are added to that category. A file holds
either a JSON array of entries or a single entry object, in the same shape as
the existing entries. To add something, drop a new file next to the others,
e.g. spells/my-homebrew-spells.json. Edits are picked up within a couple of
seconds, with no restart or rebuild.

Categories (folder names): abilities, backgrounds, species, classes, feats,
adventuring-gear-and-containers, tools-kits-and-consumables, ammunition,
weapons, armor-and-shields, mounts-vehicles-and-harnesses,
foci-and-instruments, spells, conditions.

Notes
- Classes: each class holds its subclasses in a "subclasses" array. To add a
  subclass to an existing class, either edit that class, or drop an entry
  into subclasses/<any>.json shaped like a subclass plus "class": "<exact
  class name>" (the editor portal's Subclasses category does this).
  Older flat files (separate "... Subclasses" entries) can be converted with
  scripts/migrate_subclasses.py <classes.json>.
- _reference/ holds the class name mapping and class spell lists used for
  spell filtering and the spell lists on class pages.
- A file with invalid JSON is skipped for that reload and the last good data
  stays live; the error is written to the server log.
- This folder is only seeded from the built-in copy when it is empty. To
  reset to defaults, delete the folder's contents and restart.
