package game

import (
	eonet "github.com/ethanmoffat/eolib-go/v3/protocol/net"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
)

func setInventoryFromNetItems(c *Client, items []eonet.Item) {
	c.Inventory = c.Inventory[:0]
	for _, item := range items {
		setInventoryAmount(&c.Inventory, item.Id, item.Amount)
	}
}

func setInventoryAmount(items *[]InventoryItem, itemID, amount int) {
	if itemID <= 0 {
		return
	}

	for i := range *items {
		if (*items)[i].ID != itemID {
			continue
		}
		if amount <= 0 {
			*items = append((*items)[:i], (*items)[i+1:]...)
			return
		}
		(*items)[i].Amount = amount
		return
	}

	if amount > 0 {
		*items = append(*items, InventoryItem{ID: itemID, Amount: amount})
	}
}

func addInventoryAmount(items *[]InventoryItem, itemID, delta int) {
	if itemID <= 0 || delta == 0 {
		return
	}

	for i := range *items {
		if (*items)[i].ID != itemID {
			continue
		}
		setInventoryAmount(items, itemID, (*items)[i].Amount+delta)
		return
	}

	if delta > 0 {
		*items = append(*items, InventoryItem{ID: itemID, Amount: delta})
	}
}

func syncWeight(c *Client, weight eonet.Weight) {
	c.Character.Weight = weight
}

func syncWelcomeCharacterState(c *Client, d *server.WelcomeReplyWelcomeCodeDataSelectCharacter) {
	if d == nil {
		return
	}

	c.Character.ID = d.CharacterId
	c.Character.Name = d.Name
	c.Character.Title = d.Title
	c.Character.GuildName = d.GuildName
	c.Character.GuildRank = d.GuildRankName
	c.Character.GuildTag = d.GuildTag
	c.Character.MapID = int(d.MapId)
	c.Character.ClassID = d.ClassId
	c.Character.Admin = d.Admin
	c.Character.Level = d.Level
	c.Character.Experience = d.Experience
	c.Character.Usage = d.Usage
	applyWelcomeStats(c, d.Stats)
	applyEquipmentWelcome(c, d.Equipment)
}

func syncPaperdollDetails(c *Client, details server.CharacterDetails) {
	c.Character.Name = firstNonEmpty(details.Name, c.Character.Name)
	c.Character.Title = details.Title
	c.Character.GuildName = details.Guild
	c.Character.GuildRank = details.GuildRank
	c.Character.Home = details.Home
	c.Character.Partner = details.Partner
	c.Character.ClassID = details.ClassId
	c.Character.Gender = details.Gender
	c.Character.Admin = details.Admin
}

func applyWelcomeStats(c *Client, stats server.CharacterStatsWelcome) {
	c.Character.HP = stats.Hp
	c.Character.MaxHP = stats.MaxHp
	c.Character.TP = stats.Tp
	c.Character.MaxTP = stats.MaxTp
	c.Character.MaxSP = stats.MaxSp
	c.Character.StatPoints = stats.StatPoints
	c.Character.SkillPoints = stats.SkillPoints
	c.Character.Karma = stats.Karma
	c.Character.BaseStats = CharacterBaseStats{
		Str: stats.Base.Str,
		Int: stats.Base.Intl,
		Wis: stats.Base.Wis,
		Agi: stats.Base.Agi,
		Con: stats.Base.Con,
		Cha: stats.Base.Cha,
	}
	c.Character.CombatStats = CharacterCombatStats{
		MinDamage: stats.Secondary.MinDamage,
		MaxDamage: stats.Secondary.MaxDamage,
		Accuracy:  stats.Secondary.Accuracy,
		Evade:     stats.Secondary.Evade,
		Armor:     stats.Secondary.Armor,
	}
}

func applyEquipmentChangeStats(c *Client, stats server.CharacterStatsEquipmentChange) {
	if stats.MaxHp > 0 {
		c.Character.MaxHP = stats.MaxHp
		if c.Character.HP > c.Character.MaxHP {
			c.Character.HP = c.Character.MaxHP
		}
	}
	if stats.MaxTp > 0 {
		c.Character.MaxTP = stats.MaxTp
		if c.Character.TP > c.Character.MaxTP {
			c.Character.TP = c.Character.MaxTP
		}
	}
	c.Character.BaseStats = CharacterBaseStats{
		Str: stats.BaseStats.Str,
		Int: stats.BaseStats.Intl,
		Wis: stats.BaseStats.Wis,
		Agi: stats.BaseStats.Agi,
		Con: stats.BaseStats.Con,
		Cha: stats.BaseStats.Cha,
	}
	c.Character.CombatStats = CharacterCombatStats{
		MinDamage: stats.SecondaryStats.MinDamage,
		MaxDamage: stats.SecondaryStats.MaxDamage,
		Accuracy:  stats.SecondaryStats.Accuracy,
		Evade:     stats.SecondaryStats.Evade,
		Armor:     stats.SecondaryStats.Armor,
	}
}

func applyEquipmentWelcome(c *Client, equipment server.EquipmentWelcome) {
	c.Equipment = server.EquipmentPaperdoll{
		Boots:     equipment.Boots,
		Accessory: equipment.Accessory,
		Gloves:    equipment.Gloves,
		Belt:      equipment.Belt,
		Armor:     equipment.Armor,
		Necklace:  equipment.Necklace,
		Hat:       equipment.Hat,
		Shield:    equipment.Shield,
		Weapon:    equipment.Weapon,
		Ring:      append([]int(nil), equipment.Ring...),
		Armlet:    append([]int(nil), equipment.Armlet...),
		Bracer:    append([]int(nil), equipment.Bracer...),
	}
}

func firstNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
