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

	record, ok := itemDB.Get(100)
	if !ok {
		t.Fatal("itemDB.Get(100) missing")
	}
	want := pubdata.GraphicResourceID(100, record.GraphicID, 1)
	item := game.NearbyItem{ID: 100, GraphicID: 100, Amount: 1}
	if got := g.groundItemGraphicID(item); got != want {
		t.Fatalf("groundItemGraphicID(normal) = %d, want %d", got, want)
	}
}

func TestGroundItemGraphicIDFallsBackWithoutMetadata(t *testing.T) {
	g := &Game{}

	item := game.NearbyItem{ID: 100, GraphicID: 44, Amount: 1}
	if got := g.groundItemGraphicID(item); got != 44 {
		t.Fatalf("groundItemGraphicID(fallback) = %d, want 44", got)
	}
}

func TestNpcGraphicIDUsesNPCMetadata(t *testing.T) {
	npcDB, err := pubdata.LoadNPCDB(filepath.Join("..", "..", "pub", "dtn001.enf"))
	if err != nil {
		t.Fatalf("LoadNPCDB: %v", err)
	}

	g := &Game{
		npcDB: npcDB,
	}

	record, ok := npcDB.Get(119)
	if !ok {
		t.Fatalf("npcDB.Get(119) missing")
	}
	if got := g.npcGraphicID(119); got != record.GraphicID {
		t.Fatalf("npcGraphicID(119) = %d, want %d", got, record.GraphicID)
	}
}

func TestNpcGraphicIDFallsBackWithoutMetadata(t *testing.T) {
	g := &Game{}

	if got := g.npcGraphicID(170); got != 170 {
		t.Fatalf("npcGraphicID(fallback) = %d, want 170", got)
	}
}
