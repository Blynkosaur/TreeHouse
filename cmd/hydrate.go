package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Blynkosaur/treehouse/internal/check"
	"github.com/Blynkosaur/treehouse/internal/config"
	"github.com/Blynkosaur/treehouse/internal/deps"
	"github.com/Blynkosaur/treehouse/internal/envfile"
	"github.com/spf13/cobra"
)

var (
	hydrateDry      bool
	hydrateSkipDeps bool
	hydrateCmd      = &cobra.Command{
		Use:   "hydrate",
		Short: "Fill this worktree's .env files from the main checkout",
		RunE:  runHydrate,
	}
)

func init() {
	rootCmd.AddCommand(hydrateCmd)
	hydrateCmd.Flags().BoolVar(&hydrateDry, "dry", false, "show what would change without writing")
	hydrateCmd.Flags().BoolVar(&hydrateSkipDeps, "skip-deps", false, "only fill .env; skip dependency provisioning")
}

// runHydrate is a pure adapter (same shape as runDoctor): discover this
// worktree and the main checkout, let check.PlanHydrate decide the fixes, then
// apply them with the byte-preserving envfile writers. All judgment lives in
// check — this file only gathers input, writes files, and talks to the terminal.
func runHydrate(cmd *cobra.Command, args []string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	wt, err := check.Discover(root)
	if err != nil {
		return err
	}

	// Canonical values come from the main worktree's .env files.
	sourceRoot, err := check.MainWorktree(root)
	if err != nil {
		return fmt.Errorf("locating main worktree: %w", err)
	}
	source, err := check.Discover(sourceRoot)
	if err != nil {
		return err
	}

	repairs := check.Doctor{}.PlanHydrate(wt, source)
	// A clean env phase is a note, not an exit: the deps phase below still runs.
	if len(repairs) == 0 {
		fmt.Println("nothing to hydrate — every .env already has its keys")
	}

	for _, r := range repairs {
		rel := relDir(root, filepath.Dir(r.EnvPath))
		n := nKeys(len(r.Add))

		switch {
		case hydrateDry && r.Create:
			fmt.Printf("~ %s: would create .env (%s)\n", rel, n)
		case hydrateDry:
			fmt.Printf("~ %s: would add %s\n", rel, n)
		default:
			// Create and Append share a signature, so pick one and call it.
			apply := envfile.Append
			if r.Create {
				apply = envfile.Create
			}
			if err := apply(r.EnvPath, r.Add); err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			if r.Create {
				fmt.Printf("✓ %s: created .env (%s)\n", rel, n)
			} else {
				fmt.Printf("✓ %s: added %s\n", rel, n)
			}
		}

		// Keys the main checkout couldn't supply were written empty — the human
		// still has to fill them, so name them explicitly.
		if len(r.Unsourced) > 0 {
			fmt.Printf("    fill manually (no value in main): %s\n", strings.Join(r.Unsourced, ", "))
		}
	}

	return hydrateDeps(root, wt, source, sourceRoot)
}

// hydrateDeps provisions heavy dependency dirs (node_modules, .venv, …) into
// this worktree. Rules come from the built-in defaults overlaid with any
// treehouse.toml in main — the source of truth. All judgment lives in
// check.PlanDeps; this loop only applies plans and talks to the terminal.
func hydrateDeps(root string, wt, source check.Worktree, sourceRoot string) error {
	if hydrateSkipDeps {
		return nil
	}

	cfg, _ := config.Load(sourceRoot) // absent/broken config: fall back to defaults
	rules := config.MergeRules(check.DefaultDepRules(), cfg.Deps)

	plans := check.Doctor{}.PlanDeps(wt, source, rules)
	if len(plans) == 0 {
		fmt.Println("deps: nothing to provision")
		return nil
	}

	for _, p := range plans {
		rel := relDir(root, p.Dst)

		switch {
		case p.Skip != "":
			fmt.Printf("• %s: skipped (%s)\n", rel, p.Skip)
		case hydrateDry && p.Action == check.Clone:
			fmt.Printf("~ %s: would clone\n", rel)
		case hydrateDry:
			fmt.Printf("~ %s: would recreate (%s)\n", rel, p.Command)
		case p.Action == check.Clone:
			if err := deps.CloneTree(p.Src, p.Dst); err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			fmt.Printf("✓ %s: cloned\n", rel)
		default:
			fmt.Printf("~ %s: recreating (%s)\n", rel, p.Command)
			if err := deps.RunRecreate(filepath.Dir(p.Dst), p.Command); err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			fmt.Printf("✓ %s: recreated\n", rel)
		}
	}
	return nil
}

// nKeys pluralizes a key count for human output ("1 key", "3 keys").
func nKeys(n int) string {
	if n == 1 {
		return "1 key"
	}
	return fmt.Sprintf("%d keys", n)
}
