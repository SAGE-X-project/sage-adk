#!/usr/bin/env python3
"""Require >=90% statement coverage for every new production source file."""
from collections import defaultdict
from pathlib import Path
import sys

files = defaultdict(lambda: [0, 0])
for row in Path(sys.argv[1]).read_text().splitlines()[1:]:
    location, statements, count = row.split()
    name = location.split(":", 1)[0].rsplit("/", 1)[-1]
    total = int(statements)
    files[name][0] += total
    files[name][1] += total if int(count) > 0 else 0
expected = {"binding.go", "card.go", "json.go", "schema.go", "signature.go"}
if not expected.issubset(files):
    raise SystemExit("coverage is missing a production file")
for name, (total, covered) in sorted(files.items()):
    if total == 0 or covered * 100 < total * 90:
        raise SystemExit(f"{name}: {covered}/{total} statements; 90% required")
    print(f"{name}: {covered}/{total} statements ({covered * 100 / total:.2f}%)")
