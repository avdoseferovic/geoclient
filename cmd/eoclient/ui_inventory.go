package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/avdoseferovic/geoclient/internal/game"
	"github.com/avdoseferovic/geoclient/internal/ui/hud"
	"github.com/avdoseferovic/geoclient/internal/ui/overlay"
)

func (g *Game) handleInGameOverlayClick() bool {
	layout := overlay.InGameHUDLayout(g.screenW, g.screenH)
	mx, my := ebiten.CursorPosition()
	chatPanelRect, chatInputRect := g.chatRects()
	for _, tab := range g.chatTabRects(chatPanelRect) {
		if !overlay.PointInRect(mx, my, tab.Rect) {
			continue
		}
		g.setActiveChatChannel(tab.Channel)
		return true
	}
	if overlay.PointInRect(mx, my, chatInputRect) {
		g.chat.Typing = true
		return true
	}
	for _, button := range overlay.HUDMenuButtons(layout) {
		if !overlay.PointInRect(mx, my, button.Rect) {
			continue
		}
		if g.overlay.activeMenuPanel == button.Panel {
			g.overlay.activeMenuPanel = overlay.MenuPanelNone
		} else {
			g.overlay.activeMenuPanel = button.Panel
			if button.Panel == overlay.MenuPanelOnline {
				g.sendPlayersRequest()
			}
		}
		return true
	}
	if g.overlay.activeMenuPanel != overlay.MenuPanelNone {
		if overlay.PointInRect(mx, my, layout.MenuPanelRect) && g.handleHUDPanelClick(layout.MenuPanelRect, mx, my) {
			return true
		}
		return overlay.PointInRect(mx, my, chatPanelRect) ||
			overlay.PointInRect(mx, my, layout.StatusRect) ||
			overlay.PointInRect(mx, my, layout.MenuRect) ||
			overlay.PointInRect(mx, my, layout.InfoRect)
	}
	return overlay.PointInRect(mx, my, chatPanelRect) ||
		overlay.PointInRect(mx, my, layout.StatusRect) ||
		overlay.PointInRect(mx, my, layout.MenuRect) ||
		overlay.PointInRect(mx, my, layout.MenuPanelRect) ||
		overlay.PointInRect(mx, my, layout.InfoRect)
}

func (g *Game) handleHUDPanelClick(rect image.Rectangle, mx, my int) bool {
	// Party panel: handle Leave button
	if g.overlay.activeMenuPanel == overlay.MenuPanelParty {
		leaveRect := image.Rect(rect.Min.X+12, rect.Max.Y-32, rect.Min.X+72, rect.Max.Y-14)
		if overlay.PointInRect(mx, my, leaveRect) {
			g.sendPartyRemove(g.client.PlayerID)
		}
		return true
	}
	if g.overlay.activeMenuPanel != overlay.MenuPanelInventory {
		return true
	}
	items := g.client.UISnapshot().Inventory
	if len(items) == 0 {
		g.overlay.selectedInventory = 0
		return true
	}
	for i, tabRect := range hud.PageTabRects(rect) {
		if overlay.PointInRect(mx, my, tabRect) {
			g.overlay.inventoryPage = i
			return true
		}
	}
	gridRect := hud.GridRect(rect)
	if !overlay.PointInRect(mx, my, gridRect) {
		return true
	}
	positions := g.inventoryGridPositions(items)
	cellX := (mx - gridRect.Min.X) / hud.InventoryCellSize
	cellY := (my - gridRect.Min.Y) / hud.InventoryCellSize
	for index, item := range items {
		pos, ok := positions[item.ID]
		if !ok || pos.Page != g.overlay.inventoryPage {
			continue
		}
		if cellX >= pos.X && cellX < pos.X+pos.W && cellY >= pos.Y && cellY < pos.Y+pos.H {
			if item.ID == g.overlay.lastClickItem && g.overlay.ticks-g.overlay.lastClickTick < 12 {
				g.tryEquipItem(item.ID)
				g.overlay.lastClickItem = 0
				return true
			}
			g.overlay.lastClickItem = item.ID
			g.overlay.lastClickTick = g.overlay.ticks
			g.overlay.selectedInventory = index
			itemRect := image.Rect(
				gridRect.Min.X+pos.X*hud.InventoryCellSize+1, gridRect.Min.Y+pos.Y*hud.InventoryCellSize+1,
				gridRect.Min.X+(pos.X+pos.W)*hud.InventoryCellSize, gridRect.Min.Y+(pos.Y+pos.H)*hud.InventoryCellSize,
			)
			g.inventoryDrag = inventoryDragState{Active: true, ItemID: item.ID, Amount: item.Amount, Offset: image.Pt(mx-itemRect.Min.X, my-itemRect.Min.Y)}
			return true
		}
	}
	return true
}

func (g *Game) handleInGameRightClick() {
	if g.overlay.activeMenuPanel != overlay.MenuPanelGuild {
		return
	}
	layout := overlay.InGameHUDLayout(g.screenW, g.screenH)
	mx, my := ebiten.CursorPosition()
	if !overlay.PointInRect(mx, my, layout.MenuPanelRect) {
		return
	}
	snapshot := g.client.UISnapshot()
	slots := paperdollSlotsForRect(layout.MenuPanelRect, snapshot)
	for _, slot := range slots {
		if slot.ItemID > 0 && overlay.PointInRect(mx, my, slot.Rect) {
			g.sendUnequipItem(slot.ItemID, slot.SubLoc)
			return
		}
	}
}

func paperdollSlotsForRect(rect image.Rectangle, snapshot game.UISnapshot) []hud.PaperdollSlot {
	bodyRect := image.Rect(rect.Min.X+10, rect.Min.Y+42, rect.Max.X-10, rect.Max.Y-14)
	cx := bodyRect.Min.X + bodyRect.Dx()/2
	const slotW, slotH = 42, 42
	const smallW, smallH = 32, 32
	hatX := cx - slotW/2
	hatY := bodyRect.Min.Y + 6
	neckX := hatX + slotW + 6
	neckY := hatY + 6
	armorX := cx - slotW/2
	armorY := hatY + slotH + 8
	weaponX := armorX - slotW - 10
	shieldX := armorX + slotW + 10
	beltY := armorY + slotH + 8
	glovesX := weaponX
	accX := shieldX
	bootsY := beltY + slotH + 8
	return hud.PaperdollSlots(snapshot.Equipment, hatX, hatY, neckX, neckY, weaponX, armorX, armorY, shieldX, glovesX, beltY, accX, bootsY, cx, slotW, slotH, smallW, smallH)
}

func (g *Game) tryEquipItem(itemID int) {
	if g.itemDB == nil || itemID <= 0 {
		return
	}
	item, ok := g.itemDB.Get(itemID)
	if !ok || !isEquippableType(item.Type) {
		return
	}
	g.sendEquipItem(itemID)
}

func (g *Game) inventoryGridPositions(items []game.InventoryItem) map[int]hud.InventoryGridPos {
	positions := make(map[int]hud.InventoryGridPos, len(items))
	changed := false
	occupied := make([][][]bool, hud.InventoryGridPages)
	for page := range hud.InventoryGridPages {
		occupied[page] = make([][]bool, hud.InventoryGridRows)
		for row := range hud.InventoryGridRows {
			occupied[page][row] = make([]bool, hud.InventoryGridCols)
		}
	}
	validItems := make(map[int]bool, len(items))
	for _, item := range items {
		validItems[item.ID] = true
	}
	for itemID := range g.inventoryLayout {
		if !validItems[itemID] {
			delete(g.inventoryLayout, itemID)
			changed = true
		}
	}
	for _, item := range items {
		stored, ok := g.inventoryLayout[item.ID]
		if !ok {
			continue
		}
		w, h := g.inventoryItemSize(item.ID)
		if stored.Page < 0 || stored.Page >= hud.InventoryGridPages || stored.X < 0 || stored.Y < 0 || stored.X+w > hud.InventoryGridCols || stored.Y+h > hud.InventoryGridRows {
			delete(g.inventoryLayout, item.ID)
			changed = true
			continue
		}
		if !hud.InventoryFits(occupied[stored.Page], stored.X, stored.Y, w, h) {
			delete(g.inventoryLayout, item.ID)
			changed = true
			continue
		}
		hud.InventoryOccupy(occupied[stored.Page], stored.X, stored.Y, w, h)
		positions[item.ID] = hud.InventoryGridPos{Page: stored.Page, X: stored.X, Y: stored.Y, W: w, H: h}
	}
	for _, item := range items {
		if _, ok := positions[item.ID]; ok {
			continue
		}
		w, h := g.inventoryItemSize(item.ID)
		for page := range hud.InventoryGridPages {
			placed := false
			for y := 0; y <= hud.InventoryGridRows-h; y++ {
				for x := 0; x <= hud.InventoryGridCols-w; x++ {
					if !hud.InventoryFits(occupied[page], x, y, w, h) {
						continue
					}
					hud.InventoryOccupy(occupied[page], x, y, w, h)
					positions[item.ID] = hud.InventoryGridPos{Page: page, X: x, Y: y, W: w, H: h}
					g.inventoryLayout[item.ID] = storedInventoryPos{Page: page, X: x, Y: y}
					changed = true
					placed = true
					break
				}
				if placed {
					break
				}
			}
			if placed {
				break
			}
		}
	}
	if changed {
		g.saveInventoryLayout()
	}
	return positions
}

func (g *Game) moveInventoryItem(itemID, page, x, y int, items []game.InventoryItem) bool {
	w, h := g.inventoryItemSize(itemID)
	if page < 0 || page >= hud.InventoryGridPages || x < 0 || y < 0 || x+w > hud.InventoryGridCols || y+h > hud.InventoryGridRows {
		return false
	}
	layout := make(map[int]storedInventoryPos, len(g.inventoryLayout))
	for id, pos := range g.inventoryLayout {
		if id == itemID {
			continue
		}
		layout[id] = pos
	}
	layout[itemID] = storedInventoryPos{Page: page, X: x, Y: y}
	old := g.inventoryLayout
	g.inventoryLayout = layout
	positions := g.inventoryGridPositions(items)
	pos, ok := positions[itemID]
	if !ok || pos.Page != page || pos.X != x || pos.Y != y {
		g.inventoryLayout = old
		return false
	}
	g.saveInventoryLayout()
	return true
}

func (g *Game) finishInventoryDrag() {
	if !g.inventoryDrag.Active {
		return
	}
	defer func() { g.inventoryDrag = inventoryDragState{} }()

	mx, my := ebiten.CursorPosition()

	// Drop onto chest dialog → deposit item
	if g.overlay.chestDialogOpen {
		chestRect := g.chestDialogRect()
		if overlay.PointInRect(mx, my, chestRect) {
			item := g.findInventoryItem(g.inventoryDrag.ItemID)
			if item != nil && item.Amount > 0 {
				g.sendChestAdd(g.overlay.chestX, g.overlay.chestY, item.ID, item.Amount)
			}
			return
		}
	}

	// Drop onto trade dialog → offer item
	if g.overlay.tradeDialogOpen {
		tradeRect := g.tradeDialogRect()
		leftCol, _ := g.tradeColumnRects(tradeRect)
		if overlay.PointInRect(mx, my, leftCol) {
			item := g.findInventoryItem(g.inventoryDrag.ItemID)
			if item != nil && item.Amount > 0 {
				if item.Amount > 1 {
					g.openTradeAmountPicker(item.ID, item.Amount)
				} else {
					g.sendTradeAdd(item.ID, 1)
					g.overlay.statusMessage = "Added item to trade"
				}
			}
			return
		}
	}

	layout := overlay.InGameHUDLayout(g.screenW, g.screenH)
	panelRect := layout.MenuPanelRect
	if g.overlay.activeMenuPanel != overlay.MenuPanelInventory {
		return
	}
	for i, tabRect := range hud.PageTabRects(panelRect) {
		if overlay.PointInRect(mx, my, tabRect) {
			g.overlay.inventoryPage = i
			return
		}
	}
	gridRect := hud.GridRect(panelRect)
	if !overlay.PointInRect(mx, my, gridRect) {
		g.tryDropDraggedInventoryItem()
		return
	}
	x := (mx - g.inventoryDrag.Offset.X - gridRect.Min.X) / hud.InventoryCellSize
	y := (my - g.inventoryDrag.Offset.Y - gridRect.Min.Y) / hud.InventoryCellSize
	items := g.client.UISnapshot().Inventory
	if g.moveInventoryItem(g.inventoryDrag.ItemID, g.overlay.inventoryPage, x, y, items) {
		g.overlay.statusMessage = "Inventory layout saved"
	}
}

func (g *Game) tryDropDraggedInventoryItem() bool {
	mx, my := ebiten.CursorPosition()
	if g.worldHoverBlockedByHUD(mx, my) {
		return false
	}
	hover := g.currentWorldHoverIntent()
	if !hover.Valid || hover.CursorType < 0 {
		return false
	}
	if !withinItemInteractionRange(g.client.Character.X, g.client.Character.Y, hover.TileX, hover.TileY) {
		g.overlay.statusMessage = "Drop target out of range"
		return false
	}
	item := g.findInventoryItem(g.inventoryDrag.ItemID)
	if item == nil || item.Amount <= 0 {
		return false
	}
	g.clearAutoWalk()
	if item.Amount > 1 {
		g.openItemAmountPicker(item.ID, item.Amount, hover.TileX, hover.TileY)
		return true
	}
	g.sendDropItem(item.ID, 1, hover.TileX, hover.TileY)
	g.overlay.statusMessage = "Dropped item"
	return true
}

func (g *Game) findInventoryItem(itemID int) *game.InventoryItem {
	for i := range g.client.Inventory {
		if g.client.Inventory[i].ID == itemID {
			return &g.client.Inventory[i]
		}
	}
	return nil
}

func (g *Game) drawInventoryDrag(screen *ebiten.Image) {
	mx, my := ebiten.CursorPosition()
	rect := image.Rect(mx-g.inventoryDrag.Offset.X, my-g.inventoryDrag.Offset.Y, mx-g.inventoryDrag.Offset.X+48, my-g.inventoryDrag.Offset.Y+48)
	g.drawItemIcon(screen, rect, g.inventoryDrag.ItemID, g.inventoryDrag.Amount)
}

func (g *Game) inventoryItemSize(itemID int) (int, int) {
	if g.itemDB == nil {
		return 1, 1
	}
	item, ok := g.itemDB.Get(itemID)
	if !ok {
		return 1, 1
	}
	switch item.Size {
	case 1:
		return 1, 2
	case 2:
		return 1, 3
	case 3:
		return 1, 4
	case 4:
		return 2, 1
	case 5:
		return 2, 2
	case 6:
		return 2, 3
	case 7:
		return 2, 4
	default:
		return 1, 1
	}
}

func (g *Game) saveInventoryLayout() {
	if err := saveInventoryLayout(g.inventoryLayoutPath, g.inventoryLayout); err != nil {
		g.overlay.statusMessage = "Inventory layout save failed"
	}
}
