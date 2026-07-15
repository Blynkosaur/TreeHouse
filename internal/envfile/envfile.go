package envfile

import (
	"bufio"
	"log"
	"os"
	"strings"
)

type File struct {
	Path         string
	envVariables map[string]string
}

func LoadPath(path string) (File, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	content := string(bytes)
	varPairs, parseError := Parse(content)
	if parseError != nil {
		log.Fatal(err)
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
