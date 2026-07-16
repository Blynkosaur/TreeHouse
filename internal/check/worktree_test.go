package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscover(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// A miniature birdseye: two services, decoy variants, a node_modules trap.
	write("seo_service/.env", "A=1")
	write("seo_service/.env.example", "A=\nB=")
	write("seo_service/.env.backup.20260622", "OLD=1") // exact-name rule: NOT collected
	write("seo_service/.env.dev", "DEV=1")             // exact-name rule: NOT collected
	write("seo_webapp/.env", "C=3")
	write("node_modules/pkg/.env", "EVIL=1") // pruned subtree: NOT collected
	write("README.md", "not an env file")

	wt, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(wt.EnvFiles) != 3 {
		var found []string
		for _, f := range wt.EnvFiles {
			found = append(found, f.Path)
		}
		t.Errorf("found %d env files, want 3:\n%s", len(wt.EnvFiles), strings.Join(found, "\n"))
	}
	for _, f := range wt.EnvFiles {
		if strings.Contains(f.Path, "node_modules") {
			t.Errorf("walker failed to skip node_modules: %s", f.Path)
		}
	}

	// Spot-check that files were actually parsed, not just located.
	for _, f := range wt.EnvFiles {
		if filepath.Base(f.Path) == ".env" && strings.Contains(f.Path, "seo_service") {
			if f.Vars["A"] != "1" {
				t.Errorf("seo_service/.env not parsed: Vars=%v", f.Vars)
			}
		}
	}
}

func TestDiscoverMissingRoot(t *testing.T) {
	_, err := Discover(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Error("expected error for nonexistent root, got nil")
	}
}
