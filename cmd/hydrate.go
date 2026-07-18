package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Blynkosaur/treehouse/internal/check"
	"github.com/Blynkosaur/treehouse/internal/envfile"
	"github.com/spf13/cobra"
)

var (
	hydrateDry bool
	hydrateCmd = &cobra.Command{
		Use:   "hydrate",
		Short: "Fill this worktree's .env files from the main checkout",
		RunE:  runHydrate,
	}
)

func init() {
	rootCmd.AddCommand(hydrateCmd)
	hydrateCmd.Flags().BoolVar(&hydrateDry, "dry", false, "show what would change without writing")
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
	if len(repairs) == 0 {
		fmt.Println("nothing to hydrate — every .env already has its keys")
		return nil
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
	return nil
}

// nKeys pluralizes a key count for human output ("1 key", "3 keys").
func nKeys(n int) string {
	if n == 1 {
		return "1 key"
	}
	return fmt.Sprintf("%d keys", n)
}
