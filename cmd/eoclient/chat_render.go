package main

import (
	"image"
	"image/color"
	"math"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/avdo/eoweb/internal/game"
	clientui "github.com/avdo/eoweb/internal/ui"
	"github.com/avdo/eoweb/internal/ui/overlay"
)

func (g *Game) drawChat(screen *ebiten.Image, theme clientui.Theme) {
	panelRect, inputRect := g.chatRects()
	clientui.DrawPanel(screen, panelRect, theme, clientui.PanelOptions{Accent: theme.AccentMuted})
	for _, tab := range g.chatTabRects(panelRect) {
		clientui.DrawButton(screen, tab.Rect, theme, tab.Channel.Label(), g.chat.ActiveChannel == tab.Channel, false)
	}
	clientui.DrawInset(screen, inputRect, theme, g.chat.Typing)

	historyTop := panelRect.Min.Y + chatTabHeight + 20
	historyRect := g.chatHistoryRect(panelRect, inputRect)
	historyWidth := historyRect.Dx() - 14
	maxHistoryLines := max(0, (inputRect.Min.Y-historyTop)/chatLineHeight)
	wrappedHistory := g.chatWrappedHistory(g.chat.ActiveChannel, historyWidth)
	visibleHistory, scrollOffset, scrollFraction := visibleChatLines(wrappedHistory, maxHistoryLines, g.chat.Scroll[g.chat.ActiveChannel])
	g.chat.Scroll[g.chat.ActiveChannel] = scrollOffset
	hiddenOlder := max(0.0, float64(max(0, len(wrappedHistory)-maxHistoryLines))-scrollOffset)

	historyLayer := g.chatHistoryImage(historyRect.Dx(), historyRect.Dy())
	chatColor := chatChannelTextColor(g.chat.ActiveChannel, theme)
	y := inputRect.Min.Y - 12 + int(math.Round(scrollFraction*chatLineHeight))
	for i := len(visibleHistory) - 1; i >= 0 && y > historyTop; i-- {
		clientui.DrawText(historyLayer, visibleHistory[i], 2, y-historyRect.Min.Y, chatColor)
		y -= chatLineHeight
	}
	historyOp := &ebiten.DrawImageOptions{}
	historyOp.GeoM.Translate(float64(historyRect.Min.X), float64(historyRect.Min.Y))
	screen.DrawImage(historyLayer, historyOp)

	drawChatScrollbar(screen, historyRect, theme, len(wrappedHistory), maxHistoryLines, hiddenOlder, g.chat.Dragging && g.chat.DragChannel == g.chat.ActiveChannel)

	inputText := "Press Enter"
	inputColor := theme.TextDim
	if g.chat.Typing {
		inputText = "> " + g.chat.Input
		if g.overlay.ticks%40 < 20 {
			inputText += "_"
		}
		inputColor = theme.Text
	} else if len(wrappedHistory) > 0 {
		if g.chat.ActiveChannel == game.ChatChannelSystem {
			inputText = "System channel"
		} else {
			inputText = g.chat.ActiveChannel.Label()
		}
	}
	inputText = trimChatLineLeft(inputText, inputRect.Dx()-20)
	clientui.DrawText(screen, inputText, inputRect.Min.X+10, inputRect.Min.Y+16, inputColor)
}

func (g *Game) chatRects() (image.Rectangle, image.Rectangle) {
	panelRect := image.Rect(10, g.screenH-142, 372, g.screenH-10)
	return panelRect, image.Rect(panelRect.Min.X+12, panelRect.Max.Y-36, panelRect.Max.X-12, panelRect.Max.Y-12)
}

func (g *Game) chatHistoryRect(panelRect, inputRect image.Rectangle) image.Rectangle {
	return image.Rect(panelRect.Min.X+12, panelRect.Min.Y+chatTabHeight+20, panelRect.Max.X-14, inputRect.Min.Y-12)
}

type chatTabRect struct {
	Channel game.ChatChannel
	Rect    image.Rectangle
}

func (g *Game) chatTabRects(panelRect image.Rectangle) []chatTabRect {
	tabY := panelRect.Min.Y + 8
	tabW := max(56, (panelRect.Dx()-24)/len(chatChannels))
	rects := make([]chatTabRect, 0, len(chatChannels))
	for i, channel := range chatChannels {
		x := panelRect.Min.X + 12 + i*tabW
		right := x + tabW - 4
		if i == len(chatChannels)-1 {
			right = panelRect.Max.X - 12
		}
		rects = append(rects, chatTabRect{
			Channel: channel,
			Rect:    image.Rect(x, tabY, right, tabY+chatTabHeight),
		})
	}
	return rects
}

func visibleChatLines(lines []string, maxLines int, scrollOffset float64) ([]string, float64, float64) {
	if maxLines <= 0 || len(lines) == 0 {
		return nil, 0, 0
	}
	maxOffset := float64(max(0, len(lines)-maxLines))
	scrollOffset = clampChatScroll(scrollOffset, maxOffset)
	baseOffset := int(math.Floor(scrollOffset))
	fraction := scrollOffset - float64(baseOffset)
	extraOlderLine := 0
	if fraction > 0 {
		extraOlderLine = 1
	}
	end := len(lines) - baseOffset
	start := max(0, end-maxLines-extraOlderLine)
	return lines[start:end], scrollOffset, fraction
}

func drawChatScrollbar(screen *ebiten.Image, rect image.Rectangle, theme clientui.Theme, totalLines, visibleLines int, hiddenOlder float64, dragging bool) {
	trackRect := chatScrollbarRect(rect)
	if totalLines <= 0 || visibleLines <= 0 || totalLines <= visibleLines || rect.Dy() <= 0 {
		return
	}
	clientui.FillRect(screen, float64(trackRect.Min.X), float64(trackRect.Min.Y), float64(trackRect.Dx()), float64(trackRect.Dy()), overlay.Colorize(theme.BorderDark, 90))

	thumbRect, ok := chatScrollbarThumbRect(rect, totalLines, visibleLines, hiddenOlder)
	if !ok {
		return
	}
	fill := color.NRGBA{R: 125, G: 96, B: 58, A: 255}
	if dragging {
		fill = color.NRGBA{R: 154, G: 118, B: 68, A: 255}
	}
	clientui.FillRect(screen, float64(thumbRect.Min.X), float64(thumbRect.Min.Y), float64(thumbRect.Dx()), float64(thumbRect.Dy()), fill)
	clientui.DrawBorder(screen, thumbRect, theme.BorderDark, theme.BorderMid, theme.Accent)
}

func chatScrollbarRect(historyRect image.Rectangle) image.Rectangle {
	return image.Rect(historyRect.Max.X-8, historyRect.Min.Y, historyRect.Max.X-3, historyRect.Max.Y)
}

func chatScrollbarThumbRect(historyRect image.Rectangle, totalLines, visibleLines int, hiddenOlder float64) (image.Rectangle, bool) {
	trackRect := chatScrollbarRect(historyRect)
	if totalLines <= 0 || visibleLines <= 0 || totalLines <= visibleLines || trackRect.Dy() <= 0 {
		return image.Rectangle{}, false
	}
	thumbH := max(14, int(float64(trackRect.Dy())*float64(visibleLines)/float64(totalLines)))
	maxThumbTravel := max(0, trackRect.Dy()-thumbH)
	maxOffset := float64(max(1, totalLines-visibleLines))
	thumbY := trackRect.Min.Y + int(float64(maxThumbTravel)*hiddenOlder/maxOffset)
	return image.Rect(trackRect.Min.X, thumbY, trackRect.Max.X, thumbY+thumbH), true
}

func chatChannelTextColor(channel game.ChatChannel, theme clientui.Theme) color.Color {
	switch channel {
	case game.ChatChannelGroup:
		return color.NRGBA{R: 130, G: 210, B: 225, A: 255}
	case game.ChatChannelGlobal:
		return color.NRGBA{R: 230, G: 200, B: 110, A: 255}
	case game.ChatChannelSystem:
		return color.NRGBA{R: 210, G: 150, B: 110, A: 255}
	default:
		return theme.Text
	}
}

func wrapChatLines(text string, maxWidth int) []string {
	trimmed := clientui.WrapText(text, maxWidth)
	if len(trimmed) == 0 {
		return []string{""}
	}
	return trimmed
}

func (g *Game) chatHistoryImage(w, h int) *ebiten.Image {
	if g.chatHistoryBuf == nil || g.chatHistoryW != w || g.chatHistoryH != h {
		g.chatHistoryBuf = ebiten.NewImage(w, h)
		g.chatHistoryW = w
		g.chatHistoryH = h
	} else {
		g.chatHistoryBuf.Clear()
	}
	return g.chatHistoryBuf
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
