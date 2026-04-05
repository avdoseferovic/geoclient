package game

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/ethanmoffat/eolib-go/v3/data"
	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
)

// OnlinePlayer holds info about a player in the online list.
type OnlinePlayer struct {
	Name     string
	Title    string
	Level    int
	Icon     int
	ClassID  int
	GuildTag string
}

func RegisterPlayersHandlers(reg *HandlerRegistry) {
	reg.Register(eonet.PacketFamily_Players, eonet.PacketAction_List, handlePlayersList)
}

func handleInitPlayersList(c *Client, d *server.InitInitReplyCodeDataPlayersList) error {
	players := convertOnlinePlayers(d.PlayersList.Players)
	setOnlinePlayers(c, players)
	return nil
}

func handlePlayersList(c *Client, reader *data.EoReader) error {
	var pkt server.PlayersListServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize players list: %w", err)
	}
	players := convertOnlinePlayers(pkt.PlayersList.Players)
	setOnlinePlayers(c, players)
	return nil
}

func convertOnlinePlayers(src []server.OnlinePlayer) []OnlinePlayer {
	players := make([]OnlinePlayer, 0, len(src))
	for _, p := range src {
		players = append(players, OnlinePlayer{
			Name:     p.Name,
			Title:    p.Title,
			Level:    p.Level,
			Icon:     int(p.Icon),
			ClassID:  p.ClassId,
			GuildTag: p.GuildTag,
		})
	}
	sort.Slice(players, func(i, j int) bool {
		return players[i].Name < players[j].Name
	})
	return players
}

func setOnlinePlayers(c *Client, players []OnlinePlayer) {
	c.mu.Lock()
	c.OnlinePlayers = players
	c.mu.Unlock()
	slog.Info("online players received", "count", len(players))
	c.Emit(Event{Type: EventOnlinePlayers, Data: players})
}
