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

const (
	shopViewMenu = iota
	shopViewSection
)

type shopListEntry struct {
	ItemID  int
	Title   string
	Detail  string
	Enabled bool
	Action  int
}

type shopSectionButton struct {
	ID        int
	Label     string
	Count     int
	Available bool
}

func (g *Game) shopDialogRect() image.Rectangle {
	w, h := 500, 340
	x := (g.screenW - w) / 2
	y := (g.screenH - h) / 2
	return image.Rect(x, y, x+w, y+h)
}

func (g *Game) drawShopDialog(screen *ebiten.Image, theme clientui.Theme) {
	snapshot := g.client.UISnapshot()
	g.normalizeShopState(snapshot)

	rect := g.shopDialogRect()
	title := snapshot.Shop.Name
	if title == "" {
		title = "Shop"
	}
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: title, Accent: theme.Accent})

	mx, my := ebiten.CursorPosition()
	closeRect := image.Rect(rect.Max.X-74, rect.Max.Y-32, rect.Max.X-14, rect.Max.Y-12)

	if g.overlay.shopView == shopViewMenu {
		g.drawShopMainMenu(screen, theme, rect, snapshot, mx, my)
	} else {
		g.drawShopSectionView(screen, theme, rect, snapshot, mx, my)
	}

	clientui.DrawButton(screen, closeRect, theme, "Close", false, overlay.PointInRect(mx, my, closeRect))
}

func (g *Game) drawShopMainMenu(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot, mx, my int) {
	buttons := g.shopSectionButtons(snapshot)
	contentRect := image.Rect(rect.Min.X+16, rect.Min.Y+36, rect.Max.X-16, rect.Max.Y-46)
	clientui.DrawInset(screen, contentRect, theme, false)
	clientui.DrawText(screen, "Buy stock, sell items from your bag, or craft from ingredients.", contentRect.Min.X+16, contentRect.Min.Y+24, theme.TextDim)

	for i, button := range buttons {
		cardTop := contentRect.Min.Y + 52 + i*58
		cardRect := image.Rect(contentRect.Min.X+18, cardTop, contentRect.Max.X-18, cardTop+48)
		hovered := overlay.PointInRect(mx, my, cardRect)
		selected := g.overlay.shopTab == button.ID
		g.drawShopMenuCard(screen, theme, snapshot, cardRect, button, hovered, selected)
	}
}

func (g *Game) drawShopMenuCard(screen *ebiten.Image, theme clientui.Theme, snapshot game.UISnapshot, rect image.Rectangle, button shopSectionButton, hovered, selected bool) {
	clientui.DrawInset(screen, rect, theme, selected || hovered)
	if selected || hovered {
		clientui.FillRect(screen, float64(rect.Min.X+2), float64(rect.Min.Y+2), float64(rect.Dx()-4), float64(rect.Dy()-4), overlay.Colorize(theme.AccentMuted, 40))
	}

	clientui.DrawText(screen, button.Label, rect.Min.X+14, rect.Min.Y+16, overlay.TernaryColor(button.Available, theme.Text, theme.TextDim))
	countLabel := fmt.Sprintf("%d items", button.Count)
	if button.Count == 1 {
		countLabel = "1 item"
	}
	clientui.DrawText(screen, countLabel, rect.Min.X+110, rect.Min.Y+16, theme.TextDim)
	clientui.DrawText(screen, g.shopSectionDescription(button.ID), rect.Min.X+14, rect.Min.Y+32, theme.TextDim)

	if !button.Available {
		clientui.DrawText(screen, "Unavailable", rect.Max.X-92, rect.Min.Y+16, theme.Danger)
		return
	}
	clientui.DrawText(screen, "Open", rect.Max.X-52, rect.Min.Y+16, theme.Accent)
}

func (g *Game) drawShopSectionView(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot, mx, my int) {
	backRect := image.Rect(rect.Min.X+16, rect.Min.Y+34, rect.Min.X+76, rect.Min.Y+56)
	titleRect := image.Rect(backRect.Max.X+10, rect.Min.Y+34, rect.Max.X-144, rect.Min.Y+56)
	listRect := image.Rect(rect.Min.X+16, rect.Min.Y+66, rect.Min.X+214, rect.Max.Y-46)
	detailRect := image.Rect(listRect.Max.X+10, rect.Min.Y+66, rect.Max.X-16, rect.Max.Y-46)
	actionRect := image.Rect(detailRect.Max.X-110, detailRect.Max.Y-28, detailRect.Max.X-10, detailRect.Max.Y-8)

	clientui.DrawButton(screen, backRect, theme, "Back", false, overlay.PointInRect(mx, my, backRect))
	clientui.DrawInset(screen, titleRect, theme, false)
	clientui.DrawText(screen, fmt.Sprintf("%s Counter", g.shopSectionLabel(g.overlay.shopTab)), titleRect.Min.X+10, titleRect.Min.Y+14, theme.Text)

	entries := g.shopEntries(snapshot)
	selected, ok := g.shopSelectedEntry(snapshot)
	clientui.DrawInset(screen, listRect, theme, false)
	if len(entries) == 0 {
		clientui.DrawTextCentered(screen, "Nothing available in this section.", listRect, theme.TextDim)
	} else {
		for i, entry := range entries {
			rowRect := g.shopEntryRowRect(listRect, i)
			if rowRect.Max.Y > listRect.Max.Y-6 {
				break
			}
			g.drawShopEntryRow(screen, theme, snapshot, rowRect, entry, entry.ItemID == g.overlay.shopSelectedItem, overlay.PointInRect(mx, my, rowRect))
		}
	}

	clientui.DrawInset(screen, detailRect, theme, false)
	if !ok {
		clientui.DrawTextCentered(screen, "Choose an item to see details.", detailRect, theme.TextDim)
		return
	}

	g.drawShopEntryDetails(screen, theme, snapshot, detailRect, selected)
	clientui.DrawButton(screen, actionRect, theme, g.shopActionLabel(selected.Action), true, !selected.Enabled)
}

func (g *Game) drawShopEntryRow(screen *ebiten.Image, theme clientui.Theme, _ game.UISnapshot, rect image.Rectangle, entry shopListEntry, selected, hovered bool) {
	if selected {
		clientui.FillRect(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), float64(rect.Dy()), overlay.Colorize(theme.Accent, 56))
	} else if hovered {
		clientui.FillRect(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), float64(rect.Dy()), overlay.Colorize(theme.AccentMuted, 40))
	}

	iconRect := image.Rect(rect.Min.X+6, rect.Min.Y+5, rect.Min.X+38, rect.Min.Y+37)
	clientui.DrawInset(screen, iconRect, theme, false)
	g.drawItemIcon(screen, iconRect.Inset(4), entry.ItemID, 1)

	titleColor := overlay.TernaryColor(entry.Enabled, theme.Text, theme.TextDim)
	detailColor := overlay.TernaryColor(entry.Enabled, theme.TextDim, theme.Danger)
	clientui.DrawText(screen, entry.Title, rect.Min.X+46, rect.Min.Y+14, titleColor)
	clientui.DrawText(screen, entry.Detail, rect.Min.X+46, rect.Min.Y+29, detailColor)
}

func (g *Game) drawShopEntryDetails(screen *ebiten.Image, theme clientui.Theme, snapshot game.UISnapshot, rect image.Rectangle, entry shopListEntry) {
	iconRect := image.Rect(rect.Min.X+14, rect.Min.Y+14, rect.Min.X+94, rect.Min.Y+94)
	clientui.DrawInset(screen, iconRect, theme, false)
	g.drawItemIcon(screen, iconRect.Inset(10), entry.ItemID, 1)

	textX := iconRect.Max.X + 16
	textWidth := rect.Max.X - textX - 14
	lineY := rect.Min.Y + 20
	clientui.DrawText(screen, entry.Title, textX, lineY, theme.Text)
	lineY += 18
	for _, line := range g.wrapShopText(g.shopEntryHeadline(snapshot, entry), textWidth) {
		clientui.DrawText(screen, line, textX, lineY, theme.TextDim)
		lineY += 14
	}
	for _, line := range g.wrapShopText(g.shopEntryAvailability(snapshot, entry), textWidth) {
		clientui.DrawText(screen, line, textX, lineY, overlay.TernaryColor(entry.Enabled, theme.Success, theme.Danger))
		lineY += 14
	}
	for _, meta := range g.shopEntryMetaLines(snapshot, entry) {
		for _, line := range g.wrapShopText(meta, textWidth) {
			clientui.DrawText(screen, line, textX, lineY, theme.TextDim)
			lineY += 14
		}
	}

	if entry.Action != shopTabCraft {
		return
	}

	clientui.DrawText(screen, "Ingredients", rect.Min.X+14, rect.Min.Y+128, theme.Text)
	ingredientsRect := image.Rect(rect.Min.X+14, rect.Min.Y+138, rect.Max.X-14, rect.Max.Y-42)
	g.drawCraftIngredients(screen, theme, snapshot, ingredientsRect, g.shopCraftItem(snapshot, entry.ItemID))
}

func (g *Game) drawCraftIngredients(screen *ebiten.Image, theme clientui.Theme, snapshot game.UISnapshot, rect image.Rectangle, craft game.ShopCraftItem) {
	clientui.DrawInset(screen, rect, theme, false)
	rowY := rect.Min.Y + 10
	count := 0
	for _, ingredient := range craft.Ingredients {
		if ingredient.Id <= 0 || ingredient.Amount <= 0 {
			continue
		}
		rowRect := image.Rect(rect.Min.X+6, rowY, rect.Max.X-6, rowY+38)
		if rowRect.Max.Y > rect.Max.Y-6 {
			break
		}
		clientui.DrawInset(screen, rowRect, theme, false)
		iconRect := image.Rect(rowRect.Min.X+6, rowRect.Min.Y+4, rowRect.Min.X+38, rowRect.Min.Y+36)
		clientui.DrawInset(screen, iconRect, theme, false)
		g.drawItemIcon(screen, iconRect.Inset(4), int(ingredient.Id), int(ingredient.Amount))
		owned := g.inventoryAmount(snapshot.Inventory, int(ingredient.Id))
		ok := owned >= int(ingredient.Amount)
		clientui.DrawText(screen, g.itemLabel(int(ingredient.Id)), rowRect.Min.X+46, rowRect.Min.Y+14, theme.Text)
		clientui.DrawText(screen, fmt.Sprintf("Need %d  |  Bag %d", ingredient.Amount, owned), rowRect.Min.X+46, rowRect.Min.Y+28, overlay.TernaryColor(ok, theme.Success, theme.Danger))
		rowY += 42
		count++
	}
	if count == 0 {
		clientui.DrawTextCentered(screen, "No ingredients required.", rect, theme.TextDim)
	}
}

func (g *Game) updateShopDialog(mx, my int) bool {
	snapshot := g.client.UISnapshot()
	g.normalizeShopState(snapshot)
	rect := g.shopDialogRect()

	closeRect := image.Rect(rect.Max.X-74, rect.Max.Y-32, rect.Max.X-14, rect.Max.Y-12)
	if overlay.PointInRect(mx, my, closeRect) {
		g.overlay.shopDialogOpen = false
		return true
	}

	if g.overlay.shopView == shopViewMenu {
		for _, button := range g.shopMenuButtonRects(rect, snapshot) {
			if !overlay.PointInRect(mx, my, button.Rect) {
				continue
			}
			if !button.Available {
				g.overlay.statusMessage = fmt.Sprintf("%s is unavailable here", button.Label)
				return true
			}
			g.overlay.shopTab = button.ID
			g.overlay.shopView = shopViewSection
			g.overlay.shopSelectedItem = 0
			g.normalizeShopState(snapshot)
			return true
		}
		return true
	}

	backRect := image.Rect(rect.Min.X+16, rect.Min.Y+34, rect.Min.X+76, rect.Min.Y+56)
	if overlay.PointInRect(mx, my, backRect) {
		g.overlay.shopView = shopViewMenu
		return true
	}

	listRect := image.Rect(rect.Min.X+16, rect.Min.Y+66, rect.Min.X+214, rect.Max.Y-46)
	entries := g.shopEntries(snapshot)
	for i, entry := range entries {
		rowRect := g.shopEntryRowRect(listRect, i)
		if rowRect.Max.Y > listRect.Max.Y-6 {
			break
		}
		if overlay.PointInRect(mx, my, rowRect) {
			g.overlay.shopSelectedItem = entry.ItemID
			return true
		}
	}

	selected, ok := g.shopSelectedEntry(snapshot)
	if !ok {
		return true
	}

	detailRect := image.Rect(listRect.Max.X+10, rect.Min.Y+66, rect.Max.X-16, rect.Max.Y-46)
	actionRect := image.Rect(detailRect.Max.X-110, detailRect.Max.Y-28, detailRect.Max.X-10, detailRect.Max.Y-8)
	if overlay.PointInRect(mx, my, actionRect) {
		g.performShopAction(snapshot, selected)
		return true
	}

	return true
}

func (g *Game) shopMenuButtonRects(rect image.Rectangle, snapshot game.UISnapshot) []struct {
	shopSectionButton
	Rect image.Rectangle
} {
	buttons := g.shopSectionButtons(snapshot)
	contentRect := image.Rect(rect.Min.X+16, rect.Min.Y+36, rect.Max.X-16, rect.Max.Y-46)
	result := make([]struct {
		shopSectionButton
		Rect image.Rectangle
	}, 0, len(buttons))
	for i, button := range buttons {
		cardTop := contentRect.Min.Y + 58 + i*58
		cardRect := image.Rect(contentRect.Min.X+18, cardTop, contentRect.Max.X-18, cardTop+48)
		result = append(result, struct {
			shopSectionButton
			Rect image.Rectangle
		}{
			shopSectionButton: button,
			Rect:              cardRect,
		})
	}
	return result
}

func (g *Game) normalizeShopState(snapshot game.UISnapshot) {
	buttons := g.shopSectionButtons(snapshot)
	firstAvailable := -1
	for _, button := range buttons {
		if button.Available {
			firstAvailable = button.ID
			break
		}
	}
	if firstAvailable == -1 {
		g.overlay.shopView = shopViewMenu
		g.overlay.shopSelectedItem = 0
		return
	}

	if !g.shopSectionAvailable(snapshot, g.overlay.shopTab) {
		g.overlay.shopTab = firstAvailable
	}

	if g.overlay.shopView != shopViewSection {
		return
	}

	entries := g.shopEntries(snapshot)
	if len(entries) == 0 {
		g.overlay.shopView = shopViewMenu
		g.overlay.shopSelectedItem = 0
		return
	}
	for _, entry := range entries {
		if entry.ItemID == g.overlay.shopSelectedItem {
			return
		}
	}
	g.overlay.shopSelectedItem = entries[0].ItemID
}

func (g *Game) shopSectionButtons(snapshot game.UISnapshot) []shopSectionButton {
	return []shopSectionButton{
		{ID: shopTabBuy, Label: "Buy", Count: len(g.shopBuyItems(snapshot)), Available: len(g.shopBuyItems(snapshot)) > 0},
		{ID: shopTabSell, Label: "Sell", Count: len(g.shopSellItems(snapshot)), Available: len(g.shopSellItems(snapshot)) > 0},
		{ID: shopTabCraft, Label: "Craft", Count: len(snapshot.Shop.CraftItems), Available: len(snapshot.Shop.CraftItems) > 0},
	}
}

func (g *Game) shopSectionAvailable(snapshot game.UISnapshot, tab int) bool {
	for _, button := range g.shopSectionButtons(snapshot) {
		if button.ID == tab {
			return button.Available
		}
	}
	return false
}

func (g *Game) shopEntries(snapshot game.UISnapshot) []shopListEntry {
	switch g.overlay.shopTab {
	case shopTabSell:
		sells := g.shopSellItems(snapshot)
		entries := make([]shopListEntry, 0, len(sells))
		for _, item := range sells {
			bagAmount := g.inventoryAmount(snapshot.Inventory, item.ItemID)
			entries = append(entries, shopListEntry{
				ItemID:  item.ItemID,
				Title:   g.itemLabel(item.ItemID),
				Detail:  fmt.Sprintf("Price %d | Bag %d", item.SellPrice, bagAmount),
				Enabled: bagAmount > 0,
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
				Enabled: item.MaxBuyAmount > 0 && g.inventoryAmount(snapshot.Inventory, 1) >= item.BuyPrice,
				Action:  shopTabBuy,
			})
		}
		return entries
	}
}

func (g *Game) shopSelectedEntry(snapshot game.UISnapshot) (shopListEntry, bool) {
	entries := g.shopEntries(snapshot)
	for _, entry := range entries {
		if entry.ItemID == g.overlay.shopSelectedItem {
			return entry, true
		}
	}
	if len(entries) == 0 {
		return shopListEntry{}, false
	}
	return entries[0], true
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
		if item.SellPrice > 0 {
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

func (g *Game) shopSellPrice(snapshot game.UISnapshot, itemID int) int {
	for _, item := range snapshot.Shop.TradeItems {
		if item.ItemID == itemID {
			return item.SellPrice
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
			if g.inventoryAmount(snapshot.Inventory, int(ingredient.Id)) < int(ingredient.Amount) {
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
		parts = append(parts, fmt.Sprintf("%dx %s", ingredient.Amount, g.itemLabel(int(ingredient.Id))))
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

func (g *Game) performShopAction(snapshot game.UISnapshot, entry shopListEntry) {
	switch entry.Action {
	case shopTabBuy:
		price := g.shopBuyPrice(snapshot, entry.ItemID)
		if gold := g.inventoryAmount(snapshot.Inventory, 1); gold < price {
			g.overlay.statusMessage = "Not enough gold"
			return
		}
		g.sendShopBuy(entry.ItemID, 1)
		g.overlay.statusMessage = fmt.Sprintf("Bought %s", entry.Title)
	case shopTabSell:
		if g.inventoryAmount(snapshot.Inventory, entry.ItemID) <= 0 {
			g.overlay.statusMessage = "You do not have that item"
			return
		}
		g.sendShopSell(entry.ItemID, 1)
		g.overlay.statusMessage = fmt.Sprintf("Sold %s", entry.Title)
	case shopTabCraft:
		if !g.canCraft(snapshot, entry.ItemID) {
			g.overlay.statusMessage = "Missing crafting ingredients"
			return
		}
		g.sendShopCraft(entry.ItemID)
		g.overlay.statusMessage = fmt.Sprintf("Crafting %s", entry.Title)
	}
}

func (g *Game) shopActionLabel(action int) string {
	switch action {
	case shopTabSell:
		return "Sell One"
	case shopTabCraft:
		return "Craft"
	default:
		return "Buy One"
	}
}

func (g *Game) shopSectionLabel(tab int) string {
	for _, button := range g.shopSectionButtons(game.UISnapshot{}) {
		if button.ID == tab {
			return button.Label
		}
	}
	switch tab {
	case shopTabSell:
		return "Sell"
	case shopTabCraft:
		return "Craft"
	default:
		return "Buy"
	}
}

func (g *Game) shopSectionDescription(tab int) string {
	switch tab {
	case shopTabSell:
		return "Turn spare drops into gold."
	case shopTabCraft:
		return "Combine ingredients into gear and supplies."
	default:
		return "Browse stock and buy one item at a time."
	}
}

func (g *Game) shopEntryHeadline(snapshot game.UISnapshot, entry shopListEntry) string {
	switch entry.Action {
	case shopTabSell:
		return fmt.Sprintf("Sell price: %d gold", g.shopSellPrice(snapshot, entry.ItemID))
	case shopTabCraft:
		return "Crafting recipe"
	default:
		return fmt.Sprintf("Buy price: %d gold", g.shopBuyPrice(snapshot, entry.ItemID))
	}
}

func (g *Game) shopEntryAvailability(snapshot game.UISnapshot, entry shopListEntry) string {
	switch entry.Action {
	case shopTabSell:
		amount := g.inventoryAmount(snapshot.Inventory, entry.ItemID)
		if amount > 0 {
			return fmt.Sprintf("Bag count: %d", amount)
		}
		return "Not in bag"
	case shopTabCraft:
		if entry.Enabled {
			return "Ingredients ready"
		}
		return "Missing ingredients"
	default:
		price := g.shopBuyPrice(snapshot, entry.ItemID)
		gold := g.inventoryAmount(snapshot.Inventory, 1)
		if gold >= price {
			return fmt.Sprintf("Gold: %d", gold)
		}
		return fmt.Sprintf("Need %d more gold", price-gold)
	}
}

func (g *Game) shopEntryMetaLines(snapshot game.UISnapshot, entry shopListEntry) []string {
	switch entry.Action {
	case shopTabSell:
		return []string{
			fmt.Sprintf("Value: %d gold", g.shopSellPrice(snapshot, entry.ItemID)),
			"Sell one copy.",
		}
	case shopTabCraft:
		return []string{
			"Consumes the ingredients below.",
		}
	default:
		return []string{
			fmt.Sprintf("Price: %d gold", g.shopBuyPrice(snapshot, entry.ItemID)),
			"Buy one copy.",
		}
	}
}

func (g *Game) shopCraftItem(snapshot game.UISnapshot, itemID int) game.ShopCraftItem {
	for _, item := range snapshot.Shop.CraftItems {
		if item.ItemID == itemID {
			return item
		}
	}
	return game.ShopCraftItem{}
}

func (g *Game) shopEntryRowRect(listRect image.Rectangle, index int) image.Rectangle {
	top := listRect.Min.Y + 8 + index*42
	return image.Rect(listRect.Min.X+6, top, listRect.Max.X-6, top+38)
}

func (g *Game) wrapShopText(text string, maxWidth int) []string {
	if text == "" {
		return nil
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}
	lines := make([]string, 0, 3)
	current := words[0]
	for _, word := range words[1:] {
		candidate := current + " " + word
		if clientui.MeasureText(candidate) <= maxWidth {
			current = candidate
			continue
		}
		lines = append(lines, current)
		current = word
	}
	lines = append(lines, current)
	return lines
}
