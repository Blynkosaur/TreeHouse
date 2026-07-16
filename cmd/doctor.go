package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Blynkosaur/treehouse/internal/check"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Report env drift for every service in this worktree",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

// runDoctor is pure adapter: gather input, call internal/check, present output.
// All judgment lives in check.Doctor — this file just talks to the terminal.
func runDoctor(cmd *cobra.Command, args []string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	wt, err := check.Discover(root)
	if err != nil {
		return err
	}
	findings := check.Doctor{}.CheckEnv(wt)

	problems := 0
	for _, f := range findings {
		rel, err := filepath.Rel(root, f.Dir)
		if err != nil || rel == "." {
			rel = f.Dir
		}
		switch {
		case f.NoEnv:
			problems++
			fmt.Printf("✗ %s: .env missing entirely (%d keys expected)\n", rel, f.Keys)
		case f.Drifted():
			problems++
			if len(f.Missing) > 0 {
				fmt.Printf("! %s: %d keys in .env.example missing from .env:\n", rel, len(f.Missing))
				for _, k := range f.Missing {
					fmt.Printf("    %s\n", k)
				}
			}
			if len(f.Empty) > 0 {
				fmt.Printf("! %s: %d keys present but empty:\n", rel, len(f.Empty))
				for _, k := range f.Empty {
					fmt.Printf("    %s\n", k)
				}
			}
		default:
			fmt.Printf("✓ %s: .env has all %d expected keys\n", rel, f.Keys)
		}
	}

	if problems == 0 {
		fmt.Println("\nall clear")
	} else {
		fmt.Printf("\n%d service(s) with env drift (inferred from .env.example → warnings only)\n", problems)
	}
	// Exit 0 by design: inferred requirements are WARN-level; FAIL is
	// reserved for human-curated config, which doesn't exist yet.
	return nil
}
