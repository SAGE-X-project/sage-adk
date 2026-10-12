#!/usr/bin/env python3
"""Extract field/presence tables from the exact adopted A2A protobuf bytes."""
import hashlib
import json
from pathlib import Path
import re
import sys
PIN = "e195bf96ab630c69797851970203e1b2b6b19528f2e9803b7d904b91a5104016"
data = Path(sys.argv[1]).read_bytes()
if hashlib.sha256(data).hexdigest() != PIN:
    raise SystemExit("A2A protobuf revision differs from adopted basis")
text = re.sub(r"//[^\n]*", "", data.decode())
messages = {}
for match in re.finditer(r"\bmessage\s+(\w+)\s*\{", text):
    i = match.end(); depth = 1; end = i
    while depth:
        depth += (text[end] == "{") - (text[end] == "}"); end += 1
    fields = []; oneof = ""
    for line in text[i:end - 1].splitlines():
        line = line.strip()
        if line.startswith("oneof "):
            oneof = line.split()[1]; continue
        if line == "}":
            oneof = ""; continue
        field = re.match(r"(?:(optional|repeated)\s+)?(map\s*<\s*string\s*,\s*[\w.]+\s*>|[\w.]+)\s+(\w+)\s*=\s*\d+(.*);", line)
        if not field:
            continue
        flag, typ, name, annotation = field.groups(); kind = "scalar"
        if typ.startswith("map"):
            typ = re.search(r",\s*([\w.]+)", typ)[1]; kind = "map"
        elif flag == "repeated":
            kind = "repeated"
        fields.append({"name": name, "json": re.sub(r"_([a-z])", lambda m: m[1].upper(), name), "type": typ, "kind": kind,
                       "required": "REQUIRED" in annotation, "optional": flag == "optional", "oneof": oneof})
    messages[match[1]] = fields
roots = ["AgentCard", "SendMessageRequest", "GetTaskRequest", "ListTasksRequest", "CancelTaskRequest"]
out = {}; queue = list(roots)
while queue:
    name = queue.pop()
    if name in out:
        continue
    out[name] = messages[name]
    queue.extend(f["type"] for f in out[name] if f["type"] in messages)
Path(sys.argv[2]).write_text(json.dumps(out, sort_keys=True, indent=2) + "\n")
print(f"Pinned schema: {len(out)} messages, {sum(map(len, out.values()))} fields")
