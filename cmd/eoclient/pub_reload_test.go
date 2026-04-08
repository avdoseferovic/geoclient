package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethanmoffat/eolib-go/v3/data"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"
	"github.com/ethanmoffat/eolib-go/v3/protocol/pub"

	"github.com/avdo/eoweb/internal/assets"
	"github.com/avdo/eoweb/internal/game"
	internalnet "github.com/avdo/eoweb/internal/net"
	"github.com/avdo/eoweb/internal/pubdata"
)

func TestHandleEventFileUpdatedReloadsNpcMetadata(t *testing.T) {
	tempDir := t.TempDir()
	pubPath := filepath.Join(tempDir, "pub", "dtn001.enf")
	if err := os.MkdirAll(filepath.Dir(pubPath), 0o755); err != nil {
		t.Fatalf("mkdir pub dir: %v", err)
	}
	if err := os.WriteFile(pubPath, testENFBytes(t, 15, pub.Npc_Friendly), 0o644); err != nil {
		t.Fatalf("write initial enf: %v", err)
	}

	npcDB, err := pubdata.LoadNPCDB(pubPath)
	if err != nil {
		t.Fatalf("load initial npc db: %v", err)
	}

	conn := &fakePacketConn{}
	bus := internalnet.NewPacketBus(conn)
	cl := game.NewClient()
	cl.SetBus(bus)
	cl.AssetReader = assets.NewOSReader()
	cl.NpcPubPath = pubPath

	g := &Game{
		client: cl,
		npcDB:  npcDB,
	}

	g.sendNpcInteract(4, 15)
	if len(conn.writes) != 0 {
		t.Fatalf("writes before reload = %d, want 0", len(conn.writes))
	}

	if err := os.WriteFile(pubPath, testENFBytes(t, 15, pub.Npc_Shop), 0o644); err != nil {
		t.Fatalf("write updated enf: %v", err)
	}

	g.handleEvent(game.Event{Type: game.EventFileUpdated, Data: client.File_Enf})
	g.sendNpcInteract(4, 15)

	if len(conn.writes) != 1 {
		t.Fatalf("writes after reload = %d, want 1", len(conn.writes))
	}

	var sent client.ShopOpenClientPacket
	if err := sent.Deserialize(data.NewEoReader(conn.writes[0][5:])); err != nil {
		t.Fatalf("deserialize shop open packet: %v", err)
	}
	if sent.NpcIndex != 4 {
		t.Fatalf("npc index = %d, want 4", sent.NpcIndex)
	}
}

func testENFBytes(t *testing.T, npcID int, npcType pub.NpcType) []byte {
	t.Helper()

	npcs := make([]pub.EnfRecord, npcID)
	npcs[npcID-1] = pub.EnfRecord{
		Name:       "Vendor",
		GraphicId:  1,
		Type:       npcType,
		BehaviorId: 7,
		Hp:         10,
	}

	enf := pub.Enf{
		Rid:            []int{1, 2},
		TotalNpcsCount: len(npcs),
		Version:        1,
		Npcs:           npcs,
	}

	writer := data.NewEoWriter()
	if err := enf.Serialize(writer); err != nil {
		t.Fatalf("serialize enf: %v", err)
	}
	return writer.Array()
}
