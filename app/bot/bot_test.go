package bot

import "testing"

func TestResolveCommandKind(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKind string
		wantOk   bool
	}{
		{
			name:     "poll maps to native kind",
			input:    "poll",
			wantKind: commandKindNative,
			wantOk:   true,
		},
		{
			name:     "poll-classic maps to classic kind",
			input:    "poll-classic",
			wantKind: commandKindClassic,
			wantOk:   true,
		},
		{
			name:   "unknown command is not ok",
			input:  "unknown-command",
			wantOk: false,
		},
		{
			name:   "empty string is not ok",
			input:  "",
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKind, gotOk := resolveCommandKind(tt.input)
			if gotOk != tt.wantOk {
				t.Fatalf("resolveCommandKind(%q) ok = %v, want %v", tt.input, gotOk, tt.wantOk)
			}
			if tt.wantOk && gotKind != tt.wantKind {
				t.Fatalf("resolveCommandKind(%q) kind = %q, want %q", tt.input, gotKind, tt.wantKind)
			}
		})
	}
}
