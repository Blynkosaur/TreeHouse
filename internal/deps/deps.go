// Package deps applies the provisioning plans check.PlanDeps decides — the
// mutating "doer" half (mirrors internal/envfile). It clones dependency dirs
// copy-on-write and recreates the ones that bake absolute paths.
package deps

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// CloneTree copies src to dst, preferring an APFS copy-on-write clone so a
// node_modules costs near-zero disk and time yet stays isolated. On darwin it
// tries `cp -c -R` (clonefile) and falls back to a plain `cp -R` when the
// volume can't clone (non-APFS / cross-volume); elsewhere it's just `cp -R`.
func CloneTree(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if runtime.GOOS == "darwin" {
		if err := run("cp", "-c", "-R", src, dst); err == nil {
			return nil
		}
		// clonefile unsupported here — fall back to a real copy (still isolated).
	}
	return run("cp", "-R", src, dst)
}

// RunRecreate rebuilds a dependency dir in place by running command in dir, so
// the user watches install progress live.
func RunRecreate(dir, command string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
