package envfile

import (
	"bufio"
	"os"
	"strings"
)

type File struct {
	Path string
	Vars map[string]string
}

func LoadPath(path string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}
	content := string(data)
	varPairs, parseError := Parse(content)
	if parseError != nil {
		return File{}, err
	}
	return File{path, varPairs}, err
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
