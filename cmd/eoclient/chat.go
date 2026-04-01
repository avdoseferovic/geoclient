package main

import (
	"image"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/avdo/eoweb/internal/game"
	clientui "github.com/avdo/eoweb/internal/ui"
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
	bus := g.client.GetBus()
	if bus == nil {
		return
	}

	// PM: starts with "!" e.g. "!player message"
	if strings.HasPrefix(msg, "!") {
		parts := strings.SplitN(msg[1:], " ", 2)
		if len(parts) == 2 {
			if err := bus.SendSequenced(&client.TalkTellClientPacket{
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
	if err := bus.SendSequenced(&client.TalkRequestClientPacket{
		Message: msg,
	}); err != nil {
		slog.Error("send chat failed", "err", err)
	}
	g.addChatMessage(g.client.Character.Name + ": " + msg)
}

func (g *Game) drawChat(screen *ebiten.Image, theme clientui.Theme) {
	panelRect := image.Rect(10, screenHeight-150, 392, screenHeight-10)
	clientui.DrawPanel(screen, panelRect, theme, clientui.PanelOptions{Title: "Chat", Accent: theme.AccentMuted})

	inputRect := image.Rect(panelRect.Min.X+12, panelRect.Max.Y-36, panelRect.Max.X-12, panelRect.Max.Y-12)
	clientui.DrawInset(screen, inputRect, theme, g.chat.Typing)

	historyWidth := inputRect.Dx() - 4
	maxHistoryLines := max(1, (inputRect.Min.Y-(panelRect.Min.Y+24))/14)
	wrappedHistory := make([]string, 0, len(g.chat.History))
	for _, msg := range g.chat.History {
		wrappedHistory = append(wrappedHistory, wrapChatLines(msg, historyWidth)...)
	}
	if len(wrappedHistory) > maxHistoryLines {
		wrappedHistory = wrappedHistory[len(wrappedHistory)-maxHistoryLines:]
	}

	y := inputRect.Min.Y - 10
	for i := len(wrappedHistory) - 1; i >= 0 && y > panelRect.Min.Y+18; i-- {
		clientui.DrawText(screen, wrappedHistory[i], panelRect.Min.X+14, y, theme.Text)
		y -= 14
	}

	inputText := "Press Enter to speak"
	inputColor := theme.TextDim
	if g.chat.Typing {
		inputText = "> " + g.chat.Input
		if g.overlay.ticks%40 < 20 {
			inputText += "_"
		}
		inputColor = theme.Text
	}
	inputText = trimChatLineLeft(inputText, inputRect.Dx()-20)
	clientui.DrawText(screen, inputText, inputRect.Min.X+10, inputRect.Min.Y+16, inputColor)
}

func wrapChatLines(text string, maxWidth int) []string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return []string{""}
	}

	words := strings.Fields(trimmed)
	if len(words) == 0 {
		return []string{trimmed}
	}

	lines := make([]string, 0, len(words))
	current := words[0]
	for _, word := range words[1:] {
		candidate := current + " " + word
		if clientui.MeasureText(candidate) <= maxWidth {
			current = candidate
			continue
		}
		lines = append(lines, splitOversizedChatToken(current, maxWidth)...)
		current = word
	}
	lines = append(lines, splitOversizedChatToken(current, maxWidth)...)
	return lines
}

func splitOversizedChatToken(token string, maxWidth int) []string {
	if clientui.MeasureText(token) <= maxWidth {
		return []string{token}
	}

	lines := make([]string, 0, len(token)/4+1)
	current := ""
	for _, r := range token {
		candidate := current + string(r)
		if current != "" && clientui.MeasureText(candidate) > maxWidth {
			lines = append(lines, current)
			current = string(r)
			continue
		}
		current = candidate
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func trimChatLineLeft(text string, maxWidth int) string {
	if clientui.MeasureText(text) <= maxWidth {
		return text
	}

	trimmed := text
	for trimmed != "" && clientui.MeasureText("..."+trimmed) > maxWidth {
		_, size := utf8.DecodeRuneInString(trimmed)
		if size <= 0 {
			break
		}
		trimmed = trimmed[size:]
	}
	return "..." + trimmed
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// handleChatEvent processes incoming chat events from the server.
func (g *Game) handleChatEvent(evt game.Event) {
	g.addChatMessage(evt.Message)
}
