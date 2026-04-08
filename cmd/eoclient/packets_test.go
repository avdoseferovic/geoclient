package main

import (
	"errors"
	"slices"
	"testing"

	"github.com/ethanmoffat/eolib-go/v3/data"
	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"
	"github.com/ethanmoffat/eolib-go/v3/protocol/pub"

	"github.com/avdo/eoweb/internal/game"
	internalnet "github.com/avdo/eoweb/internal/net"
	"github.com/avdo/eoweb/internal/pubdata"
)

type fakePacketConn struct {
	writes [][]byte
}

func (f *fakePacketConn) ReadPacket() ([]byte, error) { return nil, errors.New("unused") }
func (f *fakePacketConn) Close() error                { return nil }
func (f *fakePacketConn) WritePacket(buf []byte) error {
	f.writes = append(f.writes, slices.Clone(buf))
	return nil
}

func TestSendUnequipItemUsesRelativeSubLoc(t *testing.T) {
	tests := []struct {
		name         string
		absoluteSlot int
		wantSubLoc   int
	}{
		{name: "single slot uses zero", absoluteSlot: 7, wantSubLoc: 0},
		{name: "second ring uses one", absoluteSlot: 11, wantSubLoc: 1},
		{name: "second armlet uses one", absoluteSlot: 13, wantSubLoc: 1},
		{name: "second bracer uses one", absoluteSlot: 15, wantSubLoc: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &fakePacketConn{}
			bus := internalnet.NewPacketBus(conn)
			c := game.NewClient()
			c.SetBus(bus)

			g := &Game{client: c}
			g.sendUnequipItem(321, tt.absoluteSlot)

			if len(conn.writes) != 1 {
				t.Fatalf("writes = %d, want 1", len(conn.writes))
			}
			buf := conn.writes[0]
			if got := eonet.PacketAction(buf[2]); got != eonet.PacketAction_Remove {
				t.Fatalf("action = %v, want %v", got, eonet.PacketAction_Remove)
			}
			if got := eonet.PacketFamily(buf[3]); got != eonet.PacketFamily_Paperdoll {
				t.Fatalf("family = %v, want %v", got, eonet.PacketFamily_Paperdoll)
			}

			var sent client.PaperdollRemoveClientPacket
			if err := sent.Deserialize(data.NewEoReader(buf[5:])); err != nil {
				t.Fatalf("deserialize sent packet: %v", err)
			}
			if sent.ItemId != 321 {
				t.Fatalf("item id = %d, want 321", sent.ItemId)
			}
			if sent.SubLoc != tt.wantSubLoc {
				t.Fatalf("subLoc = %d, want %d", sent.SubLoc, tt.wantSubLoc)
			}
		})
	}
}

func TestSendDropItemUsesExplicitCoords(t *testing.T) {
	conn := &fakePacketConn{}
	bus := internalnet.NewPacketBus(conn)
	c := game.NewClient()
	c.SetBus(bus)

	g := &Game{client: c}
	g.sendDropItem(321, 1, 10, 12)

	if len(conn.writes) != 1 {
		t.Fatalf("writes = %d, want 1", len(conn.writes))
	}
	buf := conn.writes[0]
	if got := eonet.PacketAction(buf[2]); got != eonet.PacketAction_Drop {
		t.Fatalf("action = %v, want %v", got, eonet.PacketAction_Drop)
	}
	if got := eonet.PacketFamily(buf[3]); got != eonet.PacketFamily_Item {
		t.Fatalf("family = %v, want %v", got, eonet.PacketFamily_Item)
	}

	var sent client.ItemDropClientPacket
	if err := sent.Deserialize(data.NewEoReader(buf[5:])); err != nil {
		t.Fatalf("deserialize sent packet: %v", err)
	}
	if sent.Item.Id != 321 || sent.Item.Amount != 1 {
		t.Fatalf("drop item = %#v", sent.Item)
	}
	if sent.Coords.X != 11 || sent.Coords.Y != 13 {
		t.Fatalf("drop coords = (%d, %d), want (11, 13)", sent.Coords.X, sent.Coords.Y)
	}
}

func TestSendShopPacketsUseSessionID(t *testing.T) {
	conn := &fakePacketConn{}
	bus := internalnet.NewPacketBus(conn)
	c := game.NewClient()
	c.SetBus(bus)
	c.SessionID = 77

	g := &Game{
		client: c,
		npcDB:  pubdata.NewNPCDB(pubdata.NPCDef{ID: 15, Name: "Shop Bob", Type: pub.Npc_Shop}),
	}
	g.sendNpcInteract(8, 15) // NPC index 8, ENF ID 15 (Shop Bob, type Shop)
	g.sendShopBuy(5, 2)
	g.sendShopSell(6, 3)
	g.sendShopCraft(9)

	if len(conn.writes) != 4 {
		t.Fatalf("writes = %d, want 4", len(conn.writes))
	}

	var open client.ShopOpenClientPacket
	if err := open.Deserialize(data.NewEoReader(conn.writes[0][5:])); err != nil {
		t.Fatalf("deserialize open packet: %v", err)
	}
	if open.NpcIndex != 8 {
		t.Fatalf("open npc index = %d, want 8", open.NpcIndex)
	}

	var buy client.ShopBuyClientPacket
	if err := buy.Deserialize(data.NewEoReader(conn.writes[1][5:])); err != nil {
		t.Fatalf("deserialize buy packet: %v", err)
	}
	if buy.BuyItem.Id != 5 || buy.BuyItem.Amount != 2 || buy.SessionId != 77 {
		t.Fatalf("buy packet = %#v", buy)
	}

	var sell client.ShopSellClientPacket
	if err := sell.Deserialize(data.NewEoReader(conn.writes[2][5:])); err != nil {
		t.Fatalf("deserialize sell packet: %v", err)
	}
	if sell.SellItem.Id != 6 || sell.SellItem.Amount != 3 || sell.SessionId != 77 {
		t.Fatalf("sell packet = %#v", sell)
	}

	var craft client.ShopCreateClientPacket
	if err := craft.Deserialize(data.NewEoReader(conn.writes[3][5:])); err != nil {
		t.Fatalf("deserialize craft packet: %v", err)
	}
	if craft.CraftItemId != 9 || craft.SessionId != 77 {
		t.Fatalf("craft packet = %#v", craft)
	}
}
