package cli

import (
	"reflect"
	"testing"
)

func TestTokenize(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		want    []string
		wantErr bool
	}{
		{"empty", "", nil, false},
		{"simple", "posts list", []string{"posts", "list"}, false},
		{"double quoted", `posts create --title "hello world" --body "b"`,
			[]string{"posts", "create", "--title", "hello world", "--body", "b"}, false},
		{"single quoted", `posts create --title 'hello world'`,
			[]string{"posts", "create", "--title", "hello world"}, false},
		{"extra whitespace", "  posts   list  ", []string{"posts", "list"}, false},
		{"unterminated quote", `posts create --title "oops`, nil, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tokenize(tc.line)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error for %q, got none", tc.line)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.line, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("tokenize(%q) = %#v, want %#v", tc.line, got, tc.want)
			}
		})
	}
}
