package envfile

import (
	"bufio"
	"os"
	"sort"
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

// Marker introduces every block of keys treehouse adds to an env file.
const Marker = "# --- added by treehouse hydrate ---"

// Append adds vars to the END of the file at path, under a marker comment,
// preserving every existing byte. The file is created if it doesn't exist.
// Keys are written sorted for deterministic output.
func Append(path string, vars map[string]string) error {
	if len(vars) == 0 {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("\n" + Marker + "\n")
	for _, k := range keys {
		b.WriteString(k + "=" + vars[k] + "\n")
	}
	_, err = f.WriteString(b.String())
	return err
}

// Create writes a brand-new env file from vars. It refuses to overwrite:
// an existing file at path is an error, never a casualty.
func Create(path string, vars map[string]string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if _, err := f.WriteString(k + "=" + vars[k] + "\n"); err != nil {
			return err
		}
	}
	return nil
}
