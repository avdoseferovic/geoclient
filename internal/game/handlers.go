package game

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/ethanmoffat/eolib-go/v3/data"
	"github.com/ethanmoffat/eolib-go/v3/encrypt"
	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
)

type HandlerFunc func(c *Client, reader *data.EoReader) error

type HandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[eonet.PacketFamily]map[eonet.PacketAction]HandlerFunc
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[eonet.PacketFamily]map[eonet.PacketAction]HandlerFunc),
	}
}

func (r *HandlerRegistry) Register(family eonet.PacketFamily, action eonet.PacketAction, handler HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.handlers[family]; !ok {
		r.handlers[family] = make(map[eonet.PacketAction]HandlerFunc)
	}
	r.handlers[family][action] = handler
}

func (r *HandlerRegistry) Dispatch(family eonet.PacketFamily, action eonet.PacketAction, c *Client, reader *data.EoReader) error {
	r.mu.RLock()
	familyHandlers, ok := r.handlers[family]
	if !ok {
		r.mu.RUnlock()
		slog.Debug("unhandled packet family", "family", int(family), "action", int(action))
		return nil
	}
	handler, ok := familyHandlers[action]
	r.mu.RUnlock()
	if !ok {
		slog.Debug("unhandled packet action", "family", int(family), "action", int(action))
		return nil
	}
	return handler(c, reader)
}

// RegisterAllHandlers registers all packet handlers.
func RegisterAllHandlers(reg *HandlerRegistry) {
	reg.Register(eonet.PacketFamily_Init, eonet.PacketAction_Init, handleInitInit)
	reg.Register(eonet.PacketFamily_Connection, eonet.PacketAction_Player, handleConnectionPlayer)
	reg.Register(eonet.PacketFamily_Account, eonet.PacketAction_Reply, handleAccountReply)
	reg.Register(eonet.PacketFamily_Login, eonet.PacketAction_Reply, handleLoginReply)
	reg.Register(eonet.PacketFamily_Character, eonet.PacketAction_Reply, handleCharacterReply)
	reg.Register(eonet.PacketFamily_Welcome, eonet.PacketAction_Reply, handleWelcomeReply)
	reg.Register(eonet.PacketFamily_Talk, eonet.PacketAction_Player, handleTalkPlayer)
	reg.Register(eonet.PacketFamily_Talk, eonet.PacketAction_Open, handleTalkOpen)
	reg.Register(eonet.PacketFamily_Talk, eonet.PacketAction_Msg, handleTalkMsg)
	reg.Register(eonet.PacketFamily_Talk, eonet.PacketAction_Tell, handleTalkTell)
	reg.Register(eonet.PacketFamily_Talk, eonet.PacketAction_Reply, handleTalkReply)
	reg.Register(eonet.PacketFamily_Talk, eonet.PacketAction_Admin, handleTalkAdmin)
	reg.Register(eonet.PacketFamily_Talk, eonet.PacketAction_Announce, handleTalkAnnounce)
	// Entity handlers registered separately
	RegisterEntityHandlers(reg)
	RegisterStatSkillHandlers(reg)
	RegisterDoorHandlers(reg)
	RegisterPlayersHandlers(reg)
	RegisterChestHandlers(reg)
	RegisterPartyHandlers(reg)
	RegisterTradeHandlers(reg)
	RegisterShopHandlers(reg)
}

func emitChat(c *Client, channel ChatChannel, text string) {
	c.Emit(Event{Type: EventChat, Data: ChatMessage{Channel: channel, Text: text}, Message: text})
}

func emitChatAllChannels(c *Client, text string) {
	for _, channel := range []ChatChannel{
		ChatChannelMap,
		ChatChannelGroup,
		ChatChannelGlobal,
		ChatChannelSystem,
	} {
		emitChat(c, channel, text)
	}
}

func chatCharacterName(c *Client, playerID int) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if playerID == c.PlayerID && c.Character.Name != "" {
		return c.Character.Name
	}
	for _, ch := range c.NearbyChars {
		if ch.PlayerID == playerID && ch.Name != "" {
			return ch.Name
		}
	}
	return fmt.Sprintf("Player %03d", playerID)
}

func handleInitInit(c *Client, reader *data.EoReader) error {
	var pkt server.InitInitServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize init: %w", err)
	}

	switch pkt.ReplyCode {
	case server.InitReply_Ok:
		return handleInitOk(c, pkt.ReplyCodeData.(*server.InitInitReplyCodeDataOk))
	case server.InitReply_OutOfDate:
		d := pkt.ReplyCodeData.(*server.InitInitReplyCodeDataOutOfDate)
		c.Version = d.Version
		slog.Info("server requested version update", "version", d.Version)
		return nil
	case server.InitReply_PlayersList:
		return handleInitPlayersList(c, pkt.ReplyCodeData.(*server.InitInitReplyCodeDataPlayersList))
	case server.InitReply_Banned:
		c.Emit(Event{Type: EventError, Message: "You are banned from this server"})
		return nil
	default:
		slog.Info("unhandled init reply", "code", pkt.ReplyCode)
		return nil
	}
}

func handleInitOk(c *Client, d *server.InitInitReplyCodeDataOk) error {
	// Verify server challenge
	expected := encrypt.ServerVerificationHash(c.Challenge)
	if d.ChallengeResponse != expected {
		c.Emit(Event{Type: EventError, Message: "Server verification failed"})
		c.Disconnect()
		return fmt.Errorf("server verification failed: got %d, expected %d", d.ChallengeResponse, expected)
	}

	c.PlayerID = d.PlayerId

	// Set encryption multiples
	bus := c.GetBus()
	if bus == nil {
		return fmt.Errorf("connection bus missing during init")
	}
	bus.SetEncryption(d.ClientEncryptionMultiple, d.ServerEncryptionMultiple)

	// Set sequence from init values: value = seq1*7 + seq2 - 13
	seqStart := d.Seq1*7 + d.Seq2 - 13
	bus.Sequencer.Reset(seqStart)
	bus.Sequencer.NextSequence() // consume slot 0

	slog.Info("init complete", "playerID", c.PlayerID, "seqStart", seqStart)

	// Send connection accept
	if err := bus.SendSequenced(&client.ConnectionAcceptClientPacket{
		PlayerId:                 c.PlayerID,
		ClientEncryptionMultiple: d.ClientEncryptionMultiple,
		ServerEncryptionMultiple: d.ServerEncryptionMultiple,
	}); err != nil {
		return fmt.Errorf("connection setup failed while sending accept: %w", err)
	}

	c.SetState(StateConnected)
	return nil
}

func handleConnectionPlayer(c *Client, reader *data.EoReader) error {
	var pkt server.ConnectionPlayerServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize ping: %w", err)
	}

	// Update sequence start: value = seq1 - seq2
	bus := c.GetBus()
	if bus == nil {
		return fmt.Errorf("connection bus missing during ping")
	}
	bus.Sequencer.SetStart(pkt.Seq1 - pkt.Seq2)

	// Reply with pong
	if err := bus.SendSequenced(&client.ConnectionPingClientPacket{}); err != nil {
		return fmt.Errorf("connection keepalive failed while sending ping reply: %w", err)
	}
	return nil
}

func handleAccountReply(c *Client, reader *data.EoReader) error {
	var pkt server.AccountReplyServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize account reply: %w", err)
	}

	switch pkt.ReplyCode {
	case server.AccountReply_Exists:
		c.Emit(Event{Type: EventError, Message: "Account name already exists"})
	case server.AccountReply_NotApproved:
		c.Emit(Event{Type: EventError, Message: "Account name was not approved"})
	case server.AccountReply_ChangeFailed:
		c.Emit(Event{Type: EventError, Message: "Password change failed"})
	case server.AccountReply_Changed:
		c.Emit(Event{Type: EventError, Message: "Password changed"})
	case server.AccountReply_RequestDenied:
		c.Emit(Event{Type: EventError, Message: "Account request denied"})
	case server.AccountReply_Created:
		c.PendingAccountCreate = nil
		c.Emit(Event{Type: EventAccountCreated, Message: "Account created"})
	default:
		pending := c.PendingAccountCreate
		bus := c.GetBus()
		d, ok := pkt.ReplyCodeData.(*server.AccountReplyReplyCodeDataDefault)
		if pending == nil || bus == nil || !ok {
			return nil
		}

		c.SessionID = int(pkt.ReplyCode)
		bus.Sequencer.Reset(d.SequenceStart)
		if err := bus.SendSequenced(&client.AccountCreateClientPacket{
			SessionId: c.SessionID,
			Username:  c.Username,
			Password:  c.Password,
			FullName:  pending.FullName,
			Location:  pending.Location,
			Email:     pending.Email,
			Computer:  "geoclient",
			Hdid:      "1111111111",
		}); err != nil {
			return fmt.Errorf("account create confirm failed: %w", err)
		}
	}
	return nil
}

func handleLoginReply(c *Client, reader *data.EoReader) error {
	var pkt server.LoginReplyServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize login reply: %w", err)
	}

	switch pkt.ReplyCode {
	case server.LoginReply_Ok:
		d := pkt.ReplyCodeData.(*server.LoginReplyReplyCodeDataOk)
		c.Characters = d.Characters
		c.SetState(StateLoggedIn)
		slog.Info("login successful", "characters", len(d.Characters))
		c.Emit(Event{Type: EventCharacterList, Data: d.Characters})
	case server.LoginReply_WrongUser:
		c.Emit(Event{Type: EventError, Message: "Account not found"})
	case server.LoginReply_WrongUserPassword:
		c.Emit(Event{Type: EventError, Message: "Wrong password"})
	case server.LoginReply_LoggedIn:
		c.Emit(Event{Type: EventError, Message: "Already logged in"})
	default:
		c.Emit(Event{Type: EventError, Message: fmt.Sprintf("Login failed: code %d", pkt.ReplyCode)})
	}
	return nil
}

func handleCharacterReply(c *Client, reader *data.EoReader) error {
	var pkt server.CharacterReplyServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize character reply: %w", err)
	}

	switch pkt.ReplyCode {
	case server.CharacterReply_Exists:
		c.Emit(Event{Type: EventError, Message: "Character name already exists"})
		return nil
	case server.CharacterReply_Full, server.CharacterReply_Full3:
		c.Emit(Event{Type: EventError, Message: "Character roster is full"})
		return nil
	case server.CharacterReply_NotApproved:
		c.Emit(Event{Type: EventError, Message: "Character name was not approved"})
		return nil
	case server.CharacterReply_Ok:
		d, ok := pkt.ReplyCodeData.(*server.CharacterReplyReplyCodeDataOk)
		if !ok || len(d.Characters) == 0 {
			c.Emit(Event{Type: EventError, Message: "Character creation failed"})
			return nil
		}

		c.PendingCharacterCreate = nil
		c.Characters = d.Characters
		c.Emit(Event{Type: EventCharacterList, Data: d.Characters})
		slog.Info("character list updated", "count", len(d.Characters))
		return nil
	default:
		pending := c.PendingCharacterCreate
		bus := c.GetBus()
		if pending == nil || bus == nil {
			return nil
		}

		c.SessionID = int(pkt.ReplyCode)
		if err := bus.SendSequenced(&client.CharacterCreateClientPacket{
			SessionId: c.SessionID,
			Gender:    pending.Gender,
			HairStyle: pending.HairStyle,
			HairColor: pending.HairColor,
			Skin:      pending.Skin,
			Name:      pending.Name,
		}); err != nil {
			return fmt.Errorf("character create confirm failed: %w", err)
		}
		return nil
	}
}

func handleWelcomeReply(c *Client, reader *data.EoReader) error {
	var pkt server.WelcomeReplyServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize welcome reply: %w", err)
	}

	switch pkt.WelcomeCode {
	case server.WelcomeCode_SelectCharacter:
		d := pkt.WelcomeCodeData.(*server.WelcomeReplyWelcomeCodeDataSelectCharacter)
		c.SessionID = d.SessionId
		c.Character.ID = d.CharacterId
		c.Character.MapID = int(d.MapId)
		syncWelcomeCharacterState(c, d)
		slog.Info("character selected", "sessionID", d.SessionId, "mapID", d.MapId)

		// Send welcome message to enter game
		bus := c.GetBus()
		if bus == nil {
			return fmt.Errorf("connection bus missing during welcome")
		}
		if err := bus.SendSequenced(&client.WelcomeMsgClientPacket{
			SessionId:   d.SessionId,
			CharacterId: d.CharacterId,
		}); err != nil {
			return fmt.Errorf("enter game failed while sending welcome message: %w", err)
		}
		return nil

	case server.WelcomeCode_EnterGame:
		d := pkt.WelcomeCodeData.(*server.WelcomeReplyWelcomeCodeDataEnterGame)

		c.mu.Lock()
		// Populate nearby characters
		c.NearbyChars = nil
		for _, ch := range d.Nearby.Characters {
			syncLocalVitalsFromCharacter(c, ch)
			boots, armor, hat, weapon, shield := visibleEquipmentFromCharacterMapInfo(ch)
			nc := NearbyCharacter{
				PlayerID:  ch.PlayerId,
				Name:      ch.Name,
				GuildTag:  ch.GuildTag,
				X:         ch.Coords.X,
				Y:         ch.Coords.Y,
				Direction: int(ch.Direction),
				Gender:    int(ch.Gender),
				Skin:      ch.Skin,
				HairStyle: ch.HairStyle,
				HairColor: ch.HairColor,
				Armor:     armor,
				Boots:     boots,
				Hat:       hat,
				Weapon:    weapon,
				Shield:    shield,
				SitState:  int(ch.SitState),
				Level:     ch.Level,
			}
			c.NearbyChars = append(c.NearbyChars, nc)

		}

		// Populate nearby NPCs
		c.NearbyNpcs = nil
		for _, npc := range d.Nearby.Npcs {
			c.NearbyNpcs = append(c.NearbyNpcs, NearbyNPC{
				Index:     npc.Index,
				ID:        npc.Id,
				X:         npc.Coords.X,
				Y:         npc.Coords.Y,
				Direction: int(npc.Direction),
			})
		}

		// Populate nearby items
		c.NearbyItems = nil
		for _, item := range d.Nearby.Items {
			c.NearbyItems = append(c.NearbyItems, NearbyItem{
				UID:       item.Uid,
				ID:        item.Id,
				GraphicID: item.Id, // simplified; real mapping uses eif spec1
				X:         item.Coords.X,
				Y:         item.Coords.Y,
				Amount:    item.Amount,
			})
		}

		syncWeight(c, d.Weight)
		setInventoryFromNetItems(c, d.Items)
		c.mu.Unlock()

		c.SetState(StateInGame)
		slog.Info("entered game",
			"map", c.Character.MapID,
			"x", c.Character.X,
			"y", c.Character.Y,
			"name", c.Character.Name,
			"chars", len(c.NearbyChars),
			"npcs", len(c.NearbyNpcs),
			"items", len(c.NearbyItems),
		)

		if bus := c.GetBus(); bus != nil {
			if err := bus.SendSequenced(&client.PaperdollRequestClientPacket{PlayerId: c.PlayerID}); err != nil {
				slog.Warn("paperdoll request failed", "err", err)
			}
		}
		c.Emit(Event{Type: EventEnterGame, Data: d})
	}
	return nil
}

func handleTalkPlayer(c *Client, reader *data.EoReader) error {
	var pkt server.TalkPlayerServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return err
	}
	emitChat(c, ChatChannelMap, fmt.Sprintf("%s: %s", chatCharacterName(c, pkt.PlayerId), pkt.Message))
	return nil
}

func handleTalkOpen(c *Client, reader *data.EoReader) error {
	var pkt server.TalkOpenServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return err
	}
	emitChat(c, ChatChannelGroup, fmt.Sprintf("%s: %s", chatCharacterName(c, pkt.PlayerId), pkt.Message))
	return nil
}

func handleTalkMsg(c *Client, reader *data.EoReader) error {
	var pkt server.TalkMsgServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return err
	}
	emitChat(c, ChatChannelGlobal, fmt.Sprintf("%s: %s", pkt.PlayerName, pkt.Message))
	return nil
}

func handleTalkTell(c *Client, reader *data.EoReader) error {
	var pkt server.TalkTellServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return err
	}
	emitChat(c, ChatChannelMap, fmt.Sprintf("[PM] %s: %s", pkt.PlayerName, pkt.Message))
	return nil
}

func handleTalkReply(c *Client, reader *data.EoReader) error {
	var pkt server.TalkReplyServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return err
	}
	if pkt.ReplyCode == server.TalkReply_NotFound {
		emitChat(c, ChatChannelSystem, fmt.Sprintf("%s could not be found.", pkt.Name))
	}
	return nil
}

func handleTalkAdmin(c *Client, reader *data.EoReader) error {
	var pkt server.TalkAdminServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return err
	}
	emitChat(c, ChatChannelGroup, fmt.Sprintf("[GM] %s: %s", pkt.PlayerName, pkt.Message))
	return nil
}

func handleTalkAnnounce(c *Client, reader *data.EoReader) error {
	var pkt server.TalkAnnounceServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return err
	}
	text := fmt.Sprintf("[Announcement] %s: %s", pkt.PlayerName, pkt.Message)
	emitChatAllChannels(c, text)
	return nil
}
