package envfile

import (
	"bufio"
	"strings"
)

type WorktreeEnvParser struct {
	worktreeDirectory string
	envVariables      map[string]string
}

func Parse(envContent string) (map[string]string, error) {
	reader := strings.NewReader(envContent)
	scanner := bufio.NewScanner(reader)
	envmap := make(map[string]string)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, val, found := strings.Cut(line, "=")
		if found {
			key = strings.TrimSpace(key)
			val = strings.TrimSpace(val)
			val = strings.Trim(val, `"'`)
			envmap[key] = val
		}
	}
	return envmap, scanner.Err()
}
