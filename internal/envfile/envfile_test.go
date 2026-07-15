package envfile

import (
	"reflect"
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
		input: "FOO=bar",
		want:  map[string]string{"FOO": "bar"},
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
