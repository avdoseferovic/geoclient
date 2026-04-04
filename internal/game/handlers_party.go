package game

import (
	"fmt"
	"log/slog"

	"github.com/ethanmoffat/eolib-go/v3/data"
	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
)

func RegisterPartyHandlers(reg *HandlerRegistry) {
	reg.Register(eonet.PacketFamily_Party, eonet.PacketAction_Create, handlePartyCreate)
	reg.Register(eonet.PacketFamily_Party, eonet.PacketAction_Add, handlePartyAdd)
	reg.Register(eonet.PacketFamily_Party, eonet.PacketAction_Remove, handlePartyRemove)
	reg.Register(eonet.PacketFamily_Party, eonet.PacketAction_Close, handlePartyClose)
	reg.Register(eonet.PacketFamily_Party, eonet.PacketAction_List, handlePartyList)
	reg.Register(eonet.PacketFamily_Party, eonet.PacketAction_Agree, handlePartyAgree)
	reg.Register(eonet.PacketFamily_Party, eonet.PacketAction_Request, handlePartyRequest)
	reg.Register(eonet.PacketFamily_Party, eonet.PacketAction_Reply, handlePartyReply)
}

func handlePartyCreate(c *Client, reader *data.EoReader) error {
	var pkt server.PartyCreateServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize party create: %w", err)
	}

	c.mu.Lock()
	c.PartyMembers = serverPartyMembers(pkt.Members)
	c.mu.Unlock()

	slog.Info("party created", "members", len(pkt.Members))
	c.Emit(Event{Type: EventPartyUpdated})
	return nil
}

func handlePartyAdd(c *Client, reader *data.EoReader) error {
	var pkt server.PartyAddServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize party add: %w", err)
	}

	c.mu.Lock()
	c.PartyMembers = append(c.PartyMembers, PartyMember{
		PlayerID:     pkt.Member.PlayerId,
		Name:         pkt.Member.Name,
		Level:        pkt.Member.Level,
		HpPercentage: pkt.Member.HpPercentage,
		Leader:       pkt.Member.Leader,
	})
	c.mu.Unlock()

	slog.Info("party member joined", "name", pkt.Member.Name)
	emitChat(c, ChatChannelSystem, fmt.Sprintf("%s joined the party.", pkt.Member.Name))
	c.Emit(Event{Type: EventPartyUpdated})
	return nil
}

func handlePartyRemove(c *Client, reader *data.EoReader) error {
	var pkt server.PartyRemoveServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize party remove: %w", err)
	}

	c.mu.Lock()
	name := ""
	for i, m := range c.PartyMembers {
		if m.PlayerID == pkt.PlayerId {
			name = m.Name
			c.PartyMembers = append(c.PartyMembers[:i], c.PartyMembers[i+1:]...)
			break
		}
	}
	if len(c.PartyMembers) <= 1 {
		c.PartyMembers = nil
	}
	c.mu.Unlock()

	if name != "" {
		emitChat(c, ChatChannelSystem, fmt.Sprintf("%s left the party.", name))
	}
	slog.Info("party member removed", "playerID", pkt.PlayerId)
	c.Emit(Event{Type: EventPartyUpdated})
	return nil
}

func handlePartyClose(c *Client, reader *data.EoReader) error {
	c.mu.Lock()
	c.PartyMembers = nil
	c.mu.Unlock()

	emitChat(c, ChatChannelSystem, "Party disbanded.")
	slog.Info("party disbanded")
	c.Emit(Event{Type: EventPartyUpdated})
	return nil
}

func handlePartyList(c *Client, reader *data.EoReader) error {
	var pkt server.PartyListServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize party list: %w", err)
	}

	c.mu.Lock()
	c.PartyMembers = serverPartyMembers(pkt.Members)
	c.mu.Unlock()

	c.Emit(Event{Type: EventPartyUpdated})
	return nil
}

func handlePartyAgree(c *Client, reader *data.EoReader) error {
	var pkt server.PartyAgreeServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize party agree: %w", err)
	}

	c.mu.Lock()
	for i, m := range c.PartyMembers {
		if m.PlayerID == pkt.PlayerId {
			c.PartyMembers[i].HpPercentage = pkt.HpPercentage
			break
		}
	}
	c.mu.Unlock()

	c.Emit(Event{Type: EventPartyUpdated})
	return nil
}

func handlePartyRequest(c *Client, reader *data.EoReader) error {
	var pkt server.PartyRequestServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize party request: %w", err)
	}

	c.mu.Lock()
	c.PendingPartyInvite = &PendingPartyInvite{
		PlayerID:    pkt.InviterPlayerId,
		PlayerName:  pkt.PlayerName,
		RequestType: int(pkt.RequestType),
	}
	c.mu.Unlock()

	emitChat(c, ChatChannelSystem, fmt.Sprintf("%s wants to party with you. Open party panel to accept.", pkt.PlayerName))
	slog.Info("party invite received", "from", pkt.PlayerName, "type", pkt.RequestType)
	c.Emit(Event{Type: EventPartyUpdated})
	return nil
}

func handlePartyReply(c *Client, reader *data.EoReader) error {
	var pkt server.PartyReplyServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize party reply: %w", err)
	}

	switch pkt.ReplyCode {
	case 0: // AlreadyInAnotherParty
		emitChat(c, ChatChannelSystem, "That player is already in another party.")
	case 1: // AlreadyInYourParty
		emitChat(c, ChatChannelSystem, "That player is already in your party.")
	case 2: // PartyIsFull
		emitChat(c, ChatChannelSystem, "The party is full.")
	}
	return nil
}

func serverPartyMembers(members []server.PartyMember) []PartyMember {
	result := make([]PartyMember, len(members))
	for i, m := range members {
		result[i] = PartyMember{
			PlayerID:     m.PlayerId,
			Name:         m.Name,
			Level:        m.Level,
			HpPercentage: m.HpPercentage,
			Leader:       m.Leader,
		}
	}
	return result
}
