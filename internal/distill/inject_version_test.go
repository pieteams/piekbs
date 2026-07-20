//go:build fts5

package distill

import "testing"

func TestInjectSchemaVersion(t *testing.T) {
	cases := []struct {
		name string
		text string
		ver  int
		want string
	}{
		{
			name: "adds version to frontmatter",
			text: "---\ntitle: T\nsources:\n  - raw/x.md\n---\nBody",
			ver:  2,
			want: "---\ntitle: T\nsources:\n  - raw/x.md\nschema_version: 2\n---\nBody",
		},
		{
			name: "no frontmatter unchanged",
			text: "Body only",
			ver:  2,
			want: "Body only",
		},
		{
			name: "unclosed frontmatter unchanged",
			text: "---\ntitle: T\nBody",
			ver:  2,
			want: "---\ntitle: T\nBody",
		},
		{
			name: "already present not duplicated",
			text: "---\ntitle: T\nschema_version: 1\n---\nBody",
			ver:  2,
			want: "---\ntitle: T\nschema_version: 1\n---\nBody",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := injectSchemaVersion(tc.text, tc.ver)
			if got != tc.want {
				t.Errorf("injectSchemaVersion()\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}
