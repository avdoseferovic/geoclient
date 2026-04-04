package main

import (
	"errors"
	"slices"
	"testing"

	"github.com/ethanmoffat/eolib-go/v3/data"
	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"

	"github.com/avdo/eoweb/internal/game"
	internalnet "github.com/avdo/eoweb/internal/net"
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
