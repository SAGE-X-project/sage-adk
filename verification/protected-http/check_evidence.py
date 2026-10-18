#!/usr/bin/env python3
"""Check bounded HTTP CI provenance; this does not grade arbitrary implementations."""
from pathlib import Path
import hashlib
import json
import re
import sys

ROOT = Path(__file__).resolve().parents[2]
PACKAGE = "github.com/sage-x-project/sage-adk/core/a2abinding"
MODES = {"success", "repeat", "unsupported", "source-after-setup", "expired-after-setup", "cancel-after-setup", "close-after-setup", "source-after-handler", "expired-after-handler", "close-during-handler", "cancel-during-handler"}
GOOD = {"success", "repeat", "unsupported"}
CONTRACT = {
    "Runtime": {m + "/" + h for m in MODES for h in ("bare-completion", "http-completion")},
    "FixedFraming": {"truncated-head", "bare-newline", "duplicate-host", "duplicate-length", "conflicting-length", "folding", "invalid-name", "missing-colon", "missing-name", "control-value", "transfer-encoding", "trailer", "compression", "upgrade", "continue", "proxy", "forwarded", "x-forwarded", "keepalive", "leading-zero", "negative-length", "oversize-length", "nonnumeric-length", "empty-length", "missing-length", "short-body", "large-head", "request/method", "request/version", "request/host", "request/route", "request/absolute-target", "request/startline", "response/invalid-status", "response/unsupported-protocol", "response/nonterminal", "response/no-content", "response/outside-status"},
    "TransportConfiguration": {"plain", "userinfo", "fragment", "implicit-path", "uppercase", "default-port", "opaque", "space", "too-large", "nil", "skip", "connection-callback", "peer-callback", "config-callback", "keylog", "renegotiation", "random", "time", "minimum", "maximum", "h2"},
    "FinalAdmission": {"success", "nil-context", "canceled", "closed", "unavailable-source", "verify-error", "nil-card", "budget", "rollback", "card-expired", "card-key-expired", "wire-expired", "wire-future", "nil-writer", "empty-frame", "large-frame", "short-write", "write-error"},
    "ClockAndCardBounds": set(),
    "ShutdownDuringDial": {"canceled-dial", "late-connection"},
    "DispatchHook": {"once", "missing", "twice", "hook-error", "ignored-verify-error"},
    "DispatchBudgetIncludesVerify": set(),
    "MemoryTLS": {"policy-denial-reuse", "card-unavailable-after-handler"},
    "LocalIngressRefusals": {"consumed-session", "wrong-local", "already-serving", "closed", "closed-connection", "canceled-connection", "deadline-error", "handshake-eof", "empty-response", "unsigned-bad-request", "unsigned-unauthorized"},
    "ConstructionAndClosure": {"nil-session", "wrong-target", "ttl-zero", "ttl-large", "timeout-zero", "timeout-large", "tls-nil", "server-name", "bad-tls"},
}


def source_hashes(root=ROOT):
    paths = json.loads((root / "verification/protected-http/source-files.json").read_text())
    assert isinstance(paths, list) and len(paths) == len(set(paths)) and paths == sorted(paths)
    assert all(isinstance(p, str) and not Path(p).is_absolute() and ".." not in Path(p).parts for p in paths)
    return {p: hashlib.sha256((root / p).read_bytes()).hexdigest() for p in paths}


def check_modules(data, expected):
    decoder = json.JSONDecoder()
    selected = {}
    while data.strip():
        module, end = decoder.raw_decode(data.lstrip())
        data = data.lstrip()[end:]
        assert isinstance(module, dict) and module.get("Path") in expected and "Replace" not in module
        assert module["Path"] not in selected
        selected[module["Path"]] = {k: module[k] for k in ("Version", "Sum", "GoModSum")}
    assert selected == expected
    return selected


def events(path):
    rows = [json.loads(line) for line in path.read_text().splitlines()]
    assert rows and all(r.get("Package") == PACKAGE for r in rows)
    assert not any(r.get("Action") in {"skip", "fail"} for r in rows)
    assert sum(r.get("Action") == "pass" and "Test" not in r for r in rows) == 1
    return rows


def check_tests(focused, full):
    def passed(rows):
        values = [r["Test"] for r in rows if r.get("Action") == "pass" and "Test" in r]
        assert len(values) == len(set(values))
        return set(values)
    passed_full, passed_focus = passed(full), passed(focused)
    expected = set()
    for suite, children in CONTRACT.items():
        top = "TestProtectedHTTP" + suite + "010"
        expected.add(top)
        expected.update(top + "/" + child for child in children)
    assert passed_focus == expected and passed_focus <= passed_full
    # Markers supplement the actual harness PASS set, never replace it.
    markers = {}
    pattern = r"owned-production-http handshake=(true|false) mode=([a-z-]+) handler=(\d+) usable-result=(true|false)"
    for row in focused:
        match = re.search(pattern, row.get("Output", ""))
        if not match:
            continue
        handshake, mode, handler, usable = match.groups()
        branch = "http-completion" if handshake == "true" else "bare-completion"
        assert mode in MODES and row.get("Test") == "TestProtectedHTTPRuntime010/" + mode + "/" + branch
        key = (handshake, mode)
        assert key not in markers
        expected_handler = 2 if mode == "repeat" else int(mode in {"success", "source-after-handler", "expired-after-handler", "close-during-handler", "cancel-during-handler"})
        assert int(handler) == expected_handler and (usable == "true") == (mode in GOOD)
        markers[key] = int(handler)
    assert set(markers) == {(h, m) for h in ("true", "false") for m in MODES}
    leaves = {p for p in passed_focus if not any(q.startswith(p + "/") for q in passed_focus)}
    return {"suites": len(CONTRACT), "leaf_tests": len(leaves), "owned_tls_scenarios": len(markers), "passed_tests": sorted(passed_focus)}


def coverage(path):
    result = {}
    for line in path.read_text().splitlines()[1:]:
        location, statements, count = line.split()
        name = location.rsplit(":", 1)[0].rsplit("/", 1)[-1]
        if name not in {"http_transport.go", "http_ingress.go"}:
            continue
        total, covered = result.get(name, (0, 0))
        n = int(statements)
        result[name] = (total + n, covered + (n if int(count) else 0))
    assert set(result) == {"http_transport.go", "http_ingress.go"}
    assert all(total and covered * 100 >= total * 90 for total, covered in result.values()), result
    return {k: {"statements": t, "covered": c} for k, (t, c) in result.items()}


def main(stage, directory):
    out = Path(directory)
    if stage == "sources":
        (out / "sources.json").write_text(json.dumps(source_hashes(), indent=2) + "\n")
    elif stage == "unchanged":
        assert json.loads((out / "sources.json").read_text()) == source_hashes()
    elif stage == "modules":
        expected = json.loads((ROOT / "verification/library-consumer/did-dependencies.json").read_text())
        check_modules((out / "modules.json").read_text(), expected)
    elif stage == "tests":
        report = check_tests(events(out / "tests.jsonl"), events(out / "all-tests.jsonl"))
        report["focused_coverage"] = coverage(out / "focused.cover")
        report["full_coverage"] = coverage(out / "all.cover")
        report["scope"] = "bounded owned TLS and unit integration, fixture authority; no operating Registry or historical Inspector closure"
        (out / "summary.json").write_text(json.dumps(report, indent=2) + "\n")
        print(json.dumps({k: v for k, v in report.items() if k != "passed_tests"}, indent=2))
    else:
        raise ValueError("unknown evidence stage")


if __name__ == "__main__":
    main(*sys.argv[1:])
