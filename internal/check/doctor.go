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
	NoEnv   bool     // .env.example exists but .env is missing entirely
	Keys    int      // how many keys .env.example expects (context for output)
}

// Drifted reports whether this finding represents a problem worth flagging.
func (f Finding) Drifted() bool {
	return f.NoEnv || len(f.Missing) > 0 || len(f.Empty) > 0
}

// CheckEnv compares each .env against its sibling .env.example and returns
// one Finding per directory that has an example to compare against.
// Directories with a .env but no example are skipped — nothing to infer.
func (d Doctor) CheckEnv(w Worktree) []Finding {
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

	// Map iteration order is random in Go; sort dirs so Findings are stable.
	dirs := make([]string, 0, len(pairs))
	for dir := range pairs {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	var findings []Finding
	for _, dir := range dirs {
		p := pairs[dir]
		if p.example == nil {
			continue // .env with no example: nothing to compare against
		}
		f := Finding{Dir: dir, Keys: len(p.example.Vars)}
		if p.env == nil {
			f.NoEnv = true
			findings = append(findings, f)
			continue
		}
		for key := range p.example.Vars {
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
