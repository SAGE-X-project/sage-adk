//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func fixtureDescriptors(t *testing.T, dir, epoch string) (string, string) {
	t.Helper()
	rules := sha256.Sum256([]byte(`{"tool":"calculator"}`))
	component := sha256.Sum256([]byte("calculator"))
	policy := `{"artifacts":{"files":[{"path":"rules.json","sha256":"` + hex.EncodeToString(rules[:]) + `"}],"version":"0.10.0"},"engine":"sage-adk/exact-operation/1","epoch":"` + epoch + `","issuer":"did:sage:web:agent.example:alice","version":"0.10.0"}`
	manifest := `{"files":[{"path":"calculator.json","sha256":"` + hex.EncodeToString(component[:]) + `"}],"version":"0.10.0"}`
	return write(t, filepath.Join(dir, "policy-"+epoch+".json"), policy), write(t, filepath.Join(dir, "manifest.json"), manifest)
}

// The built command runs as separate processes for keygen, sign and verify.
func TestApproveCommandProcesses(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "adk-approve")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	key := filepath.Join(dir, "operator.seed")
	out, err := exec.Command(bin, "keygen", "-key", key).Output()
	if err != nil {
		t.Fatal(err)
	}
	approver := strings.TrimSpace(string(out))
	policy, manifest := fixtureDescriptors(t, dir, "00000000-0000-4000-8000-000000000001")
	approval := filepath.Join(dir, "approval.json")
	if out, err = exec.Command(bin, "sign", "-key", key, "-policy", policy, "-manifest", manifest, "-sequence", "1", "-out", approval).CombinedOutput(); err != nil {
		t.Fatalf("sign: %v %s", err, out)
	}
	if out, err = exec.Command(bin, "verify", "-approver", approver, "-policy", policy, "-manifest", manifest, "-approval", approval).CombinedOutput(); err != nil || !bytes.HasPrefix(out, []byte("approval valid sequence=1 ")) {
		t.Fatalf("verify: %v %s", err, out)
	}
	other, _ := fixtureDescriptors(t, dir, "00000000-0000-4000-8000-000000000002")
	if out, err = exec.Command(bin, "verify", "-approver", approver, "-policy", other, "-manifest", manifest, "-approval", approval).CombinedOutput(); err == nil {
		t.Fatalf("changed policy verified: %s", out)
	}
	if out, err = exec.Command(bin, "sign", "-key", key, "-policy", policy, "-manifest", manifest, "-sequence", "2", "-out", approval).CombinedOutput(); err == nil {
		t.Fatalf("existing approval overwritten: %s", out)
	}
}

func TestApproveCommandRefusals(t *testing.T) {
	dir := t.TempDir()
	open := write(t, filepath.Join(dir, "open.seed"), strings.Repeat("a", 32))
	policy, manifest := fixtureDescriptors(t, dir, "00000000-0000-4000-8000-000000000001")
	var out bytes.Buffer
	for name, args := range map[string][]string{
		"no-command":    {},
		"unknown":       {"publish"},
		"keygen-no-key": {"keygen"},
		"sign-missing":  {"sign", "-key", open},
		"sign-open-key": {"sign", "-key", open, "-policy", policy, "-manifest", manifest, "-sequence", "1", "-out", filepath.Join(dir, "a.json")},
		"verify-bad":    {"verify", "-approver", "zz", "-policy", policy, "-manifest", manifest, "-approval", policy},
		"verify-reject": {"verify", "-approver", strings.Repeat("00", 32), "-policy", policy, "-manifest", manifest, "-approval", policy},
		"missing-file":  {"verify", "-approver", strings.Repeat("00", 32), "-policy", filepath.Join(dir, "none"), "-manifest", manifest, "-approval", policy},
	} {
		if err := run(args, &out); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}
