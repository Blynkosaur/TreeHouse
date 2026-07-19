package deps_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Blynkosaur/treehouse/internal/deps"
)

func TestCloneTreeCorrectnessAndIsolation(t *testing.T) {
	if _, err := exec.LookPath("cp"); err != nil {
		t.Skip("cp not on PATH")
	}

	src := filepath.Join(t.TempDir(), "src")
	nested := filepath.Join(src, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	srcFile := filepath.Join(nested, "f.txt")
	if err := os.WriteFile(srcFile, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	// dst's parent does not yet exist — CloneTree must MkdirAll it.
	dst := filepath.Join(t.TempDir(), "newparent", "dst")
	if err := deps.CloneTree(src, dst); err != nil {
		t.Fatalf("CloneTree: %v", err)
	}

	dstFile := filepath.Join(dst, "a", "b", "f.txt")
	got, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("read cloned file: %v", err)
	}
	if string(got) != "original" {
		t.Errorf("cloned contents = %q, want %q", got, "original")
	}

	// Isolation: writing into the clone must not touch the source.
	if err := os.WriteFile(dstFile, []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}
	srcAfter, err := os.ReadFile(srcFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(srcAfter) != "original" {
		t.Errorf("source changed after clone write = %q, want unchanged %q", srcAfter, "original")
	}
}
