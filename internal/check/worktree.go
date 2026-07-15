package check

import "github.com/Blynkosaur/treehouse/internal/envfile"

type Worktree struct {
	Root     string
	EnvFiles []envfile.File
}
