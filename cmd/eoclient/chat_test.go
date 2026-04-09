package main

import (
	"testing"

	"github.com/avdoseferovic/geoclient/internal/game"
)

func TestParseOutgoingChatUsesPrefixes(t *testing.T) {
	tests := []struct {
		name          string
		message       string
		activeChannel game.ChatChannel
		wantOK        bool
		wantMode      chatSendMode
		wantTarget    string
		wantMessage   string
	}{
		{
			name:          "tell prefix",
			message:       "!alice hello there",
			activeChannel: game.ChatChannelMap,
			wantOK:        true,
			wantMode:      chatSendTell,
			wantTarget:    "alice",
			wantMessage:   "hello there",
		},
		{
			name:          "global prefix",
			message:       "~hello world",
			activeChannel: game.ChatChannelMap,
			wantOK:        true,
			wantMode:      chatSendGlobal,
			wantMessage:   "hello world",
		},
		{
			name:          "group prefix",
			message:       "'party up",
			activeChannel: game.ChatChannelMap,
			wantOK:        true,
			wantMode:      chatSendGroup,
			wantMessage:   "party up",
		},
		{
			name:          "system tab blocks send",
			message:       "cannot send here",
			activeChannel: game.ChatChannelSystem,
			wantOK:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseOutgoingChat(tt.message, tt.activeChannel)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got.Mode != tt.wantMode {
				t.Fatalf("mode = %v, want %v", got.Mode, tt.wantMode)
			}
			if got.Target != tt.wantTarget {
				t.Fatalf("target = %q, want %q", got.Target, tt.wantTarget)
			}
			if got.Message != tt.wantMessage {
				t.Fatalf("message = %q, want %q", got.Message, tt.wantMessage)
			}
		})
	}
}

func TestParseOutgoingChatUsesActiveChannel(t *testing.T) {
	tests := []struct {
		name          string
		activeChannel game.ChatChannel
		wantMode      chatSendMode
	}{
		{name: "map", activeChannel: game.ChatChannelMap, wantMode: chatSendMap},
		{name: "group", activeChannel: game.ChatChannelGroup, wantMode: chatSendGroup},
		{name: "global", activeChannel: game.ChatChannelGlobal, wantMode: chatSendGlobal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseOutgoingChat("hello", tt.activeChannel)
			if !ok {
				t.Fatal("parseOutgoingChat returned false")
			}
			if got.Mode != tt.wantMode {
				t.Fatalf("mode = %v, want %v", got.Mode, tt.wantMode)
			}
			if got.Message != "hello" {
				t.Fatalf("message = %q, want hello", got.Message)
			}
		})
	}
}

func TestHandleChatEventRoutesByChannel(t *testing.T) {
	g := &Game{chat: newChatState()}

	g.handleChatEvent(game.Event{
		Type: game.EventChat,
		Data: game.ChatMessage{Channel: game.ChatChannelGlobal, Text: "Alice: hello"},
	})
	g.handleChatEvent(game.Event{
		Type: game.EventChat,
		Data: game.ChatMessage{Channel: game.ChatChannelSystem, Text: "[Server] welcome"},
	})

	if got := g.chat.History[game.ChatChannelGlobal]; len(got) != 1 || got[0] != "Alice: hello" {
		t.Fatalf("global history = %#v, want one global message", got)
	}
	if got := g.chat.History[game.ChatChannelSystem]; len(got) != 1 || got[0] != "[Server] welcome" {
		t.Fatalf("system history = %#v, want one system message", got)
	}
	if got := g.chat.History[game.ChatChannelMap]; len(got) != 0 {
		t.Fatalf("map history = %#v, want empty", got)
	}
}

func TestVisibleChatLinesAppliesScrollOffset(t *testing.T) {
	lines := []string{"l1", "l2", "l3", "l4", "l5"}

	visible, offset, fraction := visibleChatLines(lines, 3, 0)
	if offset != 0 {
		t.Fatalf("offset(bottom) = %v, want 0", offset)
	}
	if fraction != 0 {
		t.Fatalf("fraction(bottom) = %v, want 0", fraction)
	}
	wantBottom := []string{"l3", "l4", "l5"}
	for i := range wantBottom {
		if visible[i] != wantBottom[i] {
			t.Fatalf("visible(bottom) = %#v, want %#v", visible, wantBottom)
		}
	}

	visible, offset, fraction = visibleChatLines(lines, 3, 2)
	if offset != 2 {
		t.Fatalf("offset(scrolled) = %v, want 2", offset)
	}
	if fraction != 0 {
		t.Fatalf("fraction(scrolled) = %v, want 0", fraction)
	}
	wantScrolled := []string{"l1", "l2", "l3"}
	for i := range wantScrolled {
		if visible[i] != wantScrolled[i] {
			t.Fatalf("visible(scrolled) = %#v, want %#v", visible, wantScrolled)
		}
	}
}

func TestVisibleChatLinesClampsOverscroll(t *testing.T) {
	lines := []string{"l1", "l2", "l3", "l4"}
	visible, offset, fraction := visibleChatLines(lines, 2, 10)
	if offset != 2 {
		t.Fatalf("offset = %v, want 2", offset)
	}
	if fraction != 0 {
		t.Fatalf("fraction = %v, want 0", fraction)
	}
	want := []string{"l1", "l2"}
	for i := range want {
		if visible[i] != want[i] {
			t.Fatalf("visible = %#v, want %#v", visible, want)
		}
	}
}

func TestVisibleChatLinesReturnsFractionalOffset(t *testing.T) {
	lines := []string{"l1", "l2", "l3", "l4", "l5"}
	visible, offset, fraction := visibleChatLines(lines, 3, 1.5)
	if offset != 1.5 {
		t.Fatalf("offset = %v, want 1.5", offset)
	}
	if fraction != 0.5 {
		t.Fatalf("fraction = %v, want 0.5", fraction)
	}
	want := []string{"l1", "l2", "l3", "l4"}
	for i := range want {
		if visible[i] != want[i] {
			t.Fatalf("visible = %#v, want %#v", visible, want)
		}
	}
}
