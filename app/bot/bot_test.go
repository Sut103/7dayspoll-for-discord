package bot

import "testing"

func TestNewBot(t *testing.T) {
	b := NewBot("my-token")
	if b.token != "my-token" {
		t.Errorf("token = %q, want %q", b.token, "my-token")
	}
}

func TestResolveCommandKind(t *testing.T) {
	tests := []struct {
		name string
		want commandKind
	}{
		{"poll", commandNativePoll},
		{"poll-classic", commandClassicPoll},
		{"unknown-command", commandUnknown},
		{"", commandUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveCommandKind(tt.name); got != tt.want {
				t.Errorf("resolveCommandKind(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
