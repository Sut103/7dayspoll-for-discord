package poll

import (
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Calls go through pollSession, not the fake, so the interface itself must expose these.
func TestPollSessionExposesPollEndMethods(t *testing.T) {
	wantMessage := &discordgo.Message{ID: "message-1"}
	fake := &fakePollSession{
		pollExpireResult:              wantMessage,
		channelMessageResult:          wantMessage,
		channelMessagesResult:         []*discordgo.Message{wantMessage},
		interactionResponseEditResult: wantMessage,
	}
	var session pollSession = fake
	interaction := &discordgo.Interaction{}

	if got, err := session.PollExpire("channel-1", "message-1"); err != nil || got != wantMessage {
		t.Fatalf("PollExpire() = (%v, %v), want (%v, nil)", got, err, wantMessage)
	}
	if fake.pollExpireChannelID != "channel-1" || fake.pollExpireMessageID != "message-1" {
		t.Errorf("PollExpire recorded (%q, %q), want (%q, %q)", fake.pollExpireChannelID, fake.pollExpireMessageID, "channel-1", "message-1")
	}

	if got, err := session.ChannelMessage("channel-1", "message-1"); err != nil || got != wantMessage {
		t.Fatalf("ChannelMessage() = (%v, %v), want (%v, nil)", got, err, wantMessage)
	}
	if fake.channelMessageChannelID != "channel-1" || fake.channelMessageMessageID != "message-1" {
		t.Errorf("ChannelMessage recorded (%q, %q), want (%q, %q)", fake.channelMessageChannelID, fake.channelMessageMessageID, "channel-1", "message-1")
	}

	if got, err := session.ChannelMessages("channel-1", 100, "", "", ""); err != nil || len(got) != 1 {
		t.Fatalf("ChannelMessages() = (%v, %v), want 1 message and no error", got, err)
	}
	if fake.channelMessagesChannelID != "channel-1" || fake.channelMessagesLimit != 100 {
		t.Errorf("ChannelMessages recorded (%q, %d), want (%q, %d)", fake.channelMessagesChannelID, fake.channelMessagesLimit, "channel-1", 100)
	}

	content := "edited"
	if got, err := session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{Content: &content}); err != nil || got != wantMessage {
		t.Fatalf("InteractionResponseEdit() = (%v, %v), want (%v, nil)", got, err, wantMessage)
	}
	if fake.interactionResponseEditData == nil || fake.interactionResponseEditData.Content == nil || *fake.interactionResponseEditData.Content != content {
		t.Errorf("InteractionResponseEdit recorded %+v, want Content %q", fake.interactionResponseEditData, content)
	}

	wantCalls := []string{"PollExpire", "ChannelMessage", "ChannelMessages", "InteractionResponseEdit"}
	if len(fake.calls) != len(wantCalls) {
		t.Fatalf("calls = %v, want %v", fake.calls, wantCalls)
	}
	for i, want := range wantCalls {
		if fake.calls[i] != want {
			t.Fatalf("calls = %v, want %v", fake.calls, wantCalls)
		}
	}
}

func TestUserID(t *testing.T) {
	tests := []struct {
		name        string
		interaction *discordgo.Interaction
		want        string
	}{
		{
			name:        "guild interaction uses Member.User.ID",
			interaction: &discordgo.Interaction{Member: &discordgo.Member{User: &discordgo.User{ID: "member-1"}}},
			want:        "member-1",
		},
		{
			name:        "DM interaction falls back to User.ID",
			interaction: &discordgo.Interaction{User: &discordgo.User{ID: "dm-user-1"}},
			want:        "dm-user-1",
		},
		{
			name: "Member wins when both are set",
			interaction: &discordgo.Interaction{
				Member: &discordgo.Member{User: &discordgo.User{ID: "member-1"}},
				User:   &discordgo.User{ID: "dm-user-1"},
			},
			want: "member-1",
		},
		{
			name: "Member without User falls back to User",
			interaction: &discordgo.Interaction{
				Member: &discordgo.Member{},
				User:   &discordgo.User{ID: "dm-user-1"},
			},
			want: "dm-user-1",
		},
		{
			name:        "neither set yields empty string",
			interaction: &discordgo.Interaction{},
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := userID(tt.interaction); got != tt.want {
				t.Errorf("userID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPollCreatorID(t *testing.T) {
	tests := []struct {
		name    string
		message *discordgo.Message
		want    string
	}{
		{
			name: "InteractionMetadata.User is the poll creator",
			message: &discordgo.Message{
				InteractionMetadata: &discordgo.MessageInteractionMetadata{User: &discordgo.User{ID: "creator-1"}},
			},
			want: "creator-1",
		},
		{
			name:    "missing InteractionMetadata yields empty string",
			message: &discordgo.Message{},
			want:    "",
		},
		{
			name: "InteractionMetadata without User yields empty string",
			message: &discordgo.Message{
				InteractionMetadata: &discordgo.MessageInteractionMetadata{},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pollCreatorID(tt.message); got != tt.want {
				t.Errorf("pollCreatorID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHasManageMessages(t *testing.T) {
	tests := []struct {
		name        string
		interaction *discordgo.Interaction
		want        bool
	}{
		{
			name: "member with the Manage Messages bit",
			interaction: &discordgo.Interaction{
				Member: &discordgo.Member{Permissions: discordgo.PermissionManageMessages},
			},
			want: true,
		},
		{
			name: "member with Manage Messages among other permissions",
			interaction: &discordgo.Interaction{
				Member: &discordgo.Member{Permissions: discordgo.PermissionManageMessages | discordgo.PermissionSendMessages},
			},
			want: true,
		},
		{
			name: "member without the bit",
			interaction: &discordgo.Interaction{
				Member: &discordgo.Member{Permissions: discordgo.PermissionSendMessages},
			},
			want: false,
		},
		{
			name:        "DM has no Member, so never true",
			interaction: &discordgo.Interaction{User: &discordgo.User{ID: "dm-user-1"}},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasManageMessages(tt.interaction); got != tt.want {
				t.Errorf("hasManageMessages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanEndPoll(t *testing.T) {
	pollByCreator := &discordgo.Message{
		InteractionMetadata: &discordgo.MessageInteractionMetadata{User: &discordgo.User{ID: "creator-1"}},
	}

	tests := []struct {
		name        string
		interaction *discordgo.Interaction
		message     *discordgo.Message
		want        bool
	}{
		{
			name: "the poll's creator can end it without any permission",
			interaction: &discordgo.Interaction{
				Member: &discordgo.Member{User: &discordgo.User{ID: "creator-1"}},
			},
			message: pollByCreator,
			want:    true,
		},
		{
			name: "a member with Manage Messages can end someone else's poll",
			interaction: &discordgo.Interaction{
				Member: &discordgo.Member{
					User:        &discordgo.User{ID: "moderator-1"},
					Permissions: discordgo.PermissionManageMessages,
				},
			},
			message: pollByCreator,
			want:    true,
		},
		{
			name: "a member who is neither cannot end it",
			interaction: &discordgo.Interaction{
				Member: &discordgo.Member{
					User:        &discordgo.User{ID: "bystander-1"},
					Permissions: discordgo.PermissionSendMessages,
				},
			},
			message: pollByCreator,
			want:    false,
		},
		{
			name:        "in a DM the creator is still recognized",
			interaction: &discordgo.Interaction{User: &discordgo.User{ID: "creator-1"}},
			message:     pollByCreator,
			want:        true,
		},
		{
			name:        "in a DM a non-creator has no Manage Messages to fall back on",
			interaction: &discordgo.Interaction{User: &discordgo.User{ID: "someone-else"}},
			message:     pollByCreator,
			want:        false,
		},
		{
			name:        "an unidentifiable invoker can never end a poll",
			interaction: &discordgo.Interaction{},
			message:     pollByCreator,
			want:        false,
		},
		{
			name: "an unknown creator does not match an unidentifiable invoker",
			interaction: &discordgo.Interaction{
				Member: &discordgo.Member{User: &discordgo.User{ID: "bystander-1"}},
			},
			message: &discordgo.Message{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canEndPoll(tt.interaction, tt.message); got != tt.want {
				t.Errorf("canEndPoll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsFinalized(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)

	tests := []struct {
		name string
		poll *discordgo.Poll
		want bool
	}{
		{
			name: "Results.Finalized is the primary signal",
			poll: &discordgo.Poll{Results: &discordgo.PollResults{Finalized: true}, Expiry: &future},
			want: true,
		},
		{
			name: "a passed Expiry is the fallback when Results is nil",
			poll: &discordgo.Poll{Expiry: &past},
			want: true,
		},
		{
			name: "a future Expiry with no Results means still running",
			poll: &discordgo.Poll{Expiry: &future},
			want: false,
		},
		{
			name: "neither Results nor Expiry means still running",
			poll: &discordgo.Poll{},
			want: false,
		},
		{
			name: "a passed Expiry still counts when Results says not finalized",
			poll: &discordgo.Poll{Results: &discordgo.PollResults{Finalized: false}, Expiry: &past},
			want: true,
		},
		{
			name: "not finalized and not expired means still running",
			poll: &discordgo.Poll{Results: &discordgo.PollResults{Finalized: false}, Expiry: &future},
			want: false,
		},
		{
			name: "a nil poll is not finalized (callers guard for its absence separately)",
			poll: nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFinalized(tt.poll); got != tt.want {
				t.Errorf("isFinalized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveMessageID(t *testing.T) {
	const currentChannel = "111111111111111111"

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "a bare 19-digit ID passes through",
			input: "222222222222222222",
			want:  "222222222222222222",
		},
		{
			name:  "the shortest accepted ID (17 digits)",
			input: "12345678901234567",
			want:  "12345678901234567",
		},
		{
			name:  "the longest accepted ID (20 digits)",
			input: "12345678901234567890",
			want:  "12345678901234567890",
		},
		{
			name:  "surrounding whitespace is trimmed",
			input: "  222222222222222222  ",
			want:  "222222222222222222",
		},
		{
			name:  "a discord.com message link in this channel",
			input: "https://discord.com/channels/999999999999999999/" + currentChannel + "/222222222222222222",
			want:  "222222222222222222",
		},
		{
			name:  "a canary.discord.com link",
			input: "https://canary.discord.com/channels/999999999999999999/" + currentChannel + "/222222222222222222",
			want:  "222222222222222222",
		},
		{
			name:  "a ptb.discord.com link",
			input: "https://ptb.discord.com/channels/999999999999999999/" + currentChannel + "/222222222222222222",
			want:  "222222222222222222",
		},
		{
			name:  "a legacy discordapp.com link",
			input: "https://discordapp.com/channels/999999999999999999/" + currentChannel + "/222222222222222222",
			want:  "222222222222222222",
		},
		{
			name:  "a DM link uses @me in place of a guild ID",
			input: "https://discord.com/channels/@me/" + currentChannel + "/222222222222222222",
			want:  "222222222222222222",
		},
		{
			name:    "a link pointing at another channel is rejected",
			input:   "https://discord.com/channels/999999999999999999/333333333333333333/222222222222222222",
			wantErr: true,
		},
		{
			name:    "an ID that is too short is rejected",
			input:   "1234567890123456",
			wantErr: true,
		},
		{
			name:    "an ID that is too long is rejected",
			input:   "123456789012345678901",
			wantErr: true,
		},
		{
			name:    "a non-numeric string is rejected",
			input:   "not-a-message",
			wantErr: true,
		},
		{
			name:    "an empty string is rejected",
			input:   "",
			wantErr: true,
		},
		{
			name:    "a non-Discord host is rejected",
			input:   "https://example.com/channels/999999999999999999/" + currentChannel + "/222222222222222222",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveMessageID(tt.input, currentChannel)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolveMessageID(%q) = %q, want an error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveMessageID(%q) returned unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("resolveMessageID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
