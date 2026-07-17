package check

import (
	"reflect"
	"testing"

	"github.com/Blynkosaur/treehouse/internal/envfile"
)

func TestPlanHydrate(t *testing.T) {
	// The broken worktree: three services in trouble, one healthy.
	wt := Worktree{
		Root: "/wt",
		EnvFiles: []envfile.File{
			// svc_a: missing C and D
			{Path: "/wt/svc_a/.env", Vars: map[string]string{"A": "1", "B": "2"}},
			{Path: "/wt/svc_a/.env.example", Vars: map[string]string{"A": "", "B": "", "C": "", "D": ""}},
			// svc_b: healthy — must produce no repair
			{Path: "/wt/svc_b/.env", Vars: map[string]string{"X": "1"}},
			{Path: "/wt/svc_b/.env.example", Vars: map[string]string{"X": ""}},
			// svc_c: .env missing entirely
			{Path: "/wt/svc_c/.env.example", Vars: map[string]string{"P": "", "Q": ""}},
		},
	}
	// The source (main checkout): has C for svc_a but not D; has P for svc_c.
	source := Worktree{
		Root: "/main",
		EnvFiles: []envfile.File{
			{Path: "/main/svc_a/.env", Vars: map[string]string{"A": "1", "B": "2", "C": "3"}},
			{Path: "/main/svc_c/.env", Vars: map[string]string{"P": "9"}},
		},
	}

	repairs := Doctor{}.PlanHydrate(wt, source)

	if len(repairs) != 2 {
		t.Fatalf("got %d repairs, want 2 (svc_a, svc_c): %+v", len(repairs), repairs)
	}

	a, c := repairs[0], repairs[1] // CheckEnv findings are dir-sorted

	if a.Create {
		t.Error("svc_a has a .env; repair must not be Create")
	}
	if a.EnvPath != "/wt/svc_a/.env" {
		t.Errorf("svc_a EnvPath = %q", a.EnvPath)
	}
	wantAdd := map[string]string{"C": "3", "D": ""}
	if !reflect.DeepEqual(a.Add, wantAdd) {
		t.Errorf("svc_a Add = %v, want %v", a.Add, wantAdd)
	}
	if !reflect.DeepEqual(a.Unsourced, []string{"D"}) {
		t.Errorf("svc_a Unsourced = %v, want [D]", a.Unsourced)
	}

	if !c.Create {
		t.Error("svc_c has no .env; repair must be Create")
	}
	wantAdd = map[string]string{"P": "9", "Q": ""}
	if !reflect.DeepEqual(c.Add, wantAdd) {
		t.Errorf("svc_c Add = %v, want %v", c.Add, wantAdd)
	}
	if !reflect.DeepEqual(c.Unsourced, []string{"Q"}) {
		t.Errorf("svc_c Unsourced = %v, want [Q]", c.Unsourced)
	}
}

func TestPlanHydrateNoSourceService(t *testing.T) {
	// Source worktree lacks the service dir entirely → everything unsourced.
	wt := Worktree{
		Root: "/wt",
		EnvFiles: []envfile.File{
			{Path: "/wt/svc_new/.env.example", Vars: map[string]string{"K": ""}},
		},
	}
	source := Worktree{Root: "/main"} // empty

	repairs := Doctor{}.PlanHydrate(wt, source)
	if len(repairs) != 1 {
		t.Fatalf("got %d repairs, want 1", len(repairs))
	}
	r := repairs[0]
	if !r.Create || r.Add["K"] != "" || len(r.Unsourced) != 1 {
		t.Errorf("unexpected repair: %+v", r)
	}
}
