//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package capture_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sage-x-project/sage-adk/core/capture"
)

const fileID = "00000000-0000-4000-8000-000000000001"

func privateDirectory(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "originals")
	return dir
}
func TestFileStoreDurableAndExclusive(t *testing.T) {
	ctx := context.Background()
	dir := privateDirectory(t)
	s, e := capture.OpenFileStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	inputs := [][]byte{[]byte("원본\n"), {}, []byte("é")}
	if e = s.Create(ctx, fileID, inputs); e != nil {
		t.Fatal(e)
	}
	if e = s.Create(ctx, fileID, [][]byte{[]byte("replacement")}); e == nil {
		t.Fatal("overwrite")
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Load(ctx, fileID); e == nil {
		t.Fatal("closed store")
	}
	s, e = capture.OpenFileStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	got, e := s.Load(ctx, fileID)
	if e != nil || commitment(got) != commitment(inputs) {
		t.Fatalf("restart: %v", e)
	}
	h, _ := capture.NewHost(s)
	r, e := h.Capture(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	if b, e := r.Inputs(ctx); e != nil || len(b) != 0 {
		t.Fatal("empty list")
	}
}
func TestFileStoreRejectsInvalidCustody(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []string{"permissions", "symlink", "hardlink", "directory", "oversize", "unknown", "duplicate", "invalidutf8", "wrongid", "noncanonical", "missing"} {
		t.Run(mode, func(t *testing.T) {
			dir := privateDirectory(t)
			s, e := capture.OpenFileStore(dir)
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			if e = s.Create(ctx, fileID, [][]byte{[]byte("inert")}); e != nil {
				t.Fatal(e)
			}
			path := filepath.Join(dir, fileID+".json")
			switch mode {
			case "permissions":
				e = os.Chmod(path, 0644)
			case "symlink":
				target := filepath.Join(dir, "fixture")
				e = os.Rename(path, target)
				if e == nil {
					e = os.Symlink(target, path)
				}
			case "hardlink":
				e = os.Link(path, filepath.Join(dir, "fixture"))
			case "directory":
				e = os.Remove(path)
				if e == nil {
					e = os.Mkdir(path, 0700)
				}
			case "oversize":
				e = os.WriteFile(path, make([]byte, (2<<20)+1), 0600)
			case "unknown":
				e = os.WriteFile(path, []byte(`{"version":"0.10.0","request_id":"`+fileID+`","inputs":[],"other":true}`), 0600)
			case "duplicate":
				e = os.WriteFile(path, []byte(`{"version":"0.10.0","version":"0.10.0","request_id":"`+fileID+`","inputs":[]}`), 0600)
			case "invalidutf8":
				e = os.WriteFile(path, []byte(`{"version":"0.10.0","request_id":"`+fileID+`","inputs":["/w=="]}`), 0600)
			case "wrongid":
				e = os.WriteFile(path, []byte(`{"version":"0.10.0","request_id":"00000000-0000-4000-8000-000000000002","inputs":[]}`), 0600)
			case "noncanonical":
				e = os.WriteFile(path, []byte(" {}\n"), 0600)
			case "missing":
				e = os.Remove(path)
			}
			if e != nil {
				t.Fatal(e)
			}
			if _, e = s.Load(ctx, fileID); e == nil {
				t.Fatal("invalid record accepted")
			}
		})
	}
}
func TestFileStoreBoundaryErrors(t *testing.T) {
	dir := privateDirectory(t)
	if e := os.Mkdir(dir, 0755); e != nil {
		t.Fatal(e)
	}
	if _, e := capture.OpenFileStore(dir); e == nil {
		t.Fatal("public directory")
	}
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if e := os.Symlink(dir, alias); e != nil {
		t.Fatal(e)
	}
	if _, e := capture.OpenFileStore(alias); e == nil {
		t.Fatal("directory symlink")
	}
	s, e := capture.OpenFileStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	ctx := context.Background()
	for _, id := range []string{"../other", "00000000-0000-0000-0000-000000000000", "00000000-0000-4000-C000-000000000001"} {
		if e = s.Create(ctx, id, nil); e == nil {
			t.Fatal("invalid ID")
		}
		if _, e = s.Load(ctx, id); e == nil {
			t.Fatal("invalid load ID")
		}
	}
	if e = s.Create(ctx, fileID, [][]byte{{255}}); e == nil {
		t.Fatal("invalid UTF8")
	}
	if _, e = s.Load(nil, fileID); e == nil {
		t.Fatal("nil context")
	}
	if e = os.Chmod(dir, 0755); e != nil {
		t.Fatal(e)
	}
	if e = s.Create(ctx, fileID, nil); e == nil {
		t.Fatal("changed directory permissions")
	}
}

func TestFileStoreRejectsIncompleteRecordAndOversizedInput(t *testing.T) {
	ctx := context.Background()
	dir := privateDirectory(t)
	s, e := capture.OpenFileStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = os.WriteFile(filepath.Join(dir, fileID+".json"), []byte("{"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Load(ctx, fileID); e == nil {
		t.Fatal("incomplete read")
	}
	if e = s.Create(ctx, fileID, [][]byte{[]byte("retry")}); e == nil {
		t.Fatal("incomplete record replaced")
	}
	if e = s.Create(ctx, "00000000-0000-4000-8000-000000000002", [][]byte{make([]byte, (1<<20)+1)}); e == nil {
		t.Fatal("oversized input")
	}
	if e = s.Create(nil, fileID, nil); e == nil {
		t.Fatal("nil create context")
	}
	var absent *capture.FileStore
	if e = absent.Close(); e == nil {
		t.Fatal("nil close")
	}
	if _, e = absent.Load(ctx, fileID); e == nil {
		t.Fatal("nil load")
	}
	if e = absent.Create(ctx, fileID, nil); e == nil {
		t.Fatal("nil create")
	}
	s.Close()
	if e = s.Create(ctx, "00000000-0000-4000-8000-000000000003", nil); e == nil {
		t.Fatal("closed create")
	}
	regular := filepath.Join(t.TempDir(), "file")
	if e = os.WriteFile(regular, nil, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = capture.OpenFileStore(regular); e == nil {
		t.Fatal("file directory")
	}
	if _, e = capture.OpenFileStore(filepath.Join(t.TempDir(), "missing", "child")); e == nil {
		t.Fatal("missing ancestor")
	}
}

func TestConcurrentDurableCaptures(t *testing.T) {
	s, e := capture.OpenFileStore(privateDirectory(t))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	h, _ := capture.NewHost(s)
	results := make(chan *capture.Request, 16)
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		go func() {
			r, e := h.Capture(context.Background(), [][]byte{[]byte("inert concurrent input")})
			results <- r
			errs <- e
		}()
	}
	seen := map[string]bool{}
	for i := 0; i < 16; i++ {
		r := <-results
		e := <-errs
		if e != nil {
			t.Fatal(e)
		}
		if r == nil || seen[r.ID()] {
			t.Fatal("capture IDs collided")
		}
		seen[r.ID()] = true
		if _, e = r.Inputs(context.Background()); e != nil {
			t.Fatal(e)
		}
	}
}
