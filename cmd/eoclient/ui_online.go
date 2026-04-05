package main

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/avdo/eoweb/internal/game"
	clientui "github.com/avdo/eoweb/internal/ui"
)

func (g *Game) drawOnlinePanel(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle) {
	g.client.Lock()
	players := make([]game.OnlinePlayer, len(g.client.OnlinePlayers))
	copy(players, g.client.OnlinePlayers)
	g.client.Unlock()

	title := fmt.Sprintf("Online (%d)", len(players))
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: title, Accent: theme.AccentMuted})

	y := rect.Min.Y + 26

	if len(players) == 0 {
		clientui.DrawTextCentered(screen, "No players online.", image.Rect(rect.Min.X+10, y, rect.Max.X-10, y+30), theme.TextDim)
		return
	}

	for _, p := range players {
		if y+16 > rect.Max.Y-10 {
			break
		}
		label := p.Name
		if p.GuildTag != "" && p.GuildTag != "   " {
			label = fmt.Sprintf("[%s] %s", p.GuildTag, p.Name)
		}
		clientui.DrawText(screen, label, rect.Min.X+12, y+11, theme.Text)

		lvl := fmt.Sprintf("Lv%d", p.Level)
		clientui.DrawText(screen, lvl, rect.Max.X-48, y+11, theme.TextDim)

		y += 16
	}
}
