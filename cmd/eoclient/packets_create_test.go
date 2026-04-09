package main

import (
	"testing"

	"github.com/ethanmoffat/eolib-go/v3/data"
	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"

	"github.com/avdoseferovic/geoclient/internal/game"
	internalnet "github.com/avdoseferovic/geoclient/internal/net"
)

func TestSendCharacterCreateRequestSendsNewMarker(t *testing.T) {
	conn := &fakePacketConn{}
	bus := internalnet.NewPacketBus(conn)
	c := game.NewClient()
	c.SetBus(bus)
	g := &Game{client: c}

	g.sendCharacterCreateRequest()

	if len(conn.writes) != 1 {
		t.Fatalf("writes = %d, want 1", len(conn.writes))
	}
	buf := conn.writes[0]
	if got := eonet.PacketAction(buf[2]); got != eonet.PacketAction_Request {
		t.Fatalf("action = %v, want %v", got, eonet.PacketAction_Request)
	}
	if got := eonet.PacketFamily(buf[3]); got != eonet.PacketFamily_Character {
		t.Fatalf("family = %v, want %v", got, eonet.PacketFamily_Character)
	}

	var sent client.CharacterRequestClientPacket
	if err := sent.Deserialize(data.NewEoReader(buf[5:])); err != nil {
		t.Fatalf("deserialize sent packet: %v", err)
	}
	if sent.RequestString != "NEW" {
		t.Fatalf("RequestString = %q, want NEW", sent.RequestString)
	}
}
