package main

import (
	"log/slog"

	"github.com/avdo/eoweb/internal/pubdata"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"
)

func (g *Game) reloadPubMetadata(fileType client.FileType) {
	if g == nil || g.client == nil || g.client.AssetReader == nil {
		return
	}

	switch fileType {
	case client.File_Eif:
		itemDB, err := pubdata.LoadItemDBFromReader(g.client.AssetReader, g.client.ItemPubPath)
		if err != nil {
			slog.Warn("item metadata reload failed", "path", g.client.ItemPubPath, "err", err)
			return
		}
		g.itemDB = itemDB
		g.client.ItemTypeFunc = func(id int) int {
			if rec, ok := itemDB.Get(id); ok {
				return int(rec.Type)
			}
			return 0
		}
		slog.Info("item metadata reloaded", "path", g.client.ItemPubPath)
	case client.File_Enf:
		npcDB, err := pubdata.LoadNPCDBFromReader(g.client.AssetReader, g.client.NpcPubPath)
		if err != nil {
			slog.Warn("npc metadata reload failed", "path", g.client.NpcPubPath, "err", err)
			return
		}
		g.npcDB = npcDB
		slog.Info("npc metadata reloaded", "path", g.client.NpcPubPath)
	}
}
