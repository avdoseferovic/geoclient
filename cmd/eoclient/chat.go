package main

import (
	"log/slog"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/avdo/eoweb/internal/game"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"
)

const (
	maxChatHistory  = 20
	chatInputMaxLen = 128
)

// ChatState holds chat UI state.
type ChatState struct {
	History  []string
	Input    string
	Typing   bool
	inputBuf []rune
}

func (g *Game) updateChat() {
	// Toggle typing mode with Enter
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if g.chat.Typing {
			// Send message
			msg := strings.TrimSpace(g.chat.Input)
			if msg != "" {
				g.sendChat(msg)
			}
			g.chat.Input = ""
			g.chat.inputBuf = g.chat.inputBuf[:0]
			g.chat.Typing = false
		} else {
			g.chat.Typing = true
		}
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) && g.chat.Typing {
		g.chat.Input = ""
		g.chat.inputBuf = g.chat.inputBuf[:0]
		g.chat.Typing = false
		return
	}

	if !g.chat.Typing {
		return
	}

	// Backspace
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.chat.inputBuf) > 0 {
		g.chat.inputBuf = g.chat.inputBuf[:len(g.chat.inputBuf)-1]
		g.chat.Input = string(g.chat.inputBuf)
		return
	}

	// Capture typed characters
	runes := ebiten.AppendInputChars(nil)
	for _, r := range runes {
		if len(g.chat.inputBuf) < chatInputMaxLen {
			g.chat.inputBuf = append(g.chat.inputBuf, r)
		}
	}
	g.chat.Input = string(g.chat.inputBuf)
}

func (g *Game) addChatMessage(msg string) {
	g.chat.History = append(g.chat.History, msg)
	if len(g.chat.History) > maxChatHistory {
		g.chat.History = g.chat.History[len(g.chat.History)-maxChatHistory:]
	}
}

func (g *Game) sendChat(msg string) {
	if g.client.Bus == nil {
		return
	}

	// PM: starts with "!" e.g. "!player message"
	if strings.HasPrefix(msg, "!") {
		parts := strings.SplitN(msg[1:], " ", 2)
		if len(parts) == 2 {
			if err := g.client.Bus.SendSequenced(&client.TalkTellClientPacket{
				Name:    parts[0],
				Message: parts[1],
			}); err != nil {
				slog.Error("send PM failed", "err", err)
			}
			g.addChatMessage("[To " + parts[0] + "]: " + parts[1])
			return
		}
	}

	// Public chat
	if err := g.client.Bus.SendSequenced(&client.TalkRequestClientPacket{
		Message: msg,
	}); err != nil {
		slog.Error("send chat failed", "err", err)
	}
	g.addChatMessage(g.client.Character.Name + ": " + msg)
}

func (g *Game) drawChat(screen *ebiten.Image) {
	// Draw chat history
	y := screenHeight - 40
	for i := len(g.chat.History) - 1; i >= 0 && y > screenHeight-200; i-- {
		ebitenutil.DebugPrintAt(screen, g.chat.History[i], 4, y)
		y -= 14
	}

	// Draw input bar
	if g.chat.Typing {
		inputText := "> " + g.chat.Input + "_"
		ebitenutil.DebugPrintAt(screen, inputText, 4, screenHeight-24)
	} else {
		ebitenutil.DebugPrintAt(screen, "[Enter] to chat", 4, screenHeight-24)
	}
}

// handleChatEvent processes incoming chat events from the server.
func (g *Game) handleChatEvent(evt game.Event) {
	g.addChatMessage(evt.Message)
}
