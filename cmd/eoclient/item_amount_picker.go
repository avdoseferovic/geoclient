package main

import (
	"image"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	clientui "github.com/avdoseferovic/geoclient/internal/ui"
	"github.com/avdoseferovic/geoclient/internal/ui/overlay"
)

type itemAmountPickerState struct {
	Active bool
	ItemID int
	Max    int
	TileX  int
	TileY  int
	Action itemAmountAction

	Input    string
	inputBuf []rune
}

type itemAmountAction int

const (
	itemAmountActionDrop itemAmountAction = iota
	itemAmountActionTradeAdd
)

func (g *Game) openItemAmountPicker(itemID, maxAmount, tileX, tileY int) {
	if maxAmount <= 1 {
		return
	}
	input := strconv.Itoa(maxAmount)
	g.overlay.itemAmountPicker = itemAmountPickerState{
		Active:   true,
		ItemID:   itemID,
		Max:      maxAmount,
		TileX:    tileX,
		TileY:    tileY,
		Action:   itemAmountActionDrop,
		Input:    input,
		inputBuf: []rune(input),
	}
}

func (g *Game) openTradeAmountPicker(itemID, maxAmount int) {
	if maxAmount <= 1 {
		return
	}
	input := strconv.Itoa(maxAmount)
	g.overlay.itemAmountPicker = itemAmountPickerState{
		Active:   true,
		ItemID:   itemID,
		Max:      maxAmount,
		Action:   itemAmountActionTradeAdd,
		Input:    input,
		inputBuf: []rune(input),
	}
}

func (g *Game) closeItemAmountPicker() {
	g.overlay.itemAmountPicker = itemAmountPickerState{}
}

func (g *Game) updateItemAmountPicker() bool {
	if !g.overlay.itemAmountPicker.Active {
		return false
	}

	picker := &g.overlay.itemAmountPicker
	rect := g.itemAmountPickerRect()
	confirmRect, cancelRect := g.itemAmountPickerButtonRects(rect)

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.closeItemAmountPicker()
		return true
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		if overlay.PointInRect(mx, my, confirmRect) {
			return g.confirmItemAmountPicker()
		}
		if overlay.PointInRect(mx, my, cancelRect) {
			g.closeItemAmountPicker()
			return true
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		return g.confirmItemAmountPicker()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(picker.inputBuf) > 0 {
		picker.inputBuf = picker.inputBuf[:len(picker.inputBuf)-1]
		picker.Input = string(picker.inputBuf)
		return true
	}

	for _, r := range ebiten.AppendInputChars(nil) {
		if r < '0' || r > '9' {
			continue
		}
		if len(picker.inputBuf) >= len(strconv.Itoa(picker.Max)) {
			continue
		}
		if len(picker.inputBuf) == 1 && picker.inputBuf[0] == '0' {
			picker.inputBuf = picker.inputBuf[:0]
		}
		picker.inputBuf = append(picker.inputBuf, r)
		picker.Input = string(picker.inputBuf)
	}
	return true
}

func (g *Game) confirmItemAmountPicker() bool {
	picker := g.overlay.itemAmountPicker
	amount := g.itemAmountPickerValue()
	if amount <= 0 {
		g.overlay.statusMessage = "Enter a valid amount"
		return true
	}
	switch picker.Action {
	case itemAmountActionTradeAdd:
		g.sendTradeAdd(picker.ItemID, amount)
		g.overlay.statusMessage = "Added item to trade"
	default:
		g.sendDropItem(picker.ItemID, amount, picker.TileX, picker.TileY)
		g.overlay.statusMessage = "Dropped item"
	}
	g.closeItemAmountPicker()
	return true
}

func (g *Game) itemAmountPickerValue() int {
	picker := g.overlay.itemAmountPicker
	if !picker.Active || picker.Input == "" {
		return 0
	}
	value, err := strconv.Atoi(picker.Input)
	if err != nil || value <= 0 {
		return 0
	}
	if value > picker.Max {
		return picker.Max
	}
	return value
}

func (g *Game) drawItemAmountPicker(screen *ebiten.Image, theme clientui.Theme) {
	if !g.overlay.itemAmountPicker.Active {
		return
	}

	rect := g.itemAmountPickerRect()
	title := "Drop Amount"
	prompt := "How many items?"
	if g.overlay.itemAmountPicker.Action == itemAmountActionTradeAdd {
		title = "Trade Amount"
		prompt = "How many to offer?"
	}
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: title, Accent: theme.Accent})

	picker := g.overlay.itemAmountPicker
	clientui.DrawTextCentered(screen, prompt, image.Rect(rect.Min.X+12, rect.Min.Y+30, rect.Max.X-12, rect.Min.Y+50), theme.Text)
	clientui.DrawTextCentered(screen, "Type a number and press Enter", image.Rect(rect.Min.X+12, rect.Min.Y+48, rect.Max.X-12, rect.Min.Y+66), theme.TextDim)

	inputRect := image.Rect(rect.Min.X+54, rect.Min.Y+72, rect.Max.X-54, rect.Min.Y+98)
	clientui.DrawInset(screen, inputRect, theme, true)
	value := picker.Input
	if value == "" {
		value = "0"
	}
	clientui.DrawTextCentered(screen, value, inputRect, theme.Text)

	confirmRect, cancelRect := g.itemAmountPickerButtonRects(rect)
	clientui.DrawButton(screen, confirmRect, theme, "Confirm", true, false)
	clientui.DrawButton(screen, cancelRect, theme, "Cancel", false, false)
	clientui.DrawTextCentered(screen, "Esc cancels", image.Rect(rect.Min.X+12, rect.Max.Y-24, confirmRect.Min.X-8, rect.Max.Y-10), theme.TextDim)
}

func (g *Game) itemAmountPickerRect() image.Rectangle {
	return overlay.CenteredRect(280, 132, g.screenW, g.screenH)
}

func (g *Game) itemAmountPickerButtonRects(rect image.Rectangle) (image.Rectangle, image.Rectangle) {
	confirmRect := image.Rect(rect.Max.X-146, rect.Max.Y-30, rect.Max.X-78, rect.Max.Y-12)
	cancelRect := image.Rect(rect.Max.X-72, rect.Max.Y-30, rect.Max.X-12, rect.Max.Y-12)
	return confirmRect, cancelRect
}
