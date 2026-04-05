package main

import (
	"fmt"
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/avdo/eoweb/internal/game"
	clientui "github.com/avdo/eoweb/internal/ui"
	"github.com/avdo/eoweb/internal/ui/overlay"
)

const (
	shopTabBuy = iota
	shopTabSell
	shopTabCraft
)

type shopListEntry struct {
	ItemID  int
	Title   string
	Detail  string
	Enabled bool
	Action  int
}

func (g *Game) shopDialogRect() image.Rectangle {
	w, h := 420, 300
	x := (g.screenW - w) / 2
	y := (g.screenH - h) / 2
	return image.Rect(x, y, x+w, y+h)
}

func (g *Game) drawShopDialog(screen *ebiten.Image, theme clientui.Theme) {
	snapshot := g.client.UISnapshot()
	rect := g.shopDialogRect()
	title := snapshot.Shop.Name
	if title == "" {
		title = "Shop"
	}
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: title, Accent: theme.Accent})

	g.normalizeShopTab(snapshot)
	mx, my := ebiten.CursorPosition()

	for _, tab := range g.shopTabButtons(rect, snapshot) {
		clientui.DrawButton(screen, tab.Rect, theme, tab.Label, g.overlay.shopTab == tab.ID, overlay.PointInRect(mx, my, tab.Rect))
	}

	listRect := image.Rect(rect.Min.X+12, rect.Min.Y+64, rect.Max.X-12, rect.Max.Y-44)
	clientui.DrawInset(screen, listRect, theme, false)
	entries := g.shopEntries(snapshot)
	if len(entries) == 0 {
		clientui.DrawTextCentered(screen, "Nothing available in this section.", listRect, theme.TextDim)
	} else {
		for i, entry := range entries {
			rowY := listRect.Min.Y + 10 + i*22
			if rowY+20 > listRect.Max.Y {
				break
			}
			rowRect := image.Rect(listRect.Min.X+4, rowY, listRect.Max.X-4, rowY+18)
			if overlay.PointInRect(mx, my, rowRect) {
				clientui.FillRect(screen, float64(rowRect.Min.X), float64(rowRect.Min.Y), float64(rowRect.Dx()), float64(rowRect.Dy()), overlay.Colorize(theme.Accent, 50))
			}
			textColor := theme.Text
			if !entry.Enabled {
				textColor = theme.TextDim
			}
			clientui.DrawText(screen, entry.Title, rowRect.Min.X+6, rowRect.Min.Y+12, textColor)
			clientui.DrawText(screen, entry.Detail, rowRect.Min.X+160, rowRect.Min.Y+12, theme.TextDim)
		}
	}

	clientui.DrawText(screen, "Click an item to buy or sell one. Crafting consumes listed ingredients.", rect.Min.X+14, rect.Max.Y-18, theme.TextDim)

	closeRect := image.Rect(rect.Max.X-64, rect.Max.Y-32, rect.Max.X-14, rect.Max.Y-12)
	clientui.DrawButton(screen, closeRect, theme, "Close", false, overlay.PointInRect(mx, my, closeRect))
}

func (g *Game) updateShopDialog(mx, my int) bool {
	snapshot := g.client.UISnapshot()
	rect := g.shopDialogRect()

	closeRect := image.Rect(rect.Max.X-64, rect.Max.Y-32, rect.Max.X-14, rect.Max.Y-12)
	if overlay.PointInRect(mx, my, closeRect) {
		g.overlay.shopDialogOpen = false
		return true
	}

	for _, tab := range g.shopTabButtons(rect, snapshot) {
		if overlay.PointInRect(mx, my, tab.Rect) {
			g.overlay.shopTab = tab.ID
			return true
		}
	}

	listRect := image.Rect(rect.Min.X+12, rect.Min.Y+64, rect.Max.X-12, rect.Max.Y-44)
	if !overlay.PointInRect(mx, my, listRect) {
		return true
	}

	entries := g.shopEntries(snapshot)
	for i, entry := range entries {
		rowY := listRect.Min.Y + 10 + i*22
		if rowY+20 > listRect.Max.Y {
			break
		}
		rowRect := image.Rect(listRect.Min.X+4, rowY, listRect.Max.X-4, rowY+18)
		if !overlay.PointInRect(mx, my, rowRect) {
			continue
		}
		if !entry.Enabled {
			return true
		}
		switch entry.Action {
		case shopTabBuy:
			if gold := g.inventoryAmount(snapshot.Inventory, 1); gold < g.shopBuyPrice(snapshot, entry.ItemID) {
				g.overlay.statusMessage = "Not enough gold"
				return true
			}
			g.sendShopBuy(entry.ItemID, 1)
			g.overlay.statusMessage = fmt.Sprintf("Bought %s", entry.Title)
		case shopTabSell:
			g.sendShopSell(entry.ItemID, 1)
			g.overlay.statusMessage = fmt.Sprintf("Sold %s", entry.Title)
		case shopTabCraft:
			if !g.canCraft(snapshot, entry.ItemID) {
				g.overlay.statusMessage = "Missing crafting ingredients"
				return true
			}
			g.sendShopCraft(entry.ItemID)
			g.overlay.statusMessage = fmt.Sprintf("Crafting %s", entry.Title)
		}
		return true
	}

	return true
}

func (g *Game) shopTabButtons(rect image.Rectangle, snapshot game.UISnapshot) []struct {
	ID    int
	Label string
	Rect  image.Rectangle
} {
	buttons := make([]struct {
		ID    int
		Label string
		Rect  image.Rectangle
	}, 0, 3)
	x := rect.Min.X + 12
	for _, tab := range []struct {
		ID        int
		Label     string
		Available bool
	}{
		{ID: shopTabBuy, Label: "Buy", Available: len(g.shopBuyItems(snapshot)) > 0},
		{ID: shopTabSell, Label: "Sell", Available: len(g.shopSellItems(snapshot)) > 0},
		{ID: shopTabCraft, Label: "Craft", Available: len(snapshot.Shop.CraftItems) > 0},
	} {
		if !tab.Available {
			continue
		}
		buttons = append(buttons, struct {
			ID    int
			Label string
			Rect  image.Rectangle
		}{
			ID:    tab.ID,
			Label: tab.Label,
			Rect:  image.Rect(x, rect.Min.Y+32, x+74, rect.Min.Y+54),
		})
		x += 82
	}
	return buttons
}

func (g *Game) normalizeShopTab(snapshot game.UISnapshot) {
	if len(g.shopEntries(snapshot)) > 0 {
		return
	}
	for _, tab := range []int{shopTabBuy, shopTabSell, shopTabCraft} {
		g.overlay.shopTab = tab
		if len(g.shopEntries(snapshot)) > 0 {
			return
		}
	}
}

func (g *Game) shopEntries(snapshot game.UISnapshot) []shopListEntry {
	switch g.overlay.shopTab {
	case shopTabSell:
		sells := g.shopSellItems(snapshot)
		entries := make([]shopListEntry, 0, len(sells))
		for _, item := range sells {
			entries = append(entries, shopListEntry{
				ItemID:  item.ItemID,
				Title:   g.itemLabel(item.ItemID),
				Detail:  fmt.Sprintf("Price %d | Bag %d", item.SellPrice, g.inventoryAmount(snapshot.Inventory, item.ItemID)),
				Enabled: g.inventoryAmount(snapshot.Inventory, item.ItemID) > 0,
				Action:  shopTabSell,
			})
		}
		return entries
	case shopTabCraft:
		entries := make([]shopListEntry, 0, len(snapshot.Shop.CraftItems))
		for _, item := range snapshot.Shop.CraftItems {
			entries = append(entries, shopListEntry{
				ItemID:  item.ItemID,
				Title:   g.itemLabel(item.ItemID),
				Detail:  g.craftIngredientLabel(item),
				Enabled: g.canCraft(snapshot, item.ItemID),
				Action:  shopTabCraft,
			})
		}
		return entries
	default:
		buys := g.shopBuyItems(snapshot)
		entries := make([]shopListEntry, 0, len(buys))
		for _, item := range buys {
			entries = append(entries, shopListEntry{
				ItemID:  item.ItemID,
				Title:   g.itemLabel(item.ItemID),
				Detail:  fmt.Sprintf("Price %d", item.BuyPrice),
				Enabled: item.MaxBuyAmount > 0,
				Action:  shopTabBuy,
			})
		}
		return entries
	}
}

func (g *Game) shopBuyItems(snapshot game.UISnapshot) []game.ShopTradeItem {
	result := make([]game.ShopTradeItem, 0, len(snapshot.Shop.TradeItems))
	for _, item := range snapshot.Shop.TradeItems {
		if item.BuyPrice > 0 {
			result = append(result, item)
		}
	}
	return result
}

func (g *Game) shopSellItems(snapshot game.UISnapshot) []game.ShopTradeItem {
	result := make([]game.ShopTradeItem, 0, len(snapshot.Shop.TradeItems))
	for _, item := range snapshot.Shop.TradeItems {
		if item.SellPrice > 0 && g.inventoryAmount(snapshot.Inventory, item.ItemID) > 0 {
			result = append(result, item)
		}
	}
	return result
}

func (g *Game) shopBuyPrice(snapshot game.UISnapshot, itemID int) int {
	for _, item := range snapshot.Shop.TradeItems {
		if item.ItemID == itemID {
			return item.BuyPrice
		}
	}
	return 0
}

func (g *Game) canCraft(snapshot game.UISnapshot, itemID int) bool {
	for _, craft := range snapshot.Shop.CraftItems {
		if craft.ItemID != itemID {
			continue
		}
		for _, ingredient := range craft.Ingredients {
			if ingredient.Id <= 0 || ingredient.Amount <= 0 {
				continue
			}
			if g.inventoryAmount(snapshot.Inventory, ingredient.Id) < ingredient.Amount {
				return false
			}
		}
		return true
	}
	return false
}

func (g *Game) craftIngredientLabel(item game.ShopCraftItem) string {
	parts := make([]string, 0, len(item.Ingredients))
	for _, ingredient := range item.Ingredients {
		if ingredient.Id <= 0 || ingredient.Amount <= 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%dx %s", ingredient.Amount, g.itemLabel(ingredient.Id)))
	}
	if len(parts) == 0 {
		return "No ingredients"
	}
	return strings.Join(parts, ", ")
}

func (g *Game) inventoryAmount(items []game.InventoryItem, itemID int) int {
	for _, item := range items {
		if item.ID == itemID {
			return item.Amount
		}
	}
	return 0
}

func (g *Game) itemLabel(itemID int) string {
	if g.itemDB != nil {
		if item, ok := g.itemDB.Get(itemID); ok && item.Name != "" {
			return item.Name
		}
	}
	return fmt.Sprintf("Item %d", itemID)
}
