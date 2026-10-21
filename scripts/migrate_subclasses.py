#!/usr/bin/env python3
"""Nest subclasses under their parent class in a classes JSON file.

Old layout: a flat list where each base class is followed by a
"... Subclasses" group entry (one feature per subclass) and sometimes by
numbered standalone entries ("2. College of ...") that are more subclasses.

New layout: every class has a "subclasses" array holding its subclasses.

Usage: migrate_subclasses.py <classes.json> [more files...]   (rewrites in place)
Files that are already nested are left untouched.
"""
import json
import re
import sys

NUMBER = re.compile(r"^\d+\.\s*")


def is_group(name):
    return "subclass" in name.lower()


def clean(name):
    return NUMBER.sub("", name).strip()


def from_group_feature(feature):
    sub = {"name": clean(feature["name"])}
    for key in ("description", "focus", "details"):
        if feature.get(key):
            sub[key] = feature[key]
    if feature.get("features"):
        sub["features"] = feature["features"]
    # "content" flattens the structured features, so it is only kept when nothing else holds the text.
    elif feature.get("content"):
        sub["content"] = feature["content"]
    return sub


def from_standalone(entry):
    sub = {"name": clean(entry["name"])}
    for key in ("description", "focus", "defensive_flourish", "slashing_flourish", "mobile_flourish", "details", "content", "features"):
        if entry.get(key):
            sub[key] = entry[key]
    return sub


def migrate(classes):
    out = []
    for entry in classes:
        if is_group(entry["name"]) and out:
            out[-1].setdefault("subclasses", []).extend(from_group_feature(f) for f in entry.get("features", []))
        elif NUMBER.match(entry["name"]) and out and out[-1].get("subclasses"):
            out[-1]["subclasses"].append(from_standalone(entry))
        else:
            out.append(entry)
    return out


def main(paths):
    for path in paths:
        with open(path, encoding="utf-8") as handle:
            data = json.load(handle)
        if not isinstance(data, list) or any("subclasses" in c for c in data):
            print(f"{path}: skipped (already nested or not a list)")
            continue
        result = migrate(data)
        with open(path, "w", encoding="utf-8") as handle:
            json.dump(result, handle, indent=2, ensure_ascii=False)
            handle.write("\n")
        count = sum(len(c.get("subclasses", [])) for c in result)
        print(f"{path}: {len(data)} entries -> {len(result)} classes, {count} subclasses")


if __name__ == "__main__":
    main(sys.argv[1:])
