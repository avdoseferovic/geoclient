package main

import (
	"fmt"
	"image"
	"log/slog"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/avdoseferovic/geoclient/internal/game"
	"github.com/avdoseferovic/geoclient/internal/ui/overlay"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"
)

const (
	maxChatHistory   = 20
	chatInputMaxLen  = 128
	chatTabHeight    = 24
	chatLineHeight   = 14
	chatWheelStep    = 2.0
	chatWheelFineMax = 0.15
	chatWheelLatch   = 18
)

var chatChannels = [...]game.ChatChannel{
	game.ChatChannelMap,
	game.ChatChannelGroup,
	game.ChatChannelGlobal,
	game.ChatChannelSystem,
}

// ChatState holds chat UI state.
type ChatState struct {
	History       map[game.ChatChannel][]string
	Scroll        map[game.ChatChannel]float64
	ScrollTarget  map[game.ChatChannel]float64
	ActiveChannel game.ChatChannel
	DragChannel   game.ChatChannel
	Dragging      bool
	DragOffsetY   int
	WheelLatch    int
	WheelSign     int
	Input         string
	Typing        bool
	inputBuf      []rune

	// Wrapped history cache (invalidated on new message or width change)
	wrappedCache    map[game.ChatChannel][]string
	wrappedWidth    map[game.ChatChannel]int
	wrappedMsgCount map[game.ChatChannel]int
}

type chatSendMode int

const (
	chatSendMap chatSendMode = iota
	chatSendGroup
	chatSendGlobal
	chatSendTell
)

type outgoingChat struct {
	Mode    chatSendMode
	Target  string
	Message string
}

func newChatState() ChatState {
	history := make(map[game.ChatChannel][]string, len(chatChannels))
	scroll := make(map[game.ChatChannel]float64, len(chatChannels))
	scrollTarget := make(map[game.ChatChannel]float64, len(chatChannels))
	for _, channel := range chatChannels {
		history[channel] = nil
		scroll[channel] = 0
		scrollTarget[channel] = 0
	}
	return ChatState{
		History:       history,
		Scroll:        scroll,
		ScrollTarget:  scrollTarget,
		ActiveChannel: game.ChatChannelMap,
	}
}

func (g *Game) updateChat() {
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) && !g.chat.Typing {
		g.cycleChatChannel(1)
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if g.chat.Typing {
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

	g.updateChatScroll()

	if !g.chat.Typing {
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.chat.inputBuf) > 0 {
		g.chat.inputBuf = g.chat.inputBuf[:len(g.chat.inputBuf)-1]
		g.chat.Input = string(g.chat.inputBuf)
		return
	}

	runes := ebiten.AppendInputChars(nil)
	for _, r := range runes {
		if len(g.chat.inputBuf) < chatInputMaxLen {
			g.chat.inputBuf = append(g.chat.inputBuf, r)
		}
	}
	g.chat.Input = string(g.chat.inputBuf)
}

func (g *Game) setActiveChatChannel(channel game.ChatChannel) {
	if g.chat.History == nil {
		g.chat = newChatState()
	}
	g.chat.ActiveChannel = channel
}

func (g *Game) cycleChatChannel(delta int) {
	index := 0
	for i, channel := range chatChannels {
		if channel == g.chat.ActiveChannel {
			index = i
			break
		}
	}
	index = (index + delta + len(chatChannels)) % len(chatChannels)
	g.setActiveChatChannel(chatChannels[index])
}

func (g *Game) addChatMessage(channel game.ChatChannel, msg string) {
	if g.chat.History == nil {
		g.chat = newChatState()
	}
	g.chat.History[channel] = append(g.chat.History[channel], msg)
	if len(g.chat.History[channel]) > maxChatHistory {
		g.chat.History[channel] = g.chat.History[channel][len(g.chat.History[channel])-maxChatHistory:]
	}
}

func (g *Game) updateChatScroll() {
	if g.chat.Scroll == nil || g.chat.ScrollTarget == nil || g.chat.History == nil {
		g.chat = newChatState()
	}
	panelRect, inputRect := g.chatRects()
	historyRect := g.chatHistoryRect(panelRect, inputRect)
	mx, my := ebiten.CursorPosition()
	if g.chat.WheelLatch > 0 {
		g.chat.WheelLatch--
	} else {
		g.chat.WheelSign = 0
	}
	scrollbarRect := chatScrollbarRect(historyRect)
	thumbRect, ok := chatScrollbarThumbRect(historyRect, len(g.chatWrappedHistory(g.chat.ActiveChannel, historyRect.Dx()-14)), g.chatVisibleLineCount(inputRect), g.chat.Scroll[g.chat.ActiveChannel])

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		switch {
		case ok && overlay.PointInRect(mx, my, thumbRect):
			g.chat.Dragging = true
			g.chat.DragChannel = g.chat.ActiveChannel
			g.chat.DragOffsetY = my - thumbRect.Min.Y
			return
		case overlay.PointInRect(mx, my, scrollbarRect):
			g.setChatScrollFromScrollbar(my, historyRect, inputRect, scrollbarRect.Dy()/2, true)
			g.chat.Dragging = true
			g.chat.DragChannel = g.chat.ActiveChannel
			if thumbRect, ok := chatScrollbarThumbRect(historyRect, len(g.chatWrappedHistory(g.chat.ActiveChannel, historyRect.Dx()-14)), g.chatVisibleLineCount(inputRect), g.chat.ScrollTarget[g.chat.ActiveChannel]); ok {
				g.chat.DragOffsetY = my - thumbRect.Min.Y
			}
			return
		}
	}

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		g.chat.Dragging = false
	}

	if g.chat.Dragging {
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			g.chat.Dragging = false
		} else if g.chat.DragChannel == g.chat.ActiveChannel {
			g.setChatScrollFromScrollbar(my, historyRect, inputRect, g.chat.DragOffsetY, true)
			return
		}
	}

	if overlay.PointInRect(mx, my, panelRect) {
		_, wheelY := ebiten.Wheel()
		if wheelY != 0 {
			if delta, coarse, sign := normalizedChatWheelDelta(wheelY); delta != 0 {
				if coarse {
					if g.chat.WheelLatch == 0 || g.chat.WheelSign != sign {
						g.scrollChatLines(delta)
						g.chat.WheelLatch = chatWheelLatch
						g.chat.WheelSign = sign
					}
				} else {
					g.scrollChatLines(delta)
				}
			}
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		g.scrollChatLines(6)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		g.scrollChatLines(-6)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		g.scrollChatLines(g.maxChatScroll(inputRect))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
		g.scrollChatLines(-g.chat.ScrollTarget[g.chat.ActiveChannel])
	}

	channel := g.chat.ActiveChannel
	maxScroll := g.maxChatScrollForActiveChannel()
	g.chat.ScrollTarget[channel] = clampChatScroll(g.chat.ScrollTarget[channel], maxScroll)
	current := g.chat.Scroll[channel]
	target := g.chat.ScrollTarget[channel]
	if math.Abs(target-current) < 0.02 {
		g.chat.Scroll[channel] = target
		return
	}
	g.chat.Scroll[channel] = current + (target-current)*0.25
}

func (g *Game) scrollChatLines(delta float64) {
	if delta == 0 {
		return
	}
	if g.chat.History == nil {
		g.chat = newChatState()
	}
	channel := g.chat.ActiveChannel
	next := g.chat.ScrollTarget[channel] + delta
	maxScroll := g.maxChatScrollForActiveChannel()
	g.chat.ScrollTarget[channel] = clampChatScroll(next, maxScroll)
}

func (g *Game) sendChat(msg string) {
	bus := g.client.GetBus()
	if bus == nil {
		return
	}

	outgoing, ok := parseOutgoingChat(msg, g.chat.ActiveChannel)
	if !ok {
		g.addChatMessage(game.ChatChannelSystem, "[System] System channel is receive-only.")
		return
	}

	switch outgoing.Mode {
	case chatSendTell:
		if err := bus.SendSequenced(&client.TalkTellClientPacket{
			Name:    strings.ToLower(outgoing.Target),
			Message: outgoing.Message,
		}); err != nil {
			slog.Error("send PM failed", "err", err)
			return
		}
		g.addChatMessage(game.ChatChannelMap, fmt.Sprintf("[PM] %s->%s: %s", g.client.Character.Name, outgoing.Target, outgoing.Message))
	case chatSendGroup:
		if err := bus.SendSequenced(&client.TalkOpenClientPacket{Message: outgoing.Message}); err != nil {
			slog.Error("send group chat failed", "err", err)
			return
		}
		g.addChatMessage(game.ChatChannelGroup, fmt.Sprintf("%s: %s", g.client.Character.Name, outgoing.Message))
	case chatSendGlobal:
		if err := bus.SendSequenced(&client.TalkMsgClientPacket{Message: outgoing.Message}); err != nil {
			slog.Error("send global chat failed", "err", err)
			return
		}
		g.addChatMessage(game.ChatChannelGlobal, fmt.Sprintf("%s: %s", g.client.Character.Name, outgoing.Message))
	default:
		if err := bus.SendSequenced(&client.TalkReportClientPacket{Message: outgoing.Message}); err != nil {
			slog.Error("send map chat failed", "err", err)
			return
		}
		g.addChatMessage(game.ChatChannelMap, fmt.Sprintf("%s: %s", g.client.Character.Name, outgoing.Message))
	}
}

func (g *Game) chatWrappedHistory(channel game.ChatChannel, maxWidth int) []string {
	if g.chat.History == nil {
		return nil
	}
	channelHistory := g.chat.History[channel]
	msgCount := len(channelHistory)

	// Return cached result if inputs haven't changed
	if g.chat.wrappedCache != nil &&
		g.chat.wrappedWidth[channel] == maxWidth &&
		g.chat.wrappedMsgCount[channel] == msgCount {
		return g.chat.wrappedCache[channel]
	}

	wrappedHistory := make([]string, 0, msgCount)
	for _, msg := range channelHistory {
		wrappedHistory = append(wrappedHistory, wrapChatLines(msg, maxWidth)...)
	}

	// Cache the result
	if g.chat.wrappedCache == nil {
		g.chat.wrappedCache = make(map[game.ChatChannel][]string, len(chatChannels))
		g.chat.wrappedWidth = make(map[game.ChatChannel]int, len(chatChannels))
		g.chat.wrappedMsgCount = make(map[game.ChatChannel]int, len(chatChannels))
	}
	g.chat.wrappedCache[channel] = wrappedHistory
	g.chat.wrappedWidth[channel] = maxWidth
	g.chat.wrappedMsgCount[channel] = msgCount
	return wrappedHistory
}

func (g *Game) maxChatScroll(inputRect image.Rectangle) float64 {
	maxHistoryLines := g.chatVisibleLineCount(inputRect)
	return float64(max(0, len(g.chatWrappedHistory(g.chat.ActiveChannel, inputRect.Dx()-4))-maxHistoryLines))
}

func (g *Game) maxChatScrollForActiveChannel() float64 {
	_, inputRect := g.chatRects()
	return g.maxChatScroll(inputRect)
}

func (g *Game) chatVisibleLineCount(inputRect image.Rectangle) int {
	historyTop := g.chatRectsTop()
	return max(0, (inputRect.Min.Y-historyTop)/chatLineHeight)
}

func (g *Game) chatRectsTop() int {
	panelRect, _ := g.chatRects()
	return panelRect.Min.Y + chatTabHeight + 20
}

func (g *Game) setChatScrollFromScrollbar(mouseY int, historyRect, inputRect image.Rectangle, dragOffsetY int, immediate bool) {
	maxScroll := g.maxChatScroll(inputRect)
	if maxScroll <= 0 {
		g.chat.ScrollTarget[g.chat.ActiveChannel] = 0
		g.chat.Scroll[g.chat.ActiveChannel] = 0
		return
	}
	trackRect := chatScrollbarRect(historyRect)
	thumbRect, ok := chatScrollbarThumbRect(historyRect, len(g.chatWrappedHistory(g.chat.ActiveChannel, historyRect.Dx()-14)), g.chatVisibleLineCount(inputRect), g.chat.ScrollTarget[g.chat.ActiveChannel])
	if !ok {
		return
	}
	thumbH := thumbRect.Dy()
	maxThumbTravel := max(1, trackRect.Dy()-thumbH)
	thumbTop := min(max(mouseY-dragOffsetY, trackRect.Min.Y), trackRect.Max.Y-thumbH)
	ratio := float64(thumbTop-trackRect.Min.Y) / float64(maxThumbTravel)
	scroll := clampChatScroll((1-ratio)*maxScroll, maxScroll)
	g.chat.ScrollTarget[g.chat.ActiveChannel] = scroll
	if immediate {
		g.chat.Scroll[g.chat.ActiveChannel] = scroll
	}
}

func clampChatScroll(value, maxValue float64) float64 {
	if value < 0 {
		return 0
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func normalizedChatWheelDelta(wheelY float64) (float64, bool, int) {
	if wheelY == 0 {
		return 0, false, 0
	}
	sign := 1
	if wheelY < 0 {
		sign = -1
	}
	if math.Abs(wheelY) <= chatWheelFineMax {
		return wheelY * chatWheelStep, false, sign
	}
	return math.Copysign(chatWheelStep, wheelY), true, sign
}

func parseOutgoingChat(msg string, activeChannel game.ChatChannel) (outgoingChat, bool) {
	trimmed := strings.TrimSpace(msg)
	if trimmed == "" {
		return outgoingChat{}, false
	}

	if strings.HasPrefix(trimmed, "!") {
		if target, message, ok := strings.Cut(trimmed[1:], " "); ok {
			target = strings.TrimSpace(target)
			message = strings.TrimSpace(message)
			if target != "" && message != "" {
				return outgoingChat{
					Mode:    chatSendTell,
					Target:  target,
					Message: message,
				}, true
			}
		}
	}

	if strings.HasPrefix(trimmed, "~") {
		body := strings.TrimSpace(trimmed[1:])
		return outgoingChat{Mode: chatSendGlobal, Message: body}, body != ""
	}

	if strings.HasPrefix(trimmed, "'") {
		body := strings.TrimSpace(trimmed[1:])
		return outgoingChat{Mode: chatSendGroup, Message: body}, body != ""
	}

	switch activeChannel {
	case game.ChatChannelGroup:
		return outgoingChat{Mode: chatSendGroup, Message: trimmed}, true
	case game.ChatChannelGlobal:
		return outgoingChat{Mode: chatSendGlobal, Message: trimmed}, true
	case game.ChatChannelSystem:
		return outgoingChat{}, false
	default:
		return outgoingChat{Mode: chatSendMap, Message: trimmed}, true
	}
}

// handleChatEvent processes incoming chat events from the server.
func (g *Game) handleChatEvent(evt game.Event) {
	msg, ok := evt.Data.(game.ChatMessage)
	if !ok {
		g.addChatMessage(game.ChatChannelSystem, evt.Message)
		return
	}
	g.addChatMessage(msg.Channel, msg.Text)
}
