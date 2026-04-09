package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/avdoseferovic/geoclient/internal/game"
	"github.com/avdoseferovic/geoclient/internal/ui/overlay"
)

// updateDialogs handles input for modal dialogs (chest, trade, party invite).
// Returns true if a dialog consumed the input this frame.
func (g *Game) updateDialogs() bool {
	// Close context menu on any left click (handled separately)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && g.overlay.contextMenuOpen {
		mx, my := ebiten.CursorPosition()
		if !g.handleContextMenuClick(mx, my) {
			g.overlay.contextMenuOpen = false
		}
		return true // consume click either way when context menu is open
	}

	// Right-click opens context menu on players
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if g.tryOpenPlayerContextMenu() {
			return true
		}
	}

	// Escape closes any open dialog
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.overlay.contextMenuOpen {
			g.overlay.contextMenuOpen = false
			return true
		}
		if g.overlay.chestDialogOpen {
			g.closeChestDialog()
			return true
		}
		if g.overlay.tradeDialogOpen {
			g.sendTradeClose()
			g.overlay.tradeDialogOpen = false
			g.client.Lock()
			g.client.Trade = game.TradeState{}
			g.client.Unlock()
			return true
		}
		if g.overlay.shopDialogOpen {
			g.overlay.shopDialogOpen = false
			return true
		}
		if g.overlay.partyInviteOpen {
			g.overlay.partyInviteOpen = false
			g.client.Lock()
			g.client.PendingPartyInvite = nil
			g.client.Unlock()
			return true
		}
	}

	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return false
	}

	mx, my := ebiten.CursorPosition()

	// Modal dialogs consume ALL clicks (not just within their bounds)
	if g.overlay.partyInviteOpen {
		g.updatePartyInviteDialog(mx, my)
		return true
	}

	if g.overlay.tradeDialogOpen {
		if g.updateTradeDialog(mx, my) {
			return true
		}
	}
	if g.overlay.shopDialogOpen {
		g.updateShopDialog(mx, my)
		return true
	}

	// Semi-modal: chest consumes clicks within, passes through outside
	if g.overlay.chestDialogOpen {
		if g.updateChestDialog(mx, my) {
			return true
		}
	}

	return false
}

// --- Chest dialog input ---

func (g *Game) updateChestDialog(mx, my int) bool {
	rect := g.chestDialogRect()
	if !overlay.PointInRect(mx, my, rect) {
		return false
	}

	// Close button
	closeRect := image.Rect(rect.Max.X-60, rect.Max.Y-32, rect.Max.X-12, rect.Max.Y-12)
	if overlay.PointInRect(mx, my, closeRect) {
		g.closeChestDialog()
		return true
	}

	// Click item to take
	g.client.Lock()
	items := make([]game.ChestItem, len(g.client.ChestItems))
	copy(items, g.client.ChestItems)
	g.client.Unlock()

	for i, item := range items {
		iy := rect.Min.Y + 28 + i*16
		if iy+16 > rect.Max.Y-40 {
			break
		}
		itemRect := image.Rect(rect.Min.X+12, iy, rect.Max.X-12, iy+14)
		if overlay.PointInRect(mx, my, itemRect) {
			g.sendChestTake(g.overlay.chestX, g.overlay.chestY, item.ID)
			return true
		}
	}

	return true // consume click within dialog
}

func (g *Game) closeChestDialog() {
	g.overlay.chestDialogOpen = false
	g.client.Lock()
	g.client.ChestOpen = false
	g.client.Unlock()
}

// --- Trade dialog input ---

func (g *Game) updateTradeDialog(mx, my int) bool {
	g.client.Lock()
	trade := g.client.Trade
	playerItems := make([]game.TradeItem, len(trade.PlayerItems))
	copy(playerItems, trade.PlayerItems)
	g.client.Unlock()

	if trade.State == game.TradeStatePending {
		return g.updateTradePending(mx, my, trade)
	}
	if trade.State != game.TradeStateOpen {
		return false
	}

	rect := g.tradeDialogRect()
	if !overlay.PointInRect(mx, my, rect) {
		return false
	}

	// Agree button
	agreeRect := image.Rect(rect.Max.X-130, rect.Max.Y-32, rect.Max.X-72, rect.Max.Y-14)
	if overlay.PointInRect(mx, my, agreeRect) {
		g.sendTradeAgree(!trade.PlayerAgreed)
		return true
	}

	// Cancel button
	cancelRect := image.Rect(rect.Max.X-66, rect.Max.Y-32, rect.Max.X-12, rect.Max.Y-14)
	if overlay.PointInRect(mx, my, cancelRect) {
		g.sendTradeClose()
		g.overlay.tradeDialogOpen = false
		g.client.Lock()
		g.client.Trade = game.TradeState{}
		g.client.Unlock()
		return true
	}

	// Click player item to remove
	w := rect.Dx()
	colW := (w - 30) / 2
	leftCol := image.Rect(rect.Min.X+10, rect.Min.Y+28, rect.Min.X+10+colW, rect.Max.Y-40)
	for i, item := range playerItems {
		iy := leftCol.Min.Y + 20 + i*14
		if iy+14 > leftCol.Max.Y {
			break
		}
		itemRect := image.Rect(leftCol.Min.X+2, iy, leftCol.Max.X-2, iy+14)
		if overlay.PointInRect(mx, my, itemRect) {
			g.sendTradeRemove(item.ID)
			return true
		}
	}

	return true // consume click within dialog
}

func (g *Game) updateTradePending(mx, my int, trade game.TradeState) bool {
	rect := g.tradePendingRect()
	if !overlay.PointInRect(mx, my, rect) {
		return false
	}

	acceptRect := image.Rect(rect.Min.X+12, rect.Max.Y-30, rect.Min.X+72, rect.Max.Y-12)
	declineRect := image.Rect(rect.Min.X+80, rect.Max.Y-30, rect.Min.X+148, rect.Max.Y-12)

	if overlay.PointInRect(mx, my, acceptRect) {
		g.sendTradeAccept(trade.PartnerID)
		return true
	}
	if overlay.PointInRect(mx, my, declineRect) {
		g.sendTradeClose()
		g.overlay.tradeDialogOpen = false
		g.client.Lock()
		g.client.Trade = game.TradeState{}
		g.client.Unlock()
		return true
	}

	return true // consume click within dialog
}

func (g *Game) tradePendingRect() image.Rectangle {
	w, h := 240, 80
	x := (g.screenW - w) / 2
	y := (g.screenH - h) / 2
	return image.Rect(x, y, x+w, y+h)
}

// --- Party invite dialog input ---

func (g *Game) partyInviteRect() image.Rectangle {
	w, h := 260, 80
	x := (g.screenW - w) / 2
	y := (g.screenH - h) / 2
	return image.Rect(x, y, x+w, y+h)
}

func (g *Game) updatePartyInviteDialog(mx, my int) bool {
	rect := g.partyInviteRect()
	if !overlay.PointInRect(mx, my, rect) {
		return false
	}

	g.client.Lock()
	var pending *game.PendingPartyInvite
	if g.client.PendingPartyInvite != nil {
		p := *g.client.PendingPartyInvite
		pending = &p
	}
	g.client.Unlock()

	if pending == nil {
		g.overlay.partyInviteOpen = false
		return false
	}

	acceptRect := image.Rect(rect.Min.X+12, rect.Max.Y-30, rect.Min.X+72, rect.Max.Y-12)
	declineRect := image.Rect(rect.Min.X+80, rect.Max.Y-30, rect.Min.X+160, rect.Max.Y-12)

	if overlay.PointInRect(mx, my, acceptRect) {
		g.sendPartyAccept(pending.PlayerID, pending.RequestType)
		g.overlay.partyInviteOpen = false
		g.client.Lock()
		g.client.PendingPartyInvite = nil
		g.client.Unlock()
		return true
	}
	if overlay.PointInRect(mx, my, declineRect) {
		g.overlay.partyInviteOpen = false
		g.client.Lock()
		g.client.PendingPartyInvite = nil
		g.client.Unlock()
		return true
	}

	return true // consume click within dialog
}
