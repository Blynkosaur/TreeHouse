// Package check examines a worktree's environment state and reports findings.
package check

import (
	"errors"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Blynkosaur/treehouse/internal/envfile"
)

// Worktree is one checked-out working directory and every env file found in it.
type Worktree struct {
	Root     string
	EnvFiles []envfile.File
}

// skipDirs are directory names never worth descending into: huge, generated,
// or private to tooling. Skipping them keeps Discover fast on real monorepos.
var skipDirs = map[string]bool{
	".git":          true,
	"node_modules":  true,
	".venv":         true,
	"venv":          true,
	"__pycache__":   true,
	"vendor":        true,
	"dist":          true,
	"build":         true,
	".mypy_cache":   true,
	".ruff_cache":   true,
	".pytest_cache": true,
}

// envFileNames are the exact filenames Discover collects. Deliberately NOT a
// prefix match: .env.dev, .env.local, .env.backup.* would turn the doctor
// into a noise machine. Variants become a config option later, on purpose.
var envFileNames = map[string]bool{
	".env":         true,
	".env.example": true,
}

// Discover walks root and returns a Worktree holding every parsed env file.
// It is deliberately lenient: unreadable directories and unparsable files are
// skipped, never fatal — a doctor that faints mid-examination helps nobody.
func Discover(root string) (Worktree, error) {
	wt := Worktree{Root: root}

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				// The root itself is unreadable or missing — that's not
				// leniency territory, that's a broken invocation.
				return err
			}
			// Deeper entries (permissions, vanished mid-walk): skip, keep walking.
			return nil
		}
		if d.IsDir() {
			if path != root && skipDirs[d.Name()] {
				// fs.SkipDir returned from a directory prunes its ENTIRE
				// subtree — this one line is the node_modules immunity.
				return fs.SkipDir
			}
			return nil
		}
		if !envFileNames[d.Name()] {
			return nil
		}
		f, err := envfile.LoadPath(path)
		if err != nil {
			return nil // lenient: a broken file is not a broken walk
		}
		wt.EnvFiles = append(wt.EnvFiles, f)
		return nil
	})
	if walkErr != nil {
		// Only reachable for catastrophic cases (e.g. root doesn't exist),
		// since our callback never returns errors of its own.
		return Worktree{}, walkErr
	}
	return wt, nil
}

// MainWorktree returns the path of the repository's primary worktree — the
// first entry `git worktree list` reports (git guarantees the main checkout
// comes first). hydrate sources canonical env values from it. cwd selects the
// repository to ask; it need not be the main worktree itself.
func MainWorktree(cwd string) (string, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = cwd
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	// Porcelain output is blank-line-separated blocks, each opening with a
	// "worktree <path>" line. The first such line is the main worktree.
	for _, line := range strings.Split(string(output), "\n") {
		if path, ok := strings.CutPrefix(line, "worktree "); ok {
			return path, nil
		}
	}
	return "", errors.New("no worktrees found")
}
