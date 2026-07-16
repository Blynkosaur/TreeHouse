package check

import (
	"reflect"
	"testing"

	"github.com/Blynkosaur/treehouse/internal/envfile"
)

// CheckEnv operates on in-memory Worktrees — no disk, no binary, no stdout.
// This is the payoff of returning data instead of printing it.
func TestCheckEnv(t *testing.T) {
	wt := Worktree{
		Root: "/repo",
		EnvFiles: []envfile.File{
			// service a: one key missing, one empty, one fine
			{Path: "/repo/a/.env", Vars: map[string]string{"OK": "1", "EMPTY": ""}},
			{Path: "/repo/a/.env.example", Vars: map[string]string{"OK": "", "EMPTY": "", "GONE": ""}},
			// service b: fully healthy
			{Path: "/repo/b/.env", Vars: map[string]string{"X": "1"}},
			{Path: "/repo/b/.env.example", Vars: map[string]string{"X": ""}},
			// service c: example exists, .env missing entirely
			{Path: "/repo/c/.env.example", Vars: map[string]string{"A": "", "B": ""}},
			// service d: .env with no example — must produce NO finding
			{Path: "/repo/d/.env", Vars: map[string]string{"LONER": "1"}},
		},
	}

	findings := Doctor{}.CheckEnv(wt)

	if len(findings) != 3 {
		t.Fatalf("got %d findings, want 3 (a, b, c — d has no example): %+v", len(findings), findings)
	}

	a, b, c := findings[0], findings[1], findings[2] // sorted by dir

	if !reflect.DeepEqual(a.Missing, []string{"GONE"}) || !reflect.DeepEqual(a.Empty, []string{"EMPTY"}) {
		t.Errorf("service a: Missing=%v Empty=%v, want Missing=[GONE] Empty=[EMPTY]", a.Missing, a.Empty)
	}
	if !a.Drifted() {
		t.Error("service a should be drifted")
	}

	if b.Drifted() {
		t.Errorf("service b should be healthy, got %+v", b)
	}

	if !c.NoEnv || c.Keys != 2 {
		t.Errorf("service c: NoEnv=%v Keys=%d, want NoEnv=true Keys=2", c.NoEnv, c.Keys)
	}
}
