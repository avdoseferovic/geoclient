package main

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/avdoseferovic/geoclient/internal/game"
	clientui "github.com/avdoseferovic/geoclient/internal/ui"
	"github.com/avdoseferovic/geoclient/internal/ui/overlay"
)

func (g *Game) tradeDialogRect() image.Rectangle {
	w, h := 340, 260
	x := (g.screenW - w) / 2
	y := (g.screenH - h) / 2
	return image.Rect(x, y, x+w, y+h)
}

func (g *Game) drawTradeDialog(screen *ebiten.Image, theme clientui.Theme) {
	g.client.Lock()
	trade := g.client.Trade
	playerItems := make([]game.TradeItem, len(trade.PlayerItems))
	copy(playerItems, trade.PlayerItems)
	partnerItems := make([]game.TradeItem, len(trade.PartnerItems))
	copy(partnerItems, trade.PartnerItems)
	g.client.Unlock()

	if trade.State == game.TradeStatePending {
		g.drawTradePending(screen, theme, trade)
		return
	}
	if trade.State != game.TradeStateOpen {
		return
	}

	rect := g.tradeDialogRect()
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: fmt.Sprintf("Trade with %s", trade.PartnerName), Accent: theme.Accent})

	mx, my := ebiten.CursorPosition()

	// Two columns
	leftCol, rightCol := g.tradeColumnRects(rect)

	clientui.DrawInset(screen, leftCol, theme, false)
	clientui.DrawInset(screen, rightCol, theme, false)
	if g.inventoryDrag.Active {
		mx, my := ebiten.CursorPosition()
		if overlay.PointInRect(mx, my, leftCol) {
			clientui.FillRect(screen, float64(leftCol.Min.X+2), float64(leftCol.Min.Y+2), float64(leftCol.Dx()-4), float64(leftCol.Dy()-4), overlay.Colorize(theme.Accent, 28))
		}
	}

	clientui.DrawText(screen, "Your Offer", leftCol.Min.X+4, leftCol.Min.Y+12, theme.Text)
	clientui.DrawText(screen, "Their Offer", rightCol.Min.X+4, rightCol.Min.Y+12, theme.Text)

	// Player items (hover highlight — click handled in updateDialogs)
	for i, item := range playerItems {
		iy := leftCol.Min.Y + 20 + i*14
		if iy+14 > leftCol.Max.Y {
			break
		}
		itemRect := image.Rect(leftCol.Min.X+2, iy, leftCol.Max.X-2, iy+14)
		label := g.tradeItemLabel(item.ID, item.Amount)
		if overlay.PointInRect(mx, my, itemRect) {
			clientui.FillRect(screen, float64(itemRect.Min.X), float64(itemRect.Min.Y), float64(itemRect.Dx()), float64(itemRect.Dy()), overlay.Colorize(theme.Accent, 50))
		}
		clientui.DrawText(screen, label, leftCol.Min.X+4, iy+11, theme.TextDim)
	}

	// Partner items (read only)
	for i, item := range partnerItems {
		iy := rightCol.Min.Y + 20 + i*14
		if iy+14 > rightCol.Max.Y {
			break
		}
		label := g.tradeItemLabel(item.ID, item.Amount)
		clientui.DrawText(screen, label, rightCol.Min.X+4, iy+11, theme.TextDim)
	}

	// Status line
	hintRect := image.Rect(rect.Min.X+12, rect.Max.Y-58, rect.Max.X-136, rect.Max.Y-44)
	statusRect := image.Rect(rect.Min.X+12, rect.Max.Y-42, rect.Max.X-136, rect.Max.Y-28)
	clientui.DrawText(screen, "Drag items from bag to offer", hintRect.Min.X, hintRect.Min.Y+12, theme.TextDim)

	status := "Waiting..."
	if trade.PlayerAgreed && trade.PartnerAgreed {
		status = "Both agreed!"
	} else if trade.PlayerAgreed {
		status = "You agreed."
	} else if trade.PartnerAgreed {
		status = "Partner agreed."
	}
	clientui.DrawText(screen, status, statusRect.Min.X, statusRect.Min.Y+12, theme.TextDim)

	// Buttons (render only — input handled in updateDialogs)
	agreeRect := image.Rect(rect.Max.X-130, rect.Max.Y-32, rect.Max.X-72, rect.Max.Y-14)
	cancelRect := image.Rect(rect.Max.X-66, rect.Max.Y-32, rect.Max.X-12, rect.Max.Y-14)

	agreeLabel := "Agree"
	if trade.PlayerAgreed {
		agreeLabel = "Agreed"
	}
	clientui.DrawButton(screen, agreeRect, theme, agreeLabel, trade.PlayerAgreed, overlay.PointInRect(mx, my, agreeRect))
	clientui.DrawButton(screen, cancelRect, theme, "Cancel", false, overlay.PointInRect(mx, my, cancelRect))
}

func (g *Game) drawTradePending(screen *ebiten.Image, theme clientui.Theme, trade game.TradeState) {
	rect := g.tradePendingRect()
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Trade Request", Accent: theme.Accent})
	clientui.DrawText(screen, fmt.Sprintf("%s wants to trade", trade.PartnerName), rect.Min.X+12, rect.Min.Y+32, theme.Text)

	mx, my := ebiten.CursorPosition()
	acceptRect := image.Rect(rect.Min.X+12, rect.Max.Y-30, rect.Min.X+72, rect.Max.Y-12)
	declineRect := image.Rect(rect.Min.X+80, rect.Max.Y-30, rect.Min.X+148, rect.Max.Y-12)

	clientui.DrawButton(screen, acceptRect, theme, "Accept", false, overlay.PointInRect(mx, my, acceptRect))
	clientui.DrawButton(screen, declineRect, theme, "Decline", false, overlay.PointInRect(mx, my, declineRect))
}

func (g *Game) tradeColumnRects(rect image.Rectangle) (image.Rectangle, image.Rectangle) {
	colW := (rect.Dx() - 30) / 2
	leftCol := image.Rect(rect.Min.X+10, rect.Min.Y+28, rect.Min.X+10+colW, rect.Max.Y-40)
	rightCol := image.Rect(rect.Min.X+20+colW, rect.Min.Y+28, rect.Max.X-10, rect.Max.Y-40)
	return leftCol, rightCol
}

func (g *Game) tradeItemLabel(itemID, amount int) string {
	if g.itemDB != nil {
		if rec, ok := g.itemDB.Get(itemID); ok {
			return fmt.Sprintf("%s x%d", rec.Name, amount)
		}
	}
	return fmt.Sprintf("Item %d x%d", itemID, amount)
}
