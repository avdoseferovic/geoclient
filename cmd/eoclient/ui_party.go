package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/avdoseferovic/geoclient/internal/game"
	clientui "github.com/avdoseferovic/geoclient/internal/ui"
	"github.com/avdoseferovic/geoclient/internal/ui/overlay"
)

func (g *Game) drawPartyPanel(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle) {
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Party", Accent: theme.AccentMuted})

	g.client.Lock()
	members := make([]game.PartyMember, len(g.client.PartyMembers))
	copy(members, g.client.PartyMembers)
	g.client.Unlock()

	y := rect.Min.Y + 26

	if len(members) == 0 {
		clientui.DrawTextCentered(screen, "Not in a party.", image.Rect(rect.Min.X+10, y, rect.Max.X-10, y+30), theme.TextDim)
		return
	}

	for _, m := range members {
		if y+16 > rect.Max.Y-36 {
			break
		}
		label := m.Name
		if m.Leader {
			label = "* " + label
		}
		clientui.DrawText(screen, fmt.Sprintf("%s Lv%d", label, m.Level), rect.Min.X+12, y+11, theme.Text)

		// HP bar
		barX := rect.Max.X - 70
		barW := 56
		barH := 8
		barY := y + 4
		clientui.FillRect(screen, float64(barX), float64(barY), float64(barW), float64(barH), overlay.Colorize(theme.BorderDark, 120))
		hpW := float64(barW) * float64(m.HpPercentage) / 100.0
		hpColor := color.NRGBA{R: 110, G: 200, B: 110, A: 255}
		if m.HpPercentage <= 25 {
			hpColor = color.NRGBA{R: 200, G: 80, B: 80, A: 255}
		} else if m.HpPercentage <= 50 {
			hpColor = color.NRGBA{R: 200, G: 200, B: 80, A: 255}
		}
		clientui.FillRect(screen, float64(barX), float64(barY), hpW, float64(barH), hpColor)
		y += 16
	}

	// Leave button (render only — click would need to go through updateDialogs or panel click handler)
	mx, my := ebiten.CursorPosition()
	leaveRect := image.Rect(rect.Min.X+12, rect.Max.Y-32, rect.Min.X+72, rect.Max.Y-14)
	leaveHover := overlay.PointInRect(mx, my, leaveRect)
	clientui.DrawButton(screen, leaveRect, theme, "Leave", false, leaveHover)
}

func (g *Game) drawPartyInviteDialog(screen *ebiten.Image, theme clientui.Theme) {
	g.client.Lock()
	var pending *game.PendingPartyInvite
	if g.client.PendingPartyInvite != nil {
		p := *g.client.PendingPartyInvite
		pending = &p
	}
	g.client.Unlock()

	if pending == nil {
		return
	}

	rect := g.partyInviteRect()
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Party Invite", Accent: theme.Accent})
	clientui.DrawText(screen, fmt.Sprintf("%s wants to party", pending.PlayerName), rect.Min.X+12, rect.Min.Y+32, theme.Text)

	mx, my := ebiten.CursorPosition()
	acceptRect := image.Rect(rect.Min.X+12, rect.Max.Y-30, rect.Min.X+72, rect.Max.Y-12)
	declineRect := image.Rect(rect.Min.X+80, rect.Max.Y-30, rect.Min.X+160, rect.Max.Y-12)

	clientui.DrawButton(screen, acceptRect, theme, "Accept", false, overlay.PointInRect(mx, my, acceptRect))
	clientui.DrawButton(screen, declineRect, theme, "Decline", false, overlay.PointInRect(mx, my, declineRect))
}
