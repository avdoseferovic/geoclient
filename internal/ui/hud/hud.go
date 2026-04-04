package hud

import (
	"fmt"
	"image"
	"image/color"

	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/avdo/eoweb/internal/game"
	"github.com/avdo/eoweb/internal/render"
	clientui "github.com/avdo/eoweb/internal/ui"
	"github.com/avdo/eoweb/internal/ui/overlay"
)

const (
	InventoryGridCols  = 8
	InventoryGridRows  = 8
	InventoryGridPages = 2
	InventoryCellSize  = 22
)

type InventoryGridPos struct {
	Page int
	X    int
	Y    int
	W    int
	H    int
}

// StatButton represents a clickable stat training button in the stats panel.
type StatButton struct {
	Rect   image.Rectangle
	StatID int // 1=STR, 2=INT, 3=WIS, 4=AGI, 5=CON, 6=CHA
}

func DrawStats(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot) []StatButton {
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Stats", Accent: theme.AccentMuted})
	sections := []image.Rectangle{
		image.Rect(rect.Min.X+10, rect.Min.Y+22, rect.Max.X-10, rect.Min.Y+72),
		image.Rect(rect.Min.X+10, rect.Min.Y+76, rect.Max.X-10, rect.Min.Y+126),
		image.Rect(rect.Min.X+10, rect.Min.Y+130, rect.Max.X-10, rect.Max.Y-16),
	}
	for _, section := range sections {
		clientui.DrawInset(screen, section, theme, false)
	}
	clientui.DrawTextf(screen, sections[0].Min.X+8, sections[0].Min.Y+14, theme.Text, "%s • Lvl %d", overlay.FallbackString(snapshot.Character.Name, "Wayfarer"), snapshot.Character.Level)
	clientui.DrawTextf(screen, sections[0].Min.X+8, sections[0].Min.Y+28, theme.TextDim, "Class %d • EXP %d", snapshot.Character.ClassID, snapshot.Character.Experience)
	clientui.DrawTextf(screen, sections[0].Min.X+8, sections[0].Min.Y+42, theme.TextDim, "HP %d/%d • TP %d/%d", snapshot.Character.HP, snapshot.Character.MaxHP, snapshot.Character.TP, snapshot.Character.MaxTP)

	hasPoints := snapshot.Character.StatPoints > 0
	pointsColor := theme.Text
	if hasPoints {
		pointsColor = color.NRGBA{R: 110, G: 255, B: 140, A: 255}
	}
	clientui.DrawTextf(screen, sections[1].Min.X+8, sections[1].Min.Y+14, pointsColor, "Stats %d/%d pts", snapshot.Character.StatPoints, snapshot.Character.SkillPoints)

	// Stat buttons (clickable when stat points available)
	var buttons []StatButton
	sx := sections[1].Min.X + 8
	statRow1Y := sections[1].Min.Y + 28
	statRow2Y := sections[1].Min.Y + 42
	bw := 50

	stats := []struct {
		label string
		value int
		id    int
		x, y  int
	}{
		{"STR", snapshot.Character.BaseStats.Str, 1, sx, statRow1Y},
		{"INT", snapshot.Character.BaseStats.Int, 2, sx + bw + 4, statRow1Y},
		{"WIS", snapshot.Character.BaseStats.Wis, 3, sx + (bw+4)*2, statRow1Y},
		{"AGI", snapshot.Character.BaseStats.Agi, 4, sx, statRow2Y},
		{"CON", snapshot.Character.BaseStats.Con, 5, sx + bw + 4, statRow2Y},
		{"CHA", snapshot.Character.BaseStats.Cha, 6, sx + (bw+4)*2, statRow2Y},
	}
	for _, s := range stats {
		r := image.Rect(s.x, s.y-10, s.x+bw, s.y+4)
		txtColor := theme.TextDim
		if hasPoints {
			txtColor = theme.Text
			buttons = append(buttons, StatButton{Rect: r, StatID: s.id})
			// Draw hover highlight
			mx, my := ebiten.CursorPosition()
			if overlay.PointInRect(mx, my, r) {
				clientui.FillRect(screen, float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), overlay.Colorize(theme.Accent, 50))
			}
		}
		clientui.DrawTextf(screen, s.x, s.y, txtColor, "%s %d", s.label, s.value)
	}

	clientui.DrawTextf(screen, sections[2].Min.X+8, sections[2].Min.Y+14, theme.Text, "Combat")
	clientui.DrawTextf(screen, sections[2].Min.X+8, sections[2].Min.Y+28, theme.TextDim, "Dmg %d-%d  Hit %d  Eva %d  Arm %d", snapshot.Character.CombatStats.MinDamage, snapshot.Character.CombatStats.MaxDamage, snapshot.Character.CombatStats.Accuracy, snapshot.Character.CombatStats.Evade, snapshot.Character.CombatStats.Armor)
	clientui.DrawTextf(screen, sections[2].Min.X+8, sections[2].Min.Y+42, theme.TextDim, "Weight %d/%d  Karma %d", snapshot.Character.Weight.Current, snapshot.Character.Weight.Max, snapshot.Character.Karma)
	clientui.DrawTextf(screen, sections[2].Min.X+8, sections[2].Min.Y+56, theme.TextDim, "Bag %d  Players %d  NPCs %d", len(snapshot.Inventory), max(0, len(snapshot.NearbyChars)-1), len(snapshot.NearbyNpcs))
	return buttons
}

func DrawPaperdoll(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot, drawSlot func(*ebiten.Image, clientui.Theme, image.Rectangle, string, int), drawTooltip func(*ebiten.Image, clientui.Theme, int, int, int, int)) {
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Equipment", Accent: theme.Accent})

	// Header — name and guild
	name := overlay.FallbackString(snapshot.Character.Name, "Wayfarer")
	clientui.DrawTextf(screen, rect.Min.X+14, rect.Min.Y+24, theme.Text, "%s", overlay.TruncateLabel(name, 20))
	guild := overlay.TruncateLabel(PaperdollGuildLine(snapshot), 20)
	guildW := clientui.MeasureText(guild)
	clientui.DrawText(screen, guild, rect.Max.X-14-guildW, rect.Min.Y+24, theme.TextDim)

	// Separator
	clientui.FillRect(screen, float64(rect.Min.X+12), float64(rect.Min.Y+36), float64(rect.Dx()-24), 1, overlay.Colorize(theme.BorderMid, 120))

	// Body area
	bodyRect := image.Rect(rect.Min.X+10, rect.Min.Y+42, rect.Max.X-10, rect.Max.Y-14)
	clientui.DrawInset(screen, bodyRect, theme, false)

	// Draw subtle body silhouette lines
	cx := bodyRect.Min.X + bodyRect.Dx()/2
	clientui.FillRect(screen, float64(cx), float64(bodyRect.Min.Y+8), 1, float64(bodyRect.Dy()-16), overlay.Colorize(theme.BorderMid, 30))

	const slotW, slotH = 42, 42
	const smallW, smallH = 32, 32

	// Slot positions — body silhouette layout
	// Row 1: Hat (center-top), Necklace (right of hat)
	hatX := cx - slotW/2
	hatY := bodyRect.Min.Y + 6
	neckX := hatX + slotW + 6
	neckY := hatY + 6

	// Row 2: Weapon (left), Armor (center), Shield (right)
	armorX := cx - slotW/2
	armorY := hatY + slotH + 8
	weaponX := armorX - slotW - 10
	shieldX := armorX + slotW + 10

	// Row 3: Gloves (left), Belt (center), Accessory (right)
	beltY := armorY + slotH + 8
	glovesX := weaponX
	accX := shieldX

	// Row 4: Boots (center)
	bootsY := beltY + slotH + 8

	mx, my := ebiten.CursorPosition()
	tooltipItemID := 0

	slots := PaperdollSlots(snapshot.Equipment, hatX, hatY, neckX, neckY, weaponX, armorX, armorY, shieldX, glovesX, beltY, accX, bootsY, cx, slotW, slotH, smallW, smallH)
	for _, slot := range slots {
		drawPaperdollSlot(screen, theme, slot, drawSlot)
		if slot.ItemID > 0 && overlay.PointInRect(mx, my, slot.Rect) {
			tooltipItemID = slot.ItemID
		}
	}

	// Jewelry row — bottom
	jewY := bootsY + slotH + 10
	jewLabelY := jewY + 12
	jSlotW := 28
	jx := bodyRect.Min.X + 10
	jewSlots := []struct {
		label  string
		itemID int
		rect   image.Rectangle
	}{
		{"R", overlay.PaperdollValue(snapshot.Equipment.Ring, 0), image.Rect(jx, jewY, jx+jSlotW, jewY+jSlotW)},
		{"R", overlay.PaperdollValue(snapshot.Equipment.Ring, 1), image.Rect(jx+jSlotW+4, jewY, jx+jSlotW*2+4, jewY+jSlotW)},
		{"A", overlay.PaperdollValue(snapshot.Equipment.Armlet, 0), image.Rect(jx+jSlotW*2+14, jewY, jx+jSlotW*3+14, jewY+jSlotW)},
		{"A", overlay.PaperdollValue(snapshot.Equipment.Armlet, 1), image.Rect(jx+jSlotW*3+18, jewY, jx+jSlotW*4+18, jewY+jSlotW)},
		{"B", overlay.PaperdollValue(snapshot.Equipment.Bracer, 0), image.Rect(jx+jSlotW*4+28, jewY, jx+jSlotW*5+28, jewY+jSlotW)},
		{"B", overlay.PaperdollValue(snapshot.Equipment.Bracer, 1), image.Rect(jx+jSlotW*5+32, jewY, jx+jSlotW*6+32, jewY+jSlotW)},
	}
	for _, js := range jewSlots {
		drawSlot(screen, theme, js.rect, js.label, js.itemID)
		if js.itemID > 0 && overlay.PointInRect(mx, my, js.rect) {
			tooltipItemID = js.itemID
		}
	}

	// Equipped count
	countLabel := fmt.Sprintf("%d equipped", EquippedItemCount(snapshot.Equipment))
	clientui.DrawText(screen, countLabel, bodyRect.Max.X-10-clientui.MeasureText(countLabel), jewLabelY, theme.TextDim)

	if tooltipItemID != 0 {
		drawTooltip(screen, theme, mx, my, tooltipItemID, 1)
	}
}

// PaperdollSlot represents a single equipment slot for rendering and click detection.
type PaperdollSlot struct {
	Label  string
	ItemID int
	SubLoc int
	Rect   image.Rectangle
}

// PaperdollSlots returns the 9 main equipment slot rects for the given layout params.
func PaperdollSlots(eq server.EquipmentPaperdoll, hatX, hatY, neckX, neckY, weaponX, armorX, armorY, shieldX, glovesX, beltY, accX, bootsY, cx, slotW, slotH, smallW, smallH int) []PaperdollSlot {
	return []PaperdollSlot{
		{"Hat", eq.Hat, 7, image.Rect(hatX, hatY, hatX+slotW, hatY+slotH)},
		{"Neck", eq.Necklace, 6, image.Rect(neckX, neckY, neckX+smallW, neckY+smallH)},
		{"Weapon", eq.Weapon, 9, image.Rect(weaponX, armorY, weaponX+slotW, armorY+slotH)},
		{"Armor", eq.Armor, 5, image.Rect(armorX, armorY, armorX+slotW, armorY+slotH)},
		{"Shield", eq.Shield, 8, image.Rect(shieldX, armorY, shieldX+slotW, armorY+slotH)},
		{"Gloves", eq.Gloves, 3, image.Rect(glovesX, beltY, glovesX+slotW, beltY+slotH)},
		{"Belt", eq.Belt, 4, image.Rect(armorX, beltY, armorX+slotW, beltY+slotH)},
		{"Acc", eq.Accessory, 2, image.Rect(accX, beltY, accX+slotW, beltY+slotH)},
		{"Boots", eq.Boots, 1, image.Rect(cx-slotW/2, bootsY, cx-slotW/2+slotW, bootsY+slotH)},
	}
}

func drawPaperdollSlot(screen *ebiten.Image, theme clientui.Theme, slot PaperdollSlot, drawSlot func(*ebiten.Image, clientui.Theme, image.Rectangle, string, int)) {
	// Slot background with subtle highlight when filled
	if slot.ItemID > 0 {
		clientui.FillRect(screen, float64(slot.Rect.Min.X), float64(slot.Rect.Min.Y), float64(slot.Rect.Dx()), float64(slot.Rect.Dy()), overlay.Colorize(theme.AccentMuted, 40))
	}
	drawSlot(screen, theme, slot.Rect, slot.Label, slot.ItemID)
}

func DrawMap(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot, autoWalkActive bool, autoWalkStatus string, drawMinimap func(*ebiten.Image, clientui.Theme, image.Rectangle, game.UISnapshot)) {
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Map", Accent: theme.AccentMuted})
	clientui.DrawTextf(screen, rect.Min.X+12, rect.Min.Y+24, theme.TextDim, "Map %d", snapshot.Character.MapID)
	clientui.DrawTextf(screen, rect.Max.X-86, rect.Min.Y+24, theme.TextDim, "%d,%d", snapshot.Character.X, snapshot.Character.Y)
	mapRect := image.Rect(rect.Min.X+10, rect.Min.Y+36, rect.Max.X-10, rect.Max.Y-26)
	clientui.DrawInset(screen, mapRect, theme, true)
	drawMinimap(screen, theme, mapRect, snapshot)
	if autoWalkActive {
		clientui.DrawTextCentered(screen, autoWalkStatus, image.Rect(rect.Min.X+12, rect.Max.Y-24, rect.Max.X-12, rect.Max.Y-10), theme.TextDim)
	}
}

func DrawInventory(screen *ebiten.Image, theme clientui.Theme, rect image.Rectangle, snapshot game.UISnapshot, positions map[int]InventoryGridPos, page, selected int, drawIcon func(*ebiten.Image, image.Rectangle, int, int), drawTooltip func(*ebiten.Image, clientui.Theme, int, int, int, int)) {
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Inventory", Accent: theme.AccentMuted})
	items := snapshot.Inventory
	clientui.DrawTextf(screen, rect.Min.X+12, rect.Min.Y+24, theme.TextDim, "Weight %d/%d", snapshot.Character.Weight.Current, snapshot.Character.Weight.Max)
	itemsLabel := fmt.Sprintf("%d items", len(items))
	clientui.DrawText(screen, itemsLabel, rect.Max.X-12-clientui.MeasureText(itemsLabel), rect.Min.Y+24, theme.TextDim)

	if len(items) == 0 {
		clientui.DrawTextWrappedCentered(screen, "No inventory items reported yet.", image.Rect(rect.Min.X+12, rect.Min.Y+58, rect.Max.X-12, rect.Max.Y-86), theme.Text)
		return
	}

	for i, tabRect := range PageTabRects(rect) {
		clientui.DrawButton(screen, tabRect, theme, fmt.Sprintf("Pack %d", i+1), page == i, false)
	}

	gridRect := GridRect(rect)
	clientui.DrawInset(screen, gridRect, theme, false)
	mx, my := ebiten.CursorPosition()
	tooltipItemID := 0
	tooltipAmount := 0
	for row := 0; row <= InventoryGridRows; row++ {
		y := gridRect.Min.Y + row*InventoryCellSize
		clientui.FillRect(screen, float64(gridRect.Min.X), float64(y), float64(gridRect.Dx()), 1, overlay.Colorize(theme.BorderMid, 64))
	}
	for col := 0; col <= InventoryGridCols; col++ {
		x := gridRect.Min.X + col*InventoryCellSize
		clientui.FillRect(screen, float64(x), float64(gridRect.Min.Y), 1, float64(gridRect.Dy()), overlay.Colorize(theme.BorderMid, 64))
	}

	for index, item := range items {
		pos, ok := positions[item.ID]
		if !ok || pos.Page != page {
			continue
		}
		entry := image.Rect(
			gridRect.Min.X+pos.X*InventoryCellSize+1,
			gridRect.Min.Y+pos.Y*InventoryCellSize+1,
			gridRect.Min.X+(pos.X+pos.W)*InventoryCellSize,
			gridRect.Min.Y+(pos.Y+pos.H)*InventoryCellSize,
		)
		active := index == selected
		if active {
			clientui.FillRect(screen, float64(entry.Min.X), float64(entry.Min.Y), float64(entry.Dx()), float64(entry.Dy()), overlay.Colorize(theme.AccentMuted, 84))
		}
		drawIcon(screen, entry, item.ID, item.Amount)
		if overlay.PointInRect(mx, my, entry) {
			tooltipItemID = item.ID
			tooltipAmount = item.Amount
			clientui.FillRect(screen, float64(entry.Min.X), float64(entry.Min.Y), float64(entry.Dx()), float64(entry.Dy()), overlay.Colorize(theme.Accent, 38))
		}
	}

	if tooltipItemID != 0 {
		drawTooltip(screen, theme, mx, my, tooltipItemID, tooltipAmount)
	}
}

func PageTabRects(rect image.Rectangle) []image.Rectangle {
	return []image.Rectangle{
		image.Rect(rect.Min.X+10, rect.Min.Y+36, rect.Min.X+74, rect.Min.Y+56),
		image.Rect(rect.Min.X+80, rect.Min.Y+36, rect.Min.X+144, rect.Min.Y+56),
	}
}

func GridRect(rect image.Rectangle) image.Rectangle {
	width := InventoryGridCols * InventoryCellSize
	height := InventoryGridRows * InventoryCellSize
	x := rect.Min.X + (rect.Dx()-width)/2
	return image.Rect(x, rect.Min.Y+62, x+width, rect.Min.Y+62+height)
}

func InventoryFits(grid [][]bool, x, y, w, h int) bool {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if grid[yy][xx] {
				return false
			}
		}
	}
	return true
}

func InventoryOccupy(grid [][]bool, x, y, w, h int) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			grid[yy][xx] = true
		}
	}
}

func PaperdollGuildLine(snapshot game.UISnapshot) string {
	if len(snapshot.Character.GuildName) == 0 {
		return "No guild"
	}
	if len(snapshot.Character.GuildTag) == 0 {
		return snapshot.Character.GuildName
	}
	return fmt.Sprintf("[%s] %s", snapshot.Character.GuildTag, snapshot.Character.GuildName)
}

func EquippedItemCount(equipment server.EquipmentPaperdoll) int {
	count := 0
	for _, v := range []int{equipment.Boots, equipment.Accessory, equipment.Gloves, equipment.Belt, equipment.Armor, equipment.Necklace, equipment.Hat, equipment.Shield, equipment.Weapon} {
		if v > 0 {
			count++
		}
	}
	for _, group := range [][]int{equipment.Ring, equipment.Armlet, equipment.Bracer} {
		for _, v := range group {
			if v > 0 {
				count++
			}
		}
	}
	return count
}

func MinimapTileColor(tileState int, theme clientui.Theme) color.Color {
	switch tileState {
	case 2:
		return color.NRGBA{R: 38, G: 34, B: 42, A: 255}
	case 1:
		return overlay.Colorize(theme.AccentMuted, 120)
	default:
		return color.NRGBA{R: 91, G: 108, B: 88, A: 255}
	}
}

// Suppress unused import warnings for render — used by callers via types.
var _ render.CharacterEntity
