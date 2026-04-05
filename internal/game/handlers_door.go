package game

import (
	"fmt"
	"log/slog"

	"github.com/ethanmoffat/eolib-go/v3/data"
	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
)

func RegisterDoorHandlers(reg *HandlerRegistry) {
	reg.Register(eonet.PacketFamily_Door, eonet.PacketAction_Open, handleDoorOpen)
	reg.Register(eonet.PacketFamily_Door, eonet.PacketAction_Close, handleDoorClose)
}

func handleDoorOpen(c *Client, reader *data.EoReader) error {
	var pkt server.DoorOpenServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize door open: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	door := c.GetDoor(pkt.Coords.X, pkt.Coords.Y)
	if door == nil {
		slog.Debug("door open for unknown door", "x", pkt.Coords.X, "y", pkt.Coords.Y)
		return nil
	}

	door.Open = true
	door.OpenTicks = DoorOpenTicks
	return nil
}

func handleDoorClose(c *Client, reader *data.EoReader) error {
	var pkt server.DoorCloseServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize door close: %w", err)
	}
	slog.Debug("door locked", "key", pkt.Key)
	emitChat(c, ChatChannelSystem, "The door is locked.")
	return nil
}
