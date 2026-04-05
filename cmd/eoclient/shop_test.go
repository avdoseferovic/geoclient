package main

import (
	"testing"

	"github.com/ethanmoffat/eolib-go/v3/data"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"
	"github.com/ethanmoffat/eolib-go/v3/protocol/pub"

	"github.com/avdo/eoweb/internal/game"
	internalnet "github.com/avdo/eoweb/internal/net"
	"github.com/avdo/eoweb/internal/pubdata"
)

func TestTryOpenShopAtAdjacentVendorSendsPacket(t *testing.T) {
	conn := &fakePacketConn{}
	bus := internalnet.NewPacketBus(conn)
	c := game.NewClient()
	c.SetBus(bus)
	c.Character.X = 10
	c.Character.Y = 10
	c.NearbyNpcs = []game.NearbyNPC{{Index: 4, ID: 15, X: 11, Y: 10}}

	g := &Game{
		client: c,
		npcDB:  pubdata.NewNPCDB(pubdata.NPCDef{ID: 15, Name: "Vendor", Type: pub.Npc_Shop}),
	}

	if !g.tryOpenShopAt(11, 10) {
		t.Fatal("tryOpenShopAt() = false, want true")
	}
	if len(conn.writes) != 1 {
		t.Fatalf("writes = %d, want 1", len(conn.writes))
	}

	var sent client.ShopOpenClientPacket
	if err := sent.Deserialize(packetPayload(t, conn.writes[0])); err != nil {
		t.Fatalf("deserialize sent packet: %v", err)
	}
	if sent.NpcIndex != 4 {
		t.Fatalf("npc index = %d, want 4", sent.NpcIndex)
	}
}

func TestTryOpenShopAtNonVendorDoesNothing(t *testing.T) {
	conn := &fakePacketConn{}
	bus := internalnet.NewPacketBus(conn)
	c := game.NewClient()
	c.SetBus(bus)
	c.Character.X = 10
	c.Character.Y = 10
	c.NearbyNpcs = []game.NearbyNPC{{Index: 4, ID: 15, X: 11, Y: 10}}

	g := &Game{
		client: c,
		npcDB:  pubdata.NewNPCDB(pubdata.NPCDef{ID: 15, Name: "Wolf", Type: pub.Npc_Aggressive}),
	}

	if g.tryOpenShopAt(11, 10) {
		t.Fatal("tryOpenShopAt() = true, want false")
	}
	if len(conn.writes) != 0 {
		t.Fatalf("writes = %d, want 0", len(conn.writes))
	}
}

func TestClickedNpcNeverAttacks(t *testing.T) {
	g := &Game{
		client: &game.Client{
			Character: game.Character{X: 10, Y: 10},
			NearbyNpcs: []game.NearbyNPC{
				{Index: 2, ID: 7, X: 11, Y: 10},
			},
		},
	}

	if g.isAttackTargetTile(11, 10) {
		t.Fatal("isAttackTargetTile() = true for npc tile, want false")
	}
}

func packetPayload(t *testing.T, buf []byte) *data.EoReader {
	t.Helper()
	return data.NewEoReader(buf[5:])
}
