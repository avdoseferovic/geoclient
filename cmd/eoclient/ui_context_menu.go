package main

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	clientui "github.com/avdo/eoweb/internal/ui"
	"github.com/avdo/eoweb/internal/ui/overlay"
)

var contextMenuItems = []string{"Trade", "Invite", "Join", "Whisper"}

func (g *Game) tryOpenPlayerContextMenu() bool {
	mx, my := ebiten.CursorPosition()
	snapshot := g.client.UISnapshot()

	// Check if cursor is over a nearby character
	tileX, tileY := g.hoveredTile(snapshot)
	for _, ch := range snapshot.NearbyChars {
		if ch.PlayerID == snapshot.PlayerID {
			continue
		}
		if ch.X == tileX && ch.Y == tileY {
			g.overlay.contextMenuOpen = true
			g.overlay.contextMenuX = mx
			g.overlay.contextMenuY = my
			g.overlay.contextMenuPlayerID = ch.PlayerID
			g.overlay.contextMenuName = ch.Name
			return true
		}
	}
	return false
}

func (g *Game) contextMenuRect() image.Rectangle {
	w := 100
	lineH := 20
	h := len(contextMenuItems)*lineH + 28 // 28 for header padding
	x := g.overlay.contextMenuX
	y := g.overlay.contextMenuY
	// Keep on screen
	if x+w > g.screenW {
		x = g.screenW - w
	}
	if y+h > g.screenH {
		y = g.screenH - h
	}
	return image.Rect(x, y, x+w, y+h)
}

func (g *Game) drawContextMenu(screen *ebiten.Image, theme clientui.Theme) {
	if !g.overlay.contextMenuOpen {
		return
	}

	rect := g.contextMenuRect()
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Accent: theme.Accent})

	mx, my := ebiten.CursorPosition()
	lineH := 20

	// Header: player name
	name := g.overlay.contextMenuName
	if name == "" {
		name = fmt.Sprintf("Player %d", g.overlay.contextMenuPlayerID)
	}
	clientui.DrawText(screen, name, rect.Min.X+8, rect.Min.Y+14, theme.Text)

	// Menu items
	for i, label := range contextMenuItems {
		iy := rect.Min.Y + 22 + i*lineH
		itemRect := image.Rect(rect.Min.X+4, iy, rect.Max.X-4, iy+lineH)
		if overlay.PointInRect(mx, my, itemRect) {
			clientui.FillRect(screen, float64(itemRect.Min.X), float64(itemRect.Min.Y), float64(itemRect.Dx()), float64(itemRect.Dy()), overlay.Colorize(theme.Accent, 60))
		}
		clientui.DrawText(screen, label, rect.Min.X+10, iy+14, theme.TextDim)
	}
}

func (g *Game) handleContextMenuClick(mx, my int) bool {
	rect := g.contextMenuRect()
	if !overlay.PointInRect(mx, my, rect) {
		g.overlay.contextMenuOpen = false
		return false
	}

	lineH := 20
	for i, label := range contextMenuItems {
		iy := rect.Min.Y + 22 + i*lineH
		itemRect := image.Rect(rect.Min.X+4, iy, rect.Max.X-4, iy+lineH)
		if !overlay.PointInRect(mx, my, itemRect) {
			continue
		}
		g.overlay.contextMenuOpen = false
		playerID := g.overlay.contextMenuPlayerID
		switch label {
		case "Trade":
			g.sendTradeRequest(playerID)
		case "Invite":
			g.sendPartyRequest(playerID, 1) // PartyRequest_Invite
		case "Join":
			g.sendPartyRequest(playerID, 0) // PartyRequest_Join
		case "Whisper":
			g.chat.Typing = true
			g.chat.Input = "!" + g.overlay.contextMenuName + " "
			g.chat.inputBuf = []rune(g.chat.Input)
		}
		return true
	}

	g.overlay.contextMenuOpen = false
	return true
}
