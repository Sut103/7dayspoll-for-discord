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

func TestResolveCommandKind_PollEnd(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKind string
	}{
		{
			name:     "the message command maps to the message end-poll kind",
			input:    "End Poll",
			wantKind: commandKindEndPollMessage,
		},
		{
			name:     "poll-end maps to the slash end-poll kind",
			input:    "poll-end",
			wantKind: commandKindEndPollSlash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKind, gotOk := resolveCommandKind(tt.input)
			if !gotOk {
				t.Fatalf("resolveCommandKind(%q) ok = false, want true", tt.input)
			}
			if gotKind != tt.wantKind {
				t.Fatalf("resolveCommandKind(%q) kind = %q, want %q", tt.input, gotKind, tt.wantKind)
			}
		})
	}
}

func TestResolveAutocompleteKind(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantOk bool
	}{
		{name: "poll-end has autocomplete", input: "poll-end", wantOk: true},
		{name: "poll has no autocomplete", input: "poll", wantOk: false},
		{name: "unknown command has no autocomplete", input: "unknown-command", wantOk: false},
		{name: "empty string has no autocomplete", input: "", wantOk: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotOk := supportsAutocomplete(tt.input); gotOk != tt.wantOk {
				t.Fatalf("supportsAutocomplete(%q) = %v, want %v", tt.input, gotOk, tt.wantOk)
			}
		})
	}
}
