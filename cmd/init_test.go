package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Blynkosaur/treehouse/internal/config"
)

// The scaffold init writes must be valid TOML that config.Load accepts. All
// rules are commented out, so a fresh file parses to zero deps (zero-config
// behavior preserved) — this guards against a typo turning init's output into
// something hydrate can't read.
func TestInitTemplateParses(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "treehouse.toml"), []byte(treehouseTOML), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := config.Load(dir)
	if err != nil {
		t.Fatalf("init template does not parse as valid config: %v", err)
	}
	if len(f.Deps) != 0 {
		t.Errorf("template should have no active rules, got %d: %+v", len(f.Deps), f.Deps)
	}
}
