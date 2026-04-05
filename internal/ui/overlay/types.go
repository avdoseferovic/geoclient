package overlay

import "image"

type MenuPanel int

const (
	MenuPanelNone MenuPanel = iota
	MenuPanelStats
	MenuPanelInventory
	MenuPanelMap
	MenuPanelGuild
	MenuPanelParty
	MenuPanelOnline
)

type HUDLayout struct {
	StatusRect    image.Rectangle
	MenuRect      image.Rectangle
	MenuPanelRect image.Rectangle
	InfoRect      image.Rectangle
}

type HUDMenuButton struct {
	Panel MenuPanel
	Label string
	Rect  image.Rectangle
}

func InGameHUDLayout(sw, sh int) HUDLayout {
	statusW := 390
	menuHeight := 98
	const edgeInset = 6
	return HUDLayout{
		StatusRect:    image.Rect((sw-statusW)/2, edgeInset, (sw-statusW)/2+statusW, edgeInset+20),
		MenuRect:      image.Rect(sw-158, sh-menuHeight-edgeInset, sw-10, sh-edgeInset),
		MenuPanelRect: image.Rect(sw-290, sh-menuHeight-310-edgeInset, sw-10, sh-menuHeight-8-edgeInset),
		InfoRect:      image.Rectangle{},
	}
}

func TopStatusMeterRects(rect image.Rectangle) [3]image.Rectangle {
	const gap = 8
	inner := image.Rect(rect.Min.X+12, rect.Min.Y, rect.Max.X-12, rect.Max.Y)
	width := (inner.Dx() - gap*2) / 3
	top := rect.Min.Y
	h := rect.Dy()
	return [3]image.Rectangle{
		image.Rect(inner.Min.X, top, inner.Min.X+width, top+h),
		image.Rect(inner.Min.X+width+gap, top, inner.Min.X+width*2+gap, top+h),
		image.Rect(inner.Min.X+(width+gap)*2, top, inner.Max.X, top+h),
	}
}

func HUDMenuButtons(layout HUDLayout) []HUDMenuButton {
	panels := []struct {
		panel MenuPanel
		label string
	}{
		{panel: MenuPanelStats, label: "Stats"},
		{panel: MenuPanelInventory, label: "Bag"},
		{panel: MenuPanelMap, label: "Map"},
		{panel: MenuPanelGuild, label: "Doll"},
		{panel: MenuPanelParty, label: "Party"},
		{panel: MenuPanelOnline, label: "Online"},
	}
	buttons := make([]HUDMenuButton, 0, len(panels))
	for i, entry := range panels {
		col := i % 2
		row := i / 2
		buttons = append(buttons, HUDMenuButton{
			Panel: entry.panel,
			Label: entry.label,
			Rect:  image.Rect(layout.MenuRect.Min.X+12+col*66, layout.MenuRect.Min.Y+16+row*26, layout.MenuRect.Min.X+72+col*66, layout.MenuRect.Min.Y+38+row*26),
		})
	}
	return buttons
}
