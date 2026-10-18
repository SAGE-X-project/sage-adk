#!/usr/bin/env python3
"""Run synthetic evidence-parser units and measure that parser's executed lines."""
from pathlib import Path
import sys
import trace
import unittest


def run():
    directory = Path(__file__).resolve().parent
    tracer = trace.Trace(count=True, trace=False, ignoredirs=[sys.base_prefix])
    def execute():
        suite = unittest.defaultTestLoader.discover(str(directory), pattern="test_check_evidence.py")
        return unittest.TextTestRunner().run(suite)
    result = tracer.runfunc(execute)
    if not result.wasSuccessful() or result.testsRun != 11:
        raise SystemExit("evidence parser units did not complete")
    source = directory / "check_evidence.py"
    lines = set(trace._find_executable_linenos(str(source)))
    executed = {line for (file, line), count in tracer.results().counts.items() if Path(file).resolve() == source and count}
    covered = len(lines & executed)
    if not lines or covered * 100 < len(lines) * 90:
        raise SystemExit(f"evidence parser: {covered}/{len(lines)} executable lines; 90% required")
    print(f"Synthetic evidence parser units: 11; parser coverage {covered}/{len(lines)} lines")


if __name__ == "__main__":
    run()
