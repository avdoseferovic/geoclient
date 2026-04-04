package main

import (
	"image"
	"strings"

	eoproto "github.com/ethanmoffat/eolib-go/v3/protocol"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/avdo/eoweb/internal/game"
	"github.com/avdo/eoweb/internal/render"
	clientui "github.com/avdo/eoweb/internal/ui"
	"github.com/avdo/eoweb/internal/ui/charselect"
	"github.com/avdo/eoweb/internal/ui/overlay"
)

func (g *Game) updateCharacterSelect() {
	if g.overlay.characterCreateOpen {
		g.updateCharacterCreate()
		return
	}
	count := len(g.client.Characters)
	if count == 0 || g.overlay.selectingCharacter {
		return
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && g.handleCharacterSelectClick() {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		g.overlay.selectedCharacter = (g.overlay.selectedCharacter - 1 + count) % count
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.overlay.selectedCharacter = (g.overlay.selectedCharacter + 1) % count
	}
	charselect.NormalizeState(count, &g.overlay.selectedCharacter, &g.overlay.rosterScroll, &g.overlay.previewDirection)
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.selectCurrentCharacter()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyN) || inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.overlay.characterCreateOpen = true
		g.connectError = ""
	}
}

func (g *Game) updateCharacterCreate() {
	form := &g.overlay.characterCreate
	if form.Submitting {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) || inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		form.Focus = (form.Focus + 1) % 5
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		form.Focus = (form.Focus + 4) % 5
	}
	if form.Focus == 0 {
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(form.Name) > 0 {
			form.Name = form.Name[:len(form.Name)-1]
		}
		for _, r := range ebiten.AppendInputChars(nil) {
			if r < 32 || r > 126 || len(form.Name) >= charselect.NameMaxLength {
				continue
			}
			form.Name = append(form.Name, r)
		}
	} else {
		delta := 0
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
			delta = -1
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
			delta = 1
		}
		if delta != 0 {
			switch form.Focus {
			case 1:
				if form.Gender == eoproto.Gender_Female {
					form.Gender = eoproto.Gender_Male
				} else {
					form.Gender = eoproto.Gender_Female
				}
			case 2:
				form.HairStyle = charselect.CycleAppearance(form.HairStyle-1, delta, charselect.MaxHairStyle) + 1
			case 3:
				form.HairColor = charselect.CycleAppearance(form.HairColor, delta, charselect.MaxHairColor)
			case 4:
				form.Skin = charselect.CycleAppearance(form.Skin, delta, charselect.MaxSkinTone)
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) && len(g.client.Characters) > 0 {
		g.overlay.characterCreateOpen = false
		g.connectError = ""
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.submitCharacterCreate()
	}
}

func (g *Game) submitCharacterCreate() {
	form := &g.overlay.characterCreate
	name := strings.TrimSpace(string(form.Name))
	if errMsg := charselect.ValidateCharacterCreate(name); errMsg != "" {
		g.connectError = errMsg
		return
	}
	g.client.PendingCharacterCreate = &game.CharacterCreateProfile{
		Name: name, Gender: form.Gender, HairStyle: form.HairStyle, HairColor: form.HairColor, Skin: form.Skin,
	}
	form.Submitting = true
	g.connectError = ""
	g.overlay.statusMessage = "Requesting character slot..."
	g.sendCharacterCreateRequest()
}

func (g *Game) drawCharacterSelectDialog(screen *ebiten.Image, theme clientui.Theme) {
	layout := charselect.DialogLayout(g.screenW, g.screenH)
	charselect.DrawSelectDialog(screen, theme, layout, g.client.Characters, g.overlay.selectedCharacter, g.overlay.rosterScroll, g.overlay.previewDirection, g.overlay.ticks, g.overlay.selectingCharacter, g.overlay.characterCreateOpen, g.overlay.characterCreate, g.connectError, g.overlay.statusMessage, nil)
	var previewEntry server.CharacterSelectionListEntry
	if g.overlay.characterCreateOpen || len(g.client.Characters) == 0 {
		previewEntry = charselect.PreviewEntry(g.overlay.characterCreate)
	} else if len(g.client.Characters) > 0 {
		idx := overlay.ClampInt(g.overlay.selectedCharacter, 0, len(g.client.Characters)-1)
		previewEntry = g.client.Characters[idx]
	}
	preview := charselect.PreviewEntity(previewEntry, g.overlay.previewDirection, g.overlay.ticks)
	previewX := float64(layout.PreviewArtRect.Min.X + layout.PreviewArtRect.Dx()/2)
	previewY := float64(layout.PreviewArtRect.Min.Y + layout.PreviewArtRect.Dy() - 20 - charselect.PreviewBobOffset(g.overlay.ticks))
	render.RenderCharacter(screen, g.gfxLoad, &preview, previewX, previewY)
}

func (g *Game) handleCharacterSelectClick() bool {
	layout := charselect.DialogLayout(g.screenW, g.screenH)
	mx, my := ebiten.CursorPosition()
	if !overlay.PointInRect(mx, my, layout.Dialog) {
		return false
	}
	if g.overlay.characterCreateOpen || len(g.client.Characters) == 0 {
		if overlay.PointInRect(mx, my, layout.JoinButtonRect) {
			g.submitCharacterCreate()
			return true
		}
		if len(g.client.Characters) > 0 && overlay.PointInRect(mx, my, layout.CreateButtonRect) {
			g.overlay.characterCreateOpen = false
			g.connectError = ""
			return true
		}
		if overlay.PointInRect(mx, my, layout.PreviewArtRect) {
			if mx < layout.PreviewArtRect.Min.X+layout.PreviewArtRect.Dx()/2 {
				g.overlay.previewDirection = (g.overlay.previewDirection + 3) % 4
			} else {
				g.overlay.previewDirection = (g.overlay.previewDirection + 1) % 4
			}
		}
		return true
	}
	if overlay.PointInRect(mx, my, layout.CreateButtonRect) {
		g.overlay.characterCreateOpen = true
		g.connectError = ""
		return true
	}
	if overlay.PointInRect(mx, my, layout.JoinButtonRect) {
		g.selectCurrentCharacter()
		return true
	}
	if overlay.PointInRect(mx, my, layout.PreviewArtRect) {
		if mx < layout.PreviewArtRect.Min.X+layout.PreviewArtRect.Dx()/2 {
			g.overlay.previewDirection = (g.overlay.previewDirection + 3) % 4
		} else {
			g.overlay.previewDirection = (g.overlay.previewDirection + 1) % 4
		}
		return true
	}
	if !overlay.PointInRect(mx, my, layout.RosterRect) {
		return true
	}
	visibleRows := charselect.VisibleRows(layout.RosterRect)
	start, end := charselect.VisibleRange(len(g.client.Characters), g.overlay.rosterScroll, visibleRows)
	for rowIndex, i := 0, start; i < end; i, rowIndex = i+1, rowIndex+1 {
		row := image.Rect(layout.RosterRect.Min.X+10, layout.RosterRect.Min.Y+charselect.RosterTopPadding+rowIndex*charselect.RosterRowHeight, layout.RosterRect.Max.X-10, layout.RosterRect.Min.Y+charselect.RosterTopPadding+48+rowIndex*charselect.RosterRowHeight)
		if overlay.PointInRect(mx, my, row) {
			g.overlay.selectedCharacter = i
			charselect.NormalizeState(len(g.client.Characters), &g.overlay.selectedCharacter, &g.overlay.rosterScroll, &g.overlay.previewDirection)
			return true
		}
	}
	return true
}

func (g *Game) selectCurrentCharacter() {
	if g.overlay.selectingCharacter || len(g.client.Characters) == 0 {
		return
	}
	charID := g.client.Characters[g.overlay.selectedCharacter].Id
	g.overlay.selectingCharacter = true
	g.overlay.statusMessage = "Entering the world..."
	g.sendSelectCharacter(charID)
}

func (g *Game) resetCharacterCreateForm() {
	g.overlay.characterCreate = charselect.NewCreateForm()
}
