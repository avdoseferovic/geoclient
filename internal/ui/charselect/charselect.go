package charselect

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	eoproto "github.com/ethanmoffat/eolib-go/v3/protocol"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/avdo/eoweb/internal/render"
	clientui "github.com/avdo/eoweb/internal/ui"
	"github.com/avdo/eoweb/internal/ui/overlay"
)

const (
	RosterRowHeight   = 48
	RosterTopPadding  = 12
	RosterBottomSpace = 10
	NameMinLength     = 4
	NameMaxLength     = 12
	MaxHairStyle      = 20
	MaxHairColor      = 10
	MaxSkinTone       = 4
)

type Layout struct {
	Dialog           image.Rectangle
	PreviewRect      image.Rectangle
	PreviewArtRect   image.Rectangle
	NamePlateRect    image.Rectangle
	MetaRect         image.Rectangle
	RosterRect       image.Rectangle
	CreateButtonRect image.Rectangle
	JoinButtonRect   image.Rectangle
}

type CreateForm struct {
	Name       []rune
	Focus      int
	Gender     eoproto.Gender
	HairStyle  int
	HairColor  int
	Skin       int
	Submitting bool
}

func NewCreateForm() CreateForm {
	return CreateForm{Gender: eoproto.Gender_Female, HairStyle: 1}
}

func DialogLayout(sw, sh int) Layout {
	dialog := overlay.CenteredRect(556, 380, sw, sh)
	previewRect := image.Rect(dialog.Min.X+24, dialog.Min.Y+44, dialog.Min.X+280, dialog.Max.Y-54)
	previewArtRect := image.Rect(previewRect.Min.X+14, previewRect.Min.Y+16, previewRect.Max.X-14, previewRect.Min.Y+134)
	namePlateRect := image.Rect(previewRect.Min.X+14, previewArtRect.Max.Y+8, previewRect.Max.X-14, previewArtRect.Max.Y+42)
	metaRect := image.Rect(previewRect.Min.X+14, namePlateRect.Max.Y+10, previewRect.Max.X-14, previewRect.Max.Y-14)
	rosterRect := image.Rect(previewRect.Max.X+16, dialog.Min.Y+44, dialog.Max.X-24, dialog.Max.Y-54)
	createButtonRect := image.Rect(dialog.Max.X-282, dialog.Max.Y-44, dialog.Max.X-158, dialog.Max.Y-18)
	joinButtonRect := image.Rect(dialog.Max.X-148, dialog.Max.Y-44, dialog.Max.X-24, dialog.Max.Y-18)
	return Layout{
		Dialog: dialog, PreviewRect: previewRect, PreviewArtRect: previewArtRect,
		NamePlateRect: namePlateRect, MetaRect: metaRect, RosterRect: rosterRect,
		CreateButtonRect: createButtonRect, JoinButtonRect: joinButtonRect,
	}
}

func DrawSelectDialog(screen *ebiten.Image, theme clientui.Theme, layout Layout, characters []server.CharacterSelectionListEntry, selectedIndex, rosterScroll, previewDir, ticks int, selectingChar bool, createOpen bool, form CreateForm, connectError, statusMessage string, loader any) {
	clientui.DrawPanel(screen, layout.Dialog, theme, clientui.PanelOptions{Title: "Characters", Accent: theme.Accent})

	if createOpen || len(characters) == 0 {
		DrawCreateDialog(screen, theme, layout, form, ticks, connectError, statusMessage, len(characters) > 0, loader)
		return
	}

	selected := overlay.ClampInt(selectedIndex, 0, len(characters)-1)
	entry := characters[selected]

	clientui.DrawInset(screen, layout.PreviewRect, theme, false)
	clientui.DrawInset(screen, layout.PreviewArtRect, theme, true)
	clientui.DrawInset(screen, layout.NamePlateRect, theme, false)
	clientui.DrawInset(screen, layout.MetaRect, theme, false)

	drawPreviewBackdrop(screen, layout.PreviewArtRect, theme)
	drawSelectionPreview(screen, layout.PreviewArtRect, entry, previewDir, ticks, loader)
	clientui.DrawText(screen, entry.Name, layout.NamePlateRect.Min.X+12, layout.NamePlateRect.Min.Y+18, theme.Text)
	levelLabel := fmt.Sprintf("LVL %d", entry.Level)
	clientui.DrawText(screen, levelLabel, layout.NamePlateRect.Max.X-clientui.MeasureText(levelLabel)-12, layout.NamePlateRect.Min.Y+18, theme.Accent)
	clientui.DrawText(screen, SummaryLine(entry), layout.MetaRect.Min.X+12, layout.MetaRect.Min.Y+18, theme.TextDim)
	clientui.DrawTextf(screen, layout.MetaRect.Min.X+12, layout.MetaRect.Min.Y+40, theme.TextDim, "%d equipped", EquippedItemCount(entry))

	clientui.DrawInset(screen, layout.RosterRect, theme, false)
	visibleRows := VisibleRows(layout.RosterRect)
	start, end := VisibleRange(len(characters), rosterScroll, visibleRows)
	for rowIndex, i := 0, start; i < end; i, rowIndex = i+1, rowIndex+1 {
		ch := characters[i]
		row := image.Rect(layout.RosterRect.Min.X+10, layout.RosterRect.Min.Y+RosterTopPadding+rowIndex*RosterRowHeight, layout.RosterRect.Max.X-10, layout.RosterRect.Min.Y+RosterTopPadding+40+rowIndex*RosterRowHeight)
		drawSelectionCard(screen, row, theme, ch, i == selected)
	}

	status := "Arrow keys choose • Enter joins"
	statusColor := theme.TextDim
	if selectingChar {
		status = statusMessage
	}
	if connectError != "" {
		status = connectError
		statusColor = theme.Danger
	}
	clientui.DrawText(screen, status, layout.Dialog.Min.X+24, layout.Dialog.Max.Y-28, statusColor)
	clientui.DrawButton(screen, layout.CreateButtonRect, theme, "New Character", false, false)
	clientui.DrawButton(screen, layout.JoinButtonRect, theme, overlay.TernaryString(selectingChar, "Joining", "Enter World"), true, selectingChar)
}

func DrawCreateDialog(screen *ebiten.Image, theme clientui.Theme, layout Layout, form CreateForm, ticks int, connectError, statusMessage string, hasCharacters bool, _ any) {
	clientui.DrawInset(screen, layout.PreviewRect, theme, false)
	clientui.DrawInset(screen, layout.PreviewArtRect, theme, true)
	clientui.DrawInset(screen, layout.NamePlateRect, theme, false)
	clientui.DrawInset(screen, layout.MetaRect, theme, false)
	clientui.DrawInset(screen, layout.RosterRect, theme, false)

	drawPreviewBackdrop(screen, layout.PreviewArtRect, theme)
	clientui.DrawText(screen, overlay.FallbackString(strings.TrimSpace(string(form.Name)), "New Adventurer"), layout.NamePlateRect.Min.X+12, layout.NamePlateRect.Min.Y+18, theme.Text)
	clientui.DrawText(screen, "LVL 1", layout.NamePlateRect.Max.X-clientui.MeasureText("LVL 1")-12, layout.NamePlateRect.Min.Y+18, theme.Accent)
	clientui.DrawText(screen, "Use Left/Right to adjust appearance.", layout.MetaRect.Min.X+12, layout.MetaRect.Min.Y+18, theme.TextDim)
	clientui.DrawText(screen, "Enter confirms.", layout.MetaRect.Min.X+12, layout.MetaRect.Min.Y+40, theme.TextDim)

	formArea := image.Rect(layout.RosterRect.Min.X+10, layout.RosterRect.Min.Y+12, layout.RosterRect.Max.X-10, layout.RosterRect.Max.Y-58)
	actionTop := layout.RosterRect.Max.Y - 42
	cancelRect := image.Rect(layout.RosterRect.Min.X+10, actionTop, layout.RosterRect.Min.X+10+118, actionTop+26)
	createRect := image.Rect(layout.RosterRect.Max.X-128, actionTop, layout.RosterRect.Max.X-10, actionTop+26)

	rows := []struct{ label, value string }{
		{"Name", string(form.Name)},
		{"Gender", overlay.TernaryString(form.Gender == eoproto.Gender_Female, "Female", "Male")},
		{"Hair Style", fmt.Sprintf("%d", form.HairStyle)},
		{"Hair Color", fmt.Sprintf("%d", form.HairColor)},
		{"Skin", fmt.Sprintf("%d", form.Skin)},
	}
	for i, rowData := range rows {
		top := formArea.Min.Y + i*44
		row := image.Rect(formArea.Min.X, top, formArea.Max.X, top+36)
		clientui.DrawInset(screen, row, theme, form.Focus == i)
		value := rowData.value
		if i == 0 && form.Focus == 0 && ticks%40 < 20 {
			value += "_"
		}
		if value == "" {
			value = "Type a name"
		}
		clientui.DrawText(screen, rowData.label, row.Min.X+10, row.Min.Y+16, theme.TextDim)
		clientui.DrawText(screen, value, row.Max.X-clientui.MeasureText(value)-10, row.Min.Y+16, theme.Text)
	}

	status := ""
	statusColor := theme.TextDim
	if form.Submitting {
		status = statusMessage
	}
	if connectError != "" {
		status = connectError
		statusColor = theme.Danger
	}
	if status != "" {
		clientui.DrawText(screen, status, layout.RosterRect.Min.X+10, layout.RosterRect.Max.Y-48, statusColor)
	}
	if hasCharacters {
		clientui.DrawButton(screen, cancelRect, theme, "Cancel", false, form.Submitting)
	}
	clientui.DrawButton(screen, createRect, theme, overlay.TernaryString(form.Submitting, "Creating", "Create"), true, form.Submitting)
}

func SummaryLine(entry server.CharacterSelectionListEntry) string {
	gender := "Adventurer"
	if entry.Gender == 1 {
		gender = "Male"
	}
	if entry.Gender == 0 {
		gender = "Female"
	}
	return gender
}

func EquippedItemCount(entry server.CharacterSelectionListEntry) int {
	count := 0
	for _, v := range []int{entry.Equipment.Armor, entry.Equipment.Boots, entry.Equipment.Hat, entry.Equipment.Shield, entry.Equipment.Weapon} {
		if v > 0 {
			count++
		}
	}
	return count
}

func VisibleRows(rosterRect image.Rectangle) int {
	usableHeight := rosterRect.Dy() - RosterTopPadding - RosterBottomSpace
	if usableHeight <= 0 {
		return 1
	}
	return max(1, usableHeight/RosterRowHeight)
}

func VisibleRange(count, scroll, visibleRows int) (int, int) {
	if count <= 0 {
		return 0, 0
	}
	if visibleRows <= 0 {
		visibleRows = 1
	}
	start := overlay.ClampInt(scroll, 0, max(0, count-visibleRows))
	end := min(count, start+visibleRows)
	return start, end
}

func NormalizeState(count int, selectedChar, rosterScroll, previewDir *int) {
	if count == 0 {
		*previewDir = 0
		*selectedChar = 0
		*rosterScroll = 0
		return
	}
	*previewDir = overlay.ClampInt(*previewDir, 0, 3)
	*selectedChar = overlay.ClampInt(*selectedChar, 0, count-1)
	visibleRows := VisibleRows(image.Rect(0, 0, 236, 210))
	maxScroll := max(0, count-visibleRows)
	targetScroll := *selectedChar - visibleRows/2
	*rosterScroll = overlay.ClampInt(targetScroll, 0, maxScroll)
}

func ValidateCharacterCreate(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "Enter a character name"
	}
	if len(trimmed) < NameMinLength || len(trimmed) > NameMaxLength {
		return fmt.Sprintf("Character names must be %d-%d letters", NameMinLength, NameMaxLength)
	}
	return ""
}

func CycleAppearance(current, delta, maxValue int) int {
	next := current + delta
	for next < 0 {
		next += maxValue
	}
	return next % maxValue
}

func PreviewEntry(form CreateForm) server.CharacterSelectionListEntry {
	return server.CharacterSelectionListEntry{
		Name:      overlay.FallbackString(strings.TrimSpace(string(form.Name)), "New Adventurer"),
		Level:     1,
		Gender:    form.Gender,
		HairStyle: form.HairStyle,
		HairColor: form.HairColor,
		Skin:      form.Skin,
	}
}

func drawSelectionPreview(screen *ebiten.Image, rect image.Rectangle, entry server.CharacterSelectionListEntry, dir, ticks int, _ any) {
	preview := previewEntity(entry, dir, ticks)
	previewX := float64(rect.Min.X + rect.Dx()/2)
	previewY := float64(rect.Min.Y + rect.Dy() - 20 - previewBobOffset(ticks))
	_ = preview
	_ = previewX
	_ = previewY
	// Rendering requires the concrete *gfx.Loader — caller handles this via a callback.
}

func previewEntity(entry server.CharacterSelectionListEntry, dir, ticks int) render.CharacterEntity {
	return render.CharacterEntity{
		Direction: dir, Gender: int(entry.Gender), Skin: entry.Skin,
		HairStyle: entry.HairStyle, HairColor: entry.HairColor,
		Armor: entry.Equipment.Armor, Boots: entry.Equipment.Boots,
		Hat: entry.Equipment.Hat, Weapon: entry.Equipment.Weapon,
		Shield: entry.Equipment.Shield, Frame: previewFrame(dir, ticks),
		Mirrored: dir == 2 || dir == 3,
	}
}

func PreviewEntity(entry server.CharacterSelectionListEntry, dir, ticks int) render.CharacterEntity {
	return previewEntity(entry, dir, ticks)
}

func PreviewBobOffset(ticks int) int {
	return previewBobOffset(ticks)
}

func previewBobOffset(ticks int) int {
	return [...]int{0, 1, 2, 1}[(ticks/10)%4]
}

func previewFrame(dir, ticks int) render.CharacterFrame {
	upLeft := dir == 1 || dir == 2
	phase := (ticks / 16) % 6
	if upLeft {
		switch phase {
		case 1:
			return render.FrameWalkUp1
		case 2:
			return render.FrameWalkUp2
		case 4:
			return render.FrameWalkUp3
		case 5:
			return render.FrameWalkUp4
		default:
			return render.FrameStandUp
		}
	}
	switch phase {
	case 1:
		return render.FrameWalkDown1
	case 2:
		return render.FrameWalkDown2
	case 4:
		return render.FrameWalkDown3
	case 5:
		return render.FrameWalkDown4
	default:
		return render.FrameStandDown
	}
}

func drawPreviewBackdrop(screen *ebiten.Image, rect image.Rectangle, theme clientui.Theme) {
	for y := rect.Min.Y + 3; y < rect.Max.Y-3; y++ {
		t := float64(y-(rect.Min.Y+3)) / float64(max(1, rect.Dy()-6))
		fill := overlay.BlendColors(theme.PanelFillAlt, color.NRGBA{R: 24, G: 20, B: 18, A: 255}, t)
		clientui.FillRect(screen, float64(rect.Min.X+3), float64(y), float64(rect.Dx()-6), 1, fill)
	}
	for x := rect.Min.X + 8; x < rect.Max.X-8; x += 22 {
		clientui.FillRect(screen, float64(x), float64(rect.Min.Y+8), 1, float64(rect.Dy()-16), overlay.Colorize(theme.AccentMuted, 30))
	}
	stage := image.Rect(rect.Min.X+44, rect.Max.Y-30, rect.Max.X-44, rect.Max.Y-12)
	clientui.FillRect(screen, float64(stage.Min.X), float64(stage.Min.Y), float64(stage.Dx()), float64(stage.Dy()), overlay.Colorize(theme.AccentMuted, 90))
	clientui.DrawBorder(screen, stage, theme.BorderDark, theme.BorderMid, theme.Accent)
}

func drawSelectionCard(screen *ebiten.Image, rect image.Rectangle, theme clientui.Theme, ch server.CharacterSelectionListEntry, active bool) {
	fill := theme.PanelFillAlt
	accent := theme.BorderMid
	nameColor := theme.Text
	metaColor := theme.TextDim
	if active {
		fill = color.NRGBA{R: 86, G: 62, B: 34, A: 255}
		accent = theme.Accent
		metaColor = theme.Text
	}
	clientui.FillRect(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), float64(rect.Dy()), fill)
	clientui.DrawBorder(screen, rect, theme.BorderDark, theme.BorderMid, accent)
	clientui.DrawText(screen, ch.Name, rect.Min.X+12, rect.Min.Y+16, nameColor)
	clientui.DrawTextf(screen, rect.Min.X+12, rect.Min.Y+32, metaColor, "Lvl %d • %s", ch.Level, SummaryLine(ch))
}
