"""Evidence parser units use synthetic reports, never claim protocol execution."""
import contextlib
import io
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location("protected_http_check", Path(__file__).with_name("check_evidence.py"))
check = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(check)


def good_rows():
    rows = []
    for suite, children in check.CONTRACT.items():
        top = "TestProtectedHTTP" + suite + "010"
        for name in [top] + [top + "/" + child for child in sorted(children)]:
            rows.extend([{"Package": check.PACKAGE, "Action": "run", "Test": name}, {"Package": check.PACKAGE, "Action": "pass", "Test": name}])
    for handshake in ("true", "false"):
        for mode in sorted(check.MODES):
            handler = 2 if mode == "repeat" else int(mode in {"success", "source-after-handler", "expired-after-handler", "close-during-handler", "cancel-during-handler"})
            rows.append({"Package": check.PACKAGE, "Action": "output", "Test": "TestProtectedHTTPRuntime010/" + mode + "/" + ("http-completion" if handshake == "true" else "bare-completion"), "Output": f"owned-production-http handshake={handshake} mode={mode} handler={handler} usable-result={str(mode in check.GOOD).lower()}"})
    rows.append({"Package": check.PACKAGE, "Action": "pass"})
    return rows


class EvidenceTests(unittest.TestCase):
    def test_complete_harness_and_marker_contract(self):
        rows = good_rows()
        result = check.check_tests(rows, rows)
        self.assertEqual((result["suites"], result["leaf_tests"], result["owned_tls_scenarios"]), (10, 127, 22))

    def test_missing_duplicate_or_false_observations(self):
        rows = good_rows()
        candidates = [rows[:-2], rows + [rows[1]], rows[2:]]
        changed = json.loads(json.dumps(rows))
        next(r for r in changed if "Output" in r)["Output"] = "owned-production-http handshake=true mode=success handler=0 usable-result=true"
        candidates.append(changed)
        candidates.append(rows + [next(r for r in rows if "Output" in r)])
        for candidate in candidates:
            with self.assertRaises(AssertionError):
                check.check_tests(candidate, rows)

    def test_runtime_marker_matches_actual_handshake_branch(self):
        rows = good_rows()
        changed = json.loads(json.dumps(rows))
        for row in changed:
            if "Output" in row:
                row["Output"] = row["Output"].replace("handshake=true", "handshake=SWAP").replace("handshake=false", "handshake=true").replace("handshake=SWAP", "handshake=false")
        self.assertEqual([r.get("Test") for r in changed], [r.get("Test") for r in rows])
        with self.assertRaises(AssertionError):
            check.check_tests(changed, rows)

    def test_event_fail_skip_wrong_subject(self):
        with tempfile.TemporaryDirectory() as directory:
            p = Path(directory, "events.jsonl")
            rows = good_rows()
            p.write_text("\n".join(json.dumps(r) for r in rows))
            self.assertEqual(check.events(p), rows)
            for extra in [{"Package": check.PACKAGE, "Action": "fail"}, {"Package": check.PACKAGE, "Action": "skip"}, {"Package": "different", "Action": "output"}]:
                p.write_text("\n".join(json.dumps(r) for r in rows + [extra]))
                with self.assertRaises(AssertionError):
                    check.events(p)

    def test_module_identity_no_replacement(self):
        expected = {"core": {"Version": "v1.2.3", "Sum": "h1:fixture", "GoModSum": "h1:mod"}}
        module = {"Path": "core", **expected["core"]}
        self.assertEqual(check.check_modules(json.dumps(module), expected), expected)
        for data in ["", json.dumps(module) * 2, json.dumps({**module, "Replace": {}}), json.dumps({**module, "Sum": "wrong"}), json.dumps({**module, "Path": "different"})]:
            with self.assertRaises(AssertionError):
                check.check_modules(data, expected)

    def test_sources_scope_and_coverage(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            manifest = root / "verification/protected-http/source-files.json"
            manifest.parent.mkdir(parents=True)
            (root / "source.go").write_text("owned fixture source")
            manifest.write_text(json.dumps(["source.go"]))
            self.assertEqual(len(check.source_hashes(root)), 1)
            for invalid in [["source.go", "source.go"], ["../source.go"], ["/source.go"]]:
                manifest.write_text(json.dumps(invalid))
                with self.assertRaises(AssertionError):
                    check.source_hashes(root)
            p = root / "profile"
            good = "mode: atomic\nfixture/http_transport.go:1.1,2.1 10 1\nfixture/http_ingress.go:1.1,2.1 10 1\nfixture/other.go:1.1,2.1 1 0\n"
            p.write_text(good)
            self.assertEqual(len(check.coverage(p)), 2)
            p.write_text(good.replace("10 1", "10 0"))
            with self.assertRaises(AssertionError):
                check.coverage(p)
            p.write_text("mode: atomic\n")
            with self.assertRaises(AssertionError):
                check.coverage(p)

    def test_cli_stage_contracts(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "verification/protected-http").mkdir(parents=True)
            (root / "verification/library-consumer").mkdir(parents=True)
            (root / "verification/protected-http/source-files.json").write_text(json.dumps(["source.go"]))
            (root / "source.go").write_text("source")
            expected = {"core": {"Version": "v1", "Sum": "h1:fixture", "GoModSum": "h1:mod"}}
            (root / "verification/library-consumer/did-dependencies.json").write_text(json.dumps(expected))
            out = root / "output"
            out.mkdir()
            previous = check.ROOT
            check.ROOT = root
            original_hashes = check.source_hashes
            check.source_hashes = lambda: original_hashes(root)
            try:
                check.main("sources", out)
                check.main("unchanged", out)
                (root / "source.go").write_text("changed")
                with self.assertRaises(AssertionError):
                    check.main("unchanged", out)
                (out / "modules.json").write_text(json.dumps({"Path": "core", **expected["core"]}))
                check.main("modules", out)
                rows = "\n".join(json.dumps(r) for r in good_rows())
                for name in ("tests.jsonl", "all-tests.jsonl"):
                    (out / name).write_text(rows)
                for name in ("focused.cover", "all.cover"):
                    (out / name).write_text("mode: atomic\nf/http_transport.go:1.1,2.1 10 1\nf/http_ingress.go:1.1,2.1 10 1\n")
                with contextlib.redirect_stdout(io.StringIO()):
                    check.main("tests", out)
                self.assertEqual(json.loads((out / "summary.json").read_text())["leaf_tests"], 127)
                with self.assertRaises(ValueError):
                    check.main("invalid", out)
            finally:
                check.ROOT = previous
                check.source_hashes = original_hashes


if __name__ == "__main__":
    unittest.main()
