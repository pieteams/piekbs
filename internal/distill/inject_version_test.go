//go:build fts5

package distill

import "testing"

func TestInjectDistillVersion(t *testing.T) {
	cases := []struct {
		name string
		text string
		ver  string
		want string
	}{
		{
			name: "adds version to frontmatter",
			text: "---\ntitle: T\nsources:\n  - raw/x.md\n---\nBody",
			ver:  "0.4.7",
			want: "---\ntitle: T\nsources:\n  - raw/x.md\ndistill_version: 0.4.7\n---\nBody",
		},
		{
			name: "no frontmatter unchanged",
			text: "Body only",
			ver:  "0.4.7",
			want: "Body only",
		},
		{
			name: "unclosed frontmatter unchanged",
			text: "---\ntitle: T\nBody",
			ver:  "0.4.7",
			want: "---\ntitle: T\nBody",
		},
		{
			name: "already present not duplicated",
			text: "---\ntitle: T\ndistill_version: 0.4.6\n---\nBody",
			ver:  "0.4.7",
			want: "---\ntitle: T\ndistill_version: 0.4.6\n---\nBody",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := injectDistillVersion(tc.text, tc.ver)
			if got != tc.want {
				t.Errorf("injectDistillVersion()\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}
