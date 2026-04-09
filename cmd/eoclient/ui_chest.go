package main

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	clientui "github.com/avdoseferovic/geoclient/internal/ui"
	"github.com/avdoseferovic/geoclient/internal/ui/overlay"
)

func (g *Game) chestDialogRect() image.Rectangle {
	w, h := 260, 200
	x := (g.screenW - w) / 2
	y := (g.screenH - h) / 2
	return image.Rect(x, y, x+w, y+h)
}

func (g *Game) drawChestDialog(screen *ebiten.Image, theme clientui.Theme) {
	rect := g.chestDialogRect()
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Chest", Accent: theme.Accent})

	g.client.Lock()
	items := make([]struct{ id, amount int }, len(g.client.ChestItems))
	for i, ci := range g.client.ChestItems {
		items[i] = struct{ id, amount int }{ci.ID, ci.Amount}
	}
	g.client.Unlock()

	mx, my := ebiten.CursorPosition()

	if len(items) == 0 {
		clientui.DrawTextCentered(screen, "Chest is empty.", image.Rect(rect.Min.X+10, rect.Min.Y+30, rect.Max.X-10, rect.Max.Y-40), theme.TextDim)
	} else {
		for i, item := range items {
			iy := rect.Min.Y + 28 + i*16
			if iy+16 > rect.Max.Y-40 {
				break
			}
			itemRect := image.Rect(rect.Min.X+12, iy, rect.Max.X-12, iy+14)
			label := fmt.Sprintf("Item %d x%d", item.id, item.amount)
			if g.itemDB != nil {
				if rec, ok := g.itemDB.Get(item.id); ok {
					label = fmt.Sprintf("%s x%d", rec.Name, item.amount)
				}
			}
			if overlay.PointInRect(mx, my, itemRect) {
				clientui.FillRect(screen, float64(itemRect.Min.X), float64(itemRect.Min.Y), float64(itemRect.Dx()), float64(itemRect.Dy()), overlay.Colorize(theme.Accent, 50))
			}
			clientui.DrawText(screen, label, rect.Min.X+14, iy+11, theme.Text)
		}
	}

	clientui.DrawTextCentered(screen, "Drag items from bag to deposit", image.Rect(rect.Min.X+10, rect.Max.Y-52, rect.Max.X-66, rect.Max.Y-38), theme.TextDim)

	// Close button (render only — input handled in updateDialogs)
	closeRect := image.Rect(rect.Max.X-60, rect.Max.Y-32, rect.Max.X-12, rect.Max.Y-12)
	closeHover := overlay.PointInRect(mx, my, closeRect)
	clientui.DrawButton(screen, closeRect, theme, "Close", false, closeHover)
}
