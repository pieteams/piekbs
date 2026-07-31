//go:build fts5

package main

import "testing"

// Guards the arg-index convention for subcommand actions: flag.Args() includes
// the subcommand name at args[0], so an action word lives at args[1]. Checking
// args[0] made `piekbs schema upgrade` unreachable — runSchemaUpgrade was dead
// code and every invocation fell through to the usage error.
func TestIsSchemaUpgrade(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"schema upgrade", []string{"schema", "upgrade"}, true},
		{"schema upgrade with trailing args", []string{"schema", "upgrade", "--force"}, true},
		{"schema alone", []string{"schema"}, false},
		{"schema with other action", []string{"schema", "status"}, false},
		{"empty", nil, false},
		{"subcommand name only at args[0]", []string{"upgrade"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSchemaUpgrade(tt.args); got != tt.want {
				t.Errorf("isSchemaUpgrade(%q) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
