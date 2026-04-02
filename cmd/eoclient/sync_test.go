package main

import (
	"path/filepath"
	"testing"

	"github.com/avdo/eoweb/internal/game"
	"github.com/avdo/eoweb/internal/pubdata"
)

func TestGroundItemGraphicIDUsesItemMetadata(t *testing.T) {
	itemDB, err := pubdata.LoadItemDB(filepath.Join("..", "..", "pub", "dat001.eif"))
	if err != nil {
		t.Fatalf("LoadItemDB: %v", err)
	}

	g := &Game{
		itemDB: itemDB,
	}

	item := game.NearbyItem{ID: 100, GraphicID: 100, Amount: 1}
	if got := g.groundItemGraphicID(item); got != 13 {
		t.Fatalf("groundItemGraphicID(normal) = %d, want 13", got)
	}
}

func TestGroundItemGraphicIDFallsBackWithoutMetadata(t *testing.T) {
	g := &Game{}

	item := game.NearbyItem{ID: 100, GraphicID: 44, Amount: 1}
	if got := g.groundItemGraphicID(item); got != 44 {
		t.Fatalf("groundItemGraphicID(fallback) = %d, want 44", got)
	}
}
