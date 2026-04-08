package main

import (
	"testing"

	"github.com/avdo/eoweb/internal/game"
	"github.com/avdo/eoweb/internal/render"
)

func TestPlayerContextMenuTargetUsesFullCharacterHoverRect(t *testing.T) {
	g := &Game{
		screenW:     1024,
		screenH:     768,
		client:      game.NewClient(),
		mapRenderer: &render.MapRenderer{},
	}

	snapshot := game.UISnapshot{
		PlayerID:  1,
		Character: game.Character{X: 10, Y: 10},
	}
	ch := render.CharacterEntity{
		PlayerID:  2,
		Name:      "Target",
		X:         11,
		Y:         10,
		Direction: 0,
		Gender:    1,
	}
	g.mapRenderer.Characters = []render.CharacterEntity{ch}

	camSX, camSY := g.currentCameraScreenPosition(snapshot)
	rect := characterHoverRect(ch, camSX, camSY, float64(g.screenW)/2, float64(g.screenH)/2)

	mx := (rect.Min.X + rect.Max.X) / 2
	my := rect.Min.Y + 4

	target, ok := g.playerContextMenuTarget(snapshot, mx, my)
	if !ok {
		t.Fatal("playerContextMenuTarget() = false, want true")
	}
	if target.PlayerID != 2 {
		t.Fatalf("target.PlayerID = %d, want 2", target.PlayerID)
	}
}
