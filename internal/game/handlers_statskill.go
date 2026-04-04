package game

import (
	"fmt"
	"log/slog"

	"github.com/ethanmoffat/eolib-go/v3/data"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
)

func RegisterStatSkillHandlers(reg *HandlerRegistry) {
	reg.Register(16, 8, handleStatSkillPlayer) // StatSkill_Player
}

// handleStatSkillPlayer updates character stats after stat point allocation.
func handleStatSkillPlayer(c *Client, reader *data.EoReader) error {
	var pkt server.StatSkillPlayerServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize stat skill player: %w", err)
	}

	c.mu.Lock()
	c.Character.StatPoints = pkt.StatPoints
	applyCharacterStatsUpdate(c, pkt.Stats)
	c.mu.Unlock()

	slog.Debug("stat trained", "statPoints", pkt.StatPoints)
	c.Emit(Event{Type: EventStatsUpdated})
	return nil
}

func applyCharacterStatsUpdate(c *Client, stats server.CharacterStatsUpdate) {
	c.Character.BaseStats = CharacterBaseStats{
		Str: stats.BaseStats.Str,
		Int: stats.BaseStats.Intl,
		Wis: stats.BaseStats.Wis,
		Agi: stats.BaseStats.Agi,
		Con: stats.BaseStats.Con,
		Cha: stats.BaseStats.Cha,
	}
	c.Character.MaxHP = stats.MaxHp
	c.Character.MaxTP = stats.MaxTp
	c.Character.MaxSP = stats.MaxSp
	c.Character.CombatStats = CharacterCombatStats{
		MinDamage: stats.SecondaryStats.MinDamage,
		MaxDamage: stats.SecondaryStats.MaxDamage,
		Accuracy:  stats.SecondaryStats.Accuracy,
		Evade:     stats.SecondaryStats.Evade,
		Armor:     stats.SecondaryStats.Armor,
	}
}
