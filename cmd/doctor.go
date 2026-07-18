package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Blynkosaur/treehouse/internal/check"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

var (
	lsMode    bool
	doctorCmd = &cobra.Command{
		Use:   "doctor",
		Short: "Report env drift for every service in this worktree",
		RunE:  runDoctor,
	}
)

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVar(&lsMode, "ls", false, "compact table output")
}

// runDoctor is pure adapter: gather input, call internal/check, pick a face.
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

	// Reference for expected keys: a service's own .env.example, else the main
	// checkout's .env. Outside a git repo there's no fallback — doctor still
	// works from .env.example alone.
	var source check.Worktree
	if mainRoot, err := check.MainWorktree(root); err == nil {
		source, _ = check.Discover(mainRoot)
	}
	findings := check.Doctor{}.CheckEnv(wt, source)

	if lsMode {
		printTable(findings, root)
	} else {
		printReport(findings, root)
	}
	// Exit 0 by design: inferred requirements are WARN-level; FAIL is
	// reserved for human-curated config, which doesn't exist yet.
	return nil
}

// relDir prettifies a finding's directory relative to the scanned root.
func relDir(root, dir string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." {
		return dir
	}
	return rel
}

// printReport is the narrative face: multi-line, explanatory, human-paced.
func printReport(findings []check.Finding, root string) {
	problems := 0
	for _, f := range findings {
		rel := relDir(root, f.Dir)
		switch {
		case f.NoEnv:
			problems++
			fmt.Printf("✗ %s: .env missing entirely (%d keys expected)\n", rel, f.Keys)
		case f.Drifted():
			problems++
			if len(f.Missing) > 0 {
				fmt.Printf("! %s: %d expected keys missing from .env:\n", rel, len(f.Missing))
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
		fmt.Printf("\n%d service(s) with env drift (inferred requirements → warnings only)\n", problems)
	}
}

var (
	missingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // red
	emptyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // yellow
	okStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	headerStyle  = lipgloss.NewStyle().Bold(true)
)

// printTable is the glance face: one bordered row per service, keys listed
// explicitly as multi-line cells.
func printTable(findings []check.Finding, root string) {
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderRow(true).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle.Padding(0, 1)
			}
			return lipgloss.NewStyle().Padding(0, 1)
		}).
		Headers("SERVICE", "KEYS", "MISSING", "EMPTY")

	for _, f := range findings {
		service := relDir(root, f.Dir)
		if f.NoEnv {
			t.Row(service, fmt.Sprintf("%d", f.Keys),
				missingStyle.Render("(.env missing entirely)"), "")
			continue
		}
		missing := okStyle.Render("—")
		if len(f.Missing) > 0 {
			missing = missingStyle.Render(strings.Join(f.Missing, "\n"))
		}
		empty := okStyle.Render("—")
		if len(f.Empty) > 0 {
			empty = emptyStyle.Render(strings.Join(f.Empty, "\n"))
		}
		t.Row(service, fmt.Sprintf("%d", f.Keys), missing, empty)
	}

	fmt.Println(t)
}
