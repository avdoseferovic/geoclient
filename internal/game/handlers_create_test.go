package game

import (
	"errors"
	"slices"
	"testing"

	"github.com/ethanmoffat/eolib-go/v3/data"
	eoproto "github.com/ethanmoffat/eolib-go/v3/protocol"
	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"

	internalnet "github.com/avdo/eoweb/internal/net"
)

type fakePacketConn struct {
	writes [][]byte
}

func (f *fakePacketConn) ReadPacket() ([]byte, error) { return nil, errors.New("unused") }
func (f *fakePacketConn) Close() error                { return nil }
func (f *fakePacketConn) WritePacket(buf []byte) error {
	cp := slices.Clone(buf)
	f.writes = append(f.writes, cp)
	return nil
}

func packetReader(t *testing.T, pkt eonet.Packet) *data.EoReader {
	t.Helper()
	writer := data.NewEoWriter()
	if err := pkt.Serialize(writer); err != nil {
		t.Fatalf("serialize packet: %v", err)
	}
	return data.NewEoReader(writer.Array())
}

func TestHandleAccountReplyCreatedEmitsEvent(t *testing.T) {
	c := NewClient()
	c.PendingAccountCreate = &AccountCreateProfile{FullName: "Test"}

	err := handleAccountReply(c, packetReader(t, &server.AccountReplyServerPacket{
		ReplyCode:     server.AccountReply_Created,
		ReplyCodeData: &server.AccountReplyReplyCodeDataCreated{},
	}))
	if err != nil {
		t.Fatalf("handleAccountReply returned error: %v", err)
	}

	evt := <-c.Events
	if evt.Type != EventAccountCreated {
		t.Fatalf("event type = %v, want %v", evt.Type, EventAccountCreated)
	}
	if c.PendingAccountCreate != nil {
		t.Fatal("pending account create was not cleared")
	}
}

func TestHandleAccountReplyDefaultSendsCreatePacket(t *testing.T) {
	conn := &fakePacketConn{}
	bus := internalnet.NewPacketBus(conn)
	c := NewClient()
	c.SetBus(bus)
	c.Username = "wanderer"
	c.Password = "secret"
	c.PendingAccountCreate = &AccountCreateProfile{
		FullName: "Wanderer",
		Location: "Unknown",
		Email:    "wanderer@geoclient.local",
	}

	err := handleAccountReply(c, packetReader(t, &server.AccountReplyServerPacket{
		ReplyCode:     server.AccountReply(42),
		ReplyCodeData: &server.AccountReplyReplyCodeDataDefault{SequenceStart: 7},
	}))
	if err != nil {
		t.Fatalf("handleAccountReply returned error: %v", err)
	}
	if c.SessionID != 42 {
		t.Fatalf("session id = %d, want 42", c.SessionID)
	}
	if len(conn.writes) != 1 {
		t.Fatalf("writes = %d, want 1", len(conn.writes))
	}
	buf := conn.writes[0]
	if got := eonet.PacketAction(buf[2]); got != eonet.PacketAction_Create {
		t.Fatalf("action = %v, want %v", got, eonet.PacketAction_Create)
	}
	if got := eonet.PacketFamily(buf[3]); got != eonet.PacketFamily_Account {
		t.Fatalf("family = %v, want %v", got, eonet.PacketFamily_Account)
	}
	if got := data.DecodeNumber([]byte{buf[4]}); got != 7 {
		t.Fatalf("sequence byte = %d, want 7", got)
	}

	var sent client.AccountCreateClientPacket
	if err := sent.Deserialize(data.NewEoReader(buf[5:])); err != nil {
		t.Fatalf("deserialize sent packet: %v", err)
	}
	if sent.SessionId != 42 || sent.Username != "wanderer" || sent.Password != "secret" {
		t.Fatalf("sent packet = %#v", sent)
	}
	if sent.FullName != "Wanderer" || sent.Email != "wanderer@geoclient.local" {
		t.Fatalf("sent profile = %#v", sent)
	}
}

func TestHandleCharacterReplyDefaultSendsCreatePacket(t *testing.T) {
	conn := &fakePacketConn{}
	bus := internalnet.NewPacketBus(conn)
	c := NewClient()
	c.SetBus(bus)
	c.PendingCharacterCreate = &CharacterCreateProfile{
		Name:      "Arielle",
		Gender:    eoproto.Gender_Female,
		HairStyle: 3,
		HairColor: 2,
		Skin:      1,
	}

	err := handleCharacterReply(c, packetReader(t, &server.CharacterReplyServerPacket{
		ReplyCode:     server.CharacterReply(55),
		ReplyCodeData: &server.CharacterReplyReplyCodeDataDefault{},
	}))
	if err != nil {
		t.Fatalf("handleCharacterReply returned error: %v", err)
	}
	if c.SessionID != 55 {
		t.Fatalf("session id = %d, want 55", c.SessionID)
	}
	if len(conn.writes) != 1 {
		t.Fatalf("writes = %d, want 1", len(conn.writes))
	}
	buf := conn.writes[0]
	if got := eonet.PacketAction(buf[2]); got != eonet.PacketAction_Create {
		t.Fatalf("action = %v, want %v", got, eonet.PacketAction_Create)
	}
	if got := eonet.PacketFamily(buf[3]); got != eonet.PacketFamily_Character {
		t.Fatalf("family = %v, want %v", got, eonet.PacketFamily_Character)
	}

	var sent client.CharacterCreateClientPacket
	if err := sent.Deserialize(data.NewEoReader(buf[5:])); err != nil {
		t.Fatalf("deserialize sent packet: %v", err)
	}
	if sent.SessionId != 55 || sent.Name != "Arielle" {
		t.Fatalf("sent packet = %#v", sent)
	}
	if sent.Gender != eoproto.Gender_Female || sent.HairStyle != 3 || sent.HairColor != 2 || sent.Skin != 1 {
		t.Fatalf("appearance = %#v", sent)
	}
}

func TestHandleCharacterReplyOkUpdatesRoster(t *testing.T) {
	c := NewClient()
	c.PendingCharacterCreate = &CharacterCreateProfile{Name: "Arielle"}
	characters := []server.CharacterSelectionListEntry{{Id: 5, Name: "Arielle", Level: 1}}

	err := handleCharacterReply(c, packetReader(t, &server.CharacterReplyServerPacket{
		ReplyCode:     server.CharacterReply_Ok,
		ReplyCodeData: &server.CharacterReplyReplyCodeDataOk{Characters: characters},
	}))
	if err != nil {
		t.Fatalf("handleCharacterReply returned error: %v", err)
	}
	evt := <-c.Events
	if evt.Type != EventCharacterList {
		t.Fatalf("event type = %v, want %v", evt.Type, EventCharacterList)
	}
	if len(c.Characters) != 1 || c.Characters[0].Name != "Arielle" {
		t.Fatalf("characters = %#v", c.Characters)
	}
	if c.PendingCharacterCreate != nil {
		t.Fatal("pending character create was not cleared")
	}
}
