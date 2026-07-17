package envfile

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type envVariableItem struct {
	name  string
	input string
	want  map[string]string
}

var testItems = []envVariableItem{
	{
		name:  "simple pair",
		input: "FOO=bar\nSIX=seven",
		want:  map[string]string{"FOO": "bar", "SIX": "seven"},
	},
	{
		name:  "comments and blank lines skipped",
		input: "# database config\n\nFOO=bar\n",
		want:  map[string]string{"FOO": "bar"},
	},
	{
		name:  "empty value is present but empty",
		input: "REDIS_URL=",
		want:  map[string]string{"REDIS_URL": ""},
	},
	{
		name:  "quoted value has quotes stripped",
		input: `JWT_SECRET="abc123"`,
		want:  map[string]string{"JWT_SECRET": "abc123"},
	},
	{
		name:  "garbage line without equals is skipped",
		input: "this is not a pair\nFOO=bar",
		want:  map[string]string{"FOO": "bar"},
	},
}

func TestParse(t *testing.T) {
	for _, test := range testItems {
		t.Run(test.name, func(t *testing.T) {
			res, err := Parse(test.input)
			if err != nil {
				t.Fatal("Parse() returned idk something wrong")
			}
			if !reflect.DeepEqual(res, test.want) {
				t.Errorf("Parse(%q)\n  got:  %#v\n  want: %#v", test.input, res, test.want)
			}
		})
	}
}

func TestAppendPreservesOriginal(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	original := "# hand-written comment\nA=1\nB=custom # note\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Append(path, map[string]string{"D": "4", "C": "3"}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	// THE regression test for the WriteFile-destroys-everything bug:
	// every original byte must still open the file.
	if !strings.HasPrefix(got, original) {
		t.Fatalf("original content damaged!\ngot:\n%s", got)
	}
	if !strings.Contains(got, Marker) {
		t.Error("appended block missing marker comment")
	}
	// Sorted, complete, and re-parseable.
	if !strings.Contains(got, "C=3\nD=4\n") {
		t.Errorf("appended keys wrong or unsorted:\n%s", got)
	}
	vars, _ := Parse(got)
	if vars["A"] != "1" || vars["C"] != "3" || vars["D"] != "4" {
		t.Errorf("round-trip parse lost keys: %v", vars)
	}
}

func TestAppendCreatesWhenAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := Append(path, map[string]string{"A": "1"}); err != nil {
		t.Fatalf("Append to nonexistent file: %v", err)
	}
	vars, _ := Parse(readFile(t, path))
	if vars["A"] != "1" {
		t.Errorf("got %v", vars)
	}
}

func TestCreateRefusesOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("PRECIOUS=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Create(path, map[string]string{"A": "1"}); err == nil {
		t.Fatal("Create overwrote an existing file — must refuse")
	}
	if readFile(t, path) != "PRECIOUS=1\n" {
		t.Error("existing file was modified")
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
