#!/usr/bin/env python3
"""Temporary safe-fixture diagnostics; preserves every returned core error."""
import os
import pathlib
import re
import shutil
import subprocess

root = pathlib.Path(os.environ["RUNNER_TEMP"])
source = pathlib.Path(subprocess.check_output(
    ["go", "list", "-m", "-f", "{{.Dir}}", "github.com/sage-x-project/sage"],
    text=True).strip())
target = root / "native-hop-trace-core"
shutil.copytree(source, target)
for directory, _, _ in os.walk(target):
    pathlib.Path(directory).chmod(0o700)
for package in ("guard010", "registry010", "hpke"):
    directory = target / "pkg" / "agent" / package
    for path in directory.glob("*.go"):
        if path.name.endswith("_test.go"):
            continue
        path.chmod(0o600)
        lines = []
        for line in path.read_text().splitlines(keepends=True):
            if line.lstrip().startswith("return "):
                line = re.sub(
                    r"(?<![\w.])((?:[A-Za-z_]\w*\.)?(?:Err[A-Za-z0-9_]+|err[A-Za-z0-9_]*|e))\s*$",
                    lambda match: "traceError(" + match[1] + ")\n", line)
            lines.append(line)
        path.write_text("".join(lines))
    (directory / "trace_diagnostic.go").write_text('''package ''' + package + '''
import ("fmt"; "os"; "runtime")
func traceError(err error) error {
    if err != nil {
        pc, file, line, _ := runtime.Caller(1)
        name := "unknown"
        if f := runtime.FuncForPC(pc); f != nil { name = f.Name() }
        fmt.Fprintf(os.Stderr, "FIXTURE_TRACE %s %s:%d: %v\\n", name, file, line, err)
    }
    return err
}
''')
modfile = root / "native-hop-trace.mod"
modfile.write_text(pathlib.Path("go.mod").read_text() +
                   "\nreplace github.com/sage-x-project/sage => " + str(target) + "\n")
shutil.copyfile("go.sum", modfile.with_suffix(".sum"))
subprocess.run(["go", "test", "-mod=readonly", "-modfile=" + str(modfile),
                "-race", "-timeout", "5m", "-count", "10", "-v",
                "./core/toolhost", "-run", "TestApprovedHopOperationNativeRuntime"],
               check=True)
