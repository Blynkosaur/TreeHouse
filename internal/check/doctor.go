package check

import (
	"path/filepath"
	"sort"

	"github.com/Blynkosaur/treehouse/internal/envfile"
)

// Doctor examines worktrees and reports findings. Empty for now; it will grow
// fields when curated config arrives (required keys, services, data checks).
type Doctor struct{}

// Finding is one directory's env-drift verdict — pure data, no presentation.
// The same Findings feed the text printer today, --json tomorrow, the TUI later.
type Finding struct {
	Dir     string   // absolute directory the .env/.env.example pair lives in
	Missing []string // keys in .env.example but absent from .env (sorted)
	Empty   []string // keys present in .env but with empty values (sorted)
	NoEnv   bool     // a reference exists (example or main) but .env is missing entirely
	Keys    int      // how many keys .env.example expects (context for output)
}

// Drifted reports whether this finding represents a problem worth flagging.
func (f Finding) Drifted() bool {
	return f.NoEnv || len(f.Missing) > 0 || len(f.Empty) > 0
}

// CheckEnv reports env drift for each service. The reference for "which keys
// should exist" is a service's sibling .env.example when it has one; for repos
// that ship no .env.example, it falls back to the main checkout's .env for the
// same service (pass source = the main worktree; pass a zero Worktree to skip
// the fallback). A service with neither reference is skipped — nothing to infer.
func (d Doctor) CheckEnv(w, source Worktree) []Finding {
	// Group the flat file list into per-directory pairs: a .env and its
	// sibling .env.example belong together (= one "service" in a monorepo).
	type pair struct {
		env     *envfile.File
		example *envfile.File
	}
	pairs := map[string]*pair{}
	for i := range w.EnvFiles {
		// Index (&w.EnvFiles[i]) rather than ranging over values: we want
		// the address of the real element, not of a loop-variable copy.
		f := &w.EnvFiles[i]
		dir := filepath.Dir(f.Path)
		if pairs[dir] == nil {
			pairs[dir] = &pair{}
		}
		if filepath.Base(f.Path) == ".env" {
			pairs[dir].env = f
		} else {
			pairs[dir].example = f
		}
	}

	// Fallback reference: the main checkout's real .env vars, keyed by dir
	// relative to its root so they map onto this worktree's dirs.
	srcVars := map[string]map[string]string{}
	for i := range source.EnvFiles {
		f := &source.EnvFiles[i]
		if filepath.Base(f.Path) != ".env" {
			continue
		}
		if rel, err := filepath.Rel(source.Root, filepath.Dir(f.Path)); err == nil {
			srcVars[rel] = f.Vars
		}
	}

	// Candidate dirs: every dir with an env file here, plus every dir the main
	// checkout has a .env for — so a service missing entirely in this worktree
	// (and lacking a .env.example) is still caught. Sort for stable output.
	dirsSet := map[string]bool{}
	for dir := range pairs {
		dirsSet[dir] = true
	}
	for rel := range srcVars {
		dirsSet[filepath.Join(w.Root, rel)] = true
	}
	dirs := make([]string, 0, len(dirsSet))
	for dir := range dirsSet {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	var findings []Finding
	for _, dir := range dirs {
		p := pairs[dir] // nil when the dir came only from the main checkout

		// Resolve the reference key set: sibling .env.example wins; else the
		// main checkout's .env for this dir; else nothing to infer.
		var ref map[string]string
		if p != nil && p.example != nil {
			ref = p.example.Vars
		} else {
			rel, err := filepath.Rel(w.Root, dir)
			if err != nil {
				rel = dir
			}
			ref = srcVars[rel] // nil when main has no .env here either
		}
		if ref == nil {
			continue // .env with neither an example nor a main reference
		}

		f := Finding{Dir: dir, Keys: len(ref)}
		if p == nil || p.env == nil {
			// A nonexistent .env is missing every reference key — recording
			// them makes Finding lossless for hydrate planning.
			f.NoEnv = true
			for key := range ref {
				f.Missing = append(f.Missing, key)
			}
			sort.Strings(f.Missing)
			findings = append(findings, f)
			continue
		}
		for key := range ref {
			val, ok := p.env.Vars[key]
			switch {
			case !ok:
				f.Missing = append(f.Missing, key)
			case val == "":
				f.Empty = append(f.Empty, key)
			}
		}
		sort.Strings(f.Missing)
		sort.Strings(f.Empty)
		findings = append(findings, f)
	}
	return findings
}
