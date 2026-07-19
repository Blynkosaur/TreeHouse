package check

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DepAction is how a heavy dependency dir is provisioned into a new worktree.
type DepAction int

const (
	Clone    DepAction = iota // copy-on-write copy: contents are path-independent (node_modules)
	Recreate                  // rebuild in place: contents bake absolute paths (.venv)
)

func (a DepAction) String() string {
	if a == Recreate {
		return "recreate"
	}
	return "clone"
}

// UnmarshalText lets treehouse.toml spell the action as "clone"/"recreate".
func (a *DepAction) UnmarshalText(text []byte) error {
	switch string(text) {
	case "clone":
		*a = Clone
	case "recreate":
		*a = Recreate
	default:
		return fmt.Errorf("unknown deps action %q (want clone|recreate)", text)
	}
	return nil
}

// DepRule declares how to provision one kind of dependency dir. Built-in
// defaults (DefaultDepRules) cover node_modules and .venv; treehouse.toml
// [[deps]] entries add or override rules — the agent extension point.
type DepRule struct {
	Name    string    `toml:"name"`    // dir name ("node_modules") or worktree-relative path ("vendor/bundle") to find in the source
	Action  DepAction `toml:"action"`  // Clone or Recreate
	Command string    `toml:"command"` // Recreate only: shell command that rebuilds the dir, run in its parent
}

// DefaultDepRules are the zero-config built-ins: clone node_modules, recreate a
// Python venv (its command is resolved per-dir from the manifest — see PlanDeps).
func DefaultDepRules() []DepRule {
	return []DepRule{
		{Name: "node_modules", Action: Clone},
		{Name: ".venv", Action: Recreate},
		{Name: "venv", Action: Recreate},
	}
}

// DepPlan is one resolved provisioning operation — computed, not yet applied
// (mirrors Repair). cmd/hydrate turns these into cp -c clones and recreate
// commands; this package only decides.
type DepPlan struct {
	Action   DepAction
	Src, Dst string // Clone: source dir -> worktree dir. Recreate: Dst is the dir, Command runs in its parent.
	Command  string // Recreate only
	Skip     string // non-empty => cannot act; human-readable reason (no manifest / uv missing)
}

// PlanDeps finds every dependency dir in the source worktree that matches a
// rule and plans its provisioning into w. Idempotent: a dir already present in
// w is skipped silently. For a Recreate rule with no explicit command (the
// built-in venv rules), the command is resolved from the manifest beside the
// dir; when nothing can be resolved the plan carries a Skip reason instead.
func (d Doctor) PlanDeps(w, source Worktree, rules []DepRule) []DepPlan {
	var plans []DepPlan
	for _, m := range findDepDirs(source.Root, rules) {
		dst := filepath.Join(w.Root, m.rel)
		if _, err := os.Stat(dst); err == nil {
			continue // already provisioned — nothing to do
		}

		p := DepPlan{Action: m.rule.Action, Dst: dst}
		switch m.rule.Action {
		case Clone:
			p.Src = filepath.Join(source.Root, m.rel)
		case Recreate:
			cmd := m.rule.Command
			if cmd == "" {
				cmd = resolveVenvCommand(filepath.Dir(dst)) // built-in venv: manifest ladder
			}
			switch {
			case cmd == "":
				p.Skip = "no manifest to rebuild from (add uv.lock/pyproject.toml/requirements.txt)"
			case firstWord(cmd) == "uv" && !onPath("uv"):
				p.Skip = "uv not installed (needed to rebuild " + m.rel + ")"
			}
			p.Command = cmd
		}
		plans = append(plans, p)
	}
	return plans
}

// matchedDir is a dependency dir found in a worktree paired with its rule.
type matchedDir struct {
	rel  string
	rule DepRule
}

// findDepDirs walks root and returns each directory matching a rule (by base
// name or worktree-relative path), without descending into it — a node_modules
// holds no nested node_modules we care about. Mirrors Discover's walk shape.
func findDepDirs(root string, rules []DepRule) []matchedDir {
	var found []matchedDir
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == root {
			return nil
		}
		if d.Name() == ".git" {
			return fs.SkipDir
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		for _, r := range rules {
			if r.Name == d.Name() || r.Name == rel {
				found = append(found, matchedDir{rel: rel, rule: r})
				return fs.SkipDir // don't descend into a matched dep dir
			}
		}
		return nil
	})
	return found
}

// resolveVenvCommand picks how to rebuild a Python venv from the manifest beside
// it. uv handles both lockfile and requirements styles. "" => cannot determine.
func resolveVenvCommand(parent string) string {
	has := func(name string) bool {
		_, err := os.Stat(filepath.Join(parent, name))
		return err == nil
	}
	switch {
	case has("uv.lock"), has("pyproject.toml"):
		return "uv venv && uv sync"
	case has("requirements.txt"):
		return "uv venv && uv pip install -r requirements.txt"
	default:
		return ""
	}
}

func firstWord(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, ' '); i >= 0 {
		return s[:i]
	}
	return s
}

func onPath(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}
