package game

import (
	"fmt"

	"github.com/ethanmoffat/eolib-go/v3/data"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
	"github.com/ethanmoffat/eolib-go/v3/protocol/pub"
)

func handlePaperdollReply(c *Client, reader *data.EoReader) error {
	var pkt server.PaperdollReplyServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize paperdoll reply: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	syncPaperdollDetails(c, pkt.Details)
	c.Equipment = pkt.Equipment
	return nil
}

func handlePaperdollAgree(c *Client, reader *data.EoReader) error {
	var pkt server.PaperdollAgreeServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize paperdoll agree: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	setInventoryAmount(&c.Inventory, pkt.ItemId, pkt.RemainingAmount)
	equipSlot := itemTypeToEquipSlot(c, pkt.ItemId, pkt.SubLoc)
	applyPaperdollSubLoc(&c.Equipment, equipSlot, pkt.ItemId)
	applyPaperdollAvatarChange(c, pkt.Change)
	applyEquipmentChangeStats(c, pkt.Stats)
	return nil
}

func handlePaperdollRemove(c *Client, reader *data.EoReader) error {
	var pkt server.PaperdollRemoveServerPacket
	if err := pkt.Deserialize(reader); err != nil {
		return fmt.Errorf("deserialize paperdoll remove: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	equipSlot := findEquippedPaperdollSlot(c.Equipment, pkt.ItemId, pkt.SubLoc)
	addInventoryAmount(&c.Inventory, pkt.ItemId, 1)
	applyPaperdollSubLoc(&c.Equipment, equipSlot, 0)
	applyPaperdollAvatarChange(c, pkt.Change)
	applyEquipmentChangeStats(c, pkt.Stats)
	return nil
}

func applyPaperdollAvatarChange(c *Client, change server.AvatarChange) {
	if ch := findCharacter(c.NearbyChars, change.PlayerId); ch != nil {
		switch change.ChangeType {
		case server.AvatarChange_Equipment:
			if update, ok := change.ChangeTypeData.(*server.ChangeTypeDataEquipment); ok {
				applyEquipmentChange(ch, update.Equipment)
			}
		case server.AvatarChange_Hair:
			if update, ok := change.ChangeTypeData.(*server.ChangeTypeDataHair); ok {
				ch.HairStyle = update.HairStyle
				ch.HairColor = update.HairColor
			}
		case server.AvatarChange_HairColor:
			if update, ok := change.ChangeTypeData.(*server.ChangeTypeDataHairColor); ok {
				ch.HairColor = update.HairColor
			}
		}
	}
}

func itemTypeToEquipSlot(c *Client, itemID, subLoc int) int {
	if c.ItemTypeFunc == nil {
		return subLoc
	}
	itemType := pub.ItemType(c.ItemTypeFunc(itemID))
	switch itemType {
	case pub.Item_Weapon:
		return 9
	case pub.Item_Shield:
		return 8
	case pub.Item_Armor:
		return 5
	case pub.Item_Hat:
		return 7
	case pub.Item_Boots:
		return 1
	case pub.Item_Gloves:
		return 3
	case pub.Item_Accessory:
		return 2
	case pub.Item_Belt:
		return 4
	case pub.Item_Necklace:
		return 6
	case pub.Item_Ring:
		return 10 + subLoc
	case pub.Item_Armlet:
		return 12 + subLoc
	case pub.Item_Bracer:
		return 14 + subLoc
	}
	return subLoc
}

func findEquippedPaperdollSlot(equipment server.EquipmentPaperdoll, itemID, subLoc int) int {
	slot := findEquippedItemSlot(equipment, itemID)
	if subLoc == 1 {
		switch slot {
		case 10, 12, 14:
			slot++
		}
	}
	return slot
}

func findEquippedItemSlot(equipment server.EquipmentPaperdoll, itemID int) int {
	switch itemID {
	case equipment.Boots:
		return 1
	case equipment.Accessory:
		return 2
	case equipment.Gloves:
		return 3
	case equipment.Belt:
		return 4
	case equipment.Armor:
		return 5
	case equipment.Necklace:
		return 6
	case equipment.Hat:
		return 7
	case equipment.Shield:
		return 8
	case equipment.Weapon:
		return 9
	}
	for i, equippedID := range equipment.Ring {
		if equippedID == itemID {
			return 10 + i
		}
	}
	for i, equippedID := range equipment.Armlet {
		if equippedID == itemID {
			return 12 + i
		}
	}
	for i, equippedID := range equipment.Bracer {
		if equippedID == itemID {
			return 14 + i
		}
	}
	return 0
}

func applyPaperdollSubLoc(equipment *server.EquipmentPaperdoll, subLoc, itemID int) {
	if equipment == nil {
		return
	}

	switch subLoc {
	case 1:
		equipment.Boots = itemID
	case 2:
		equipment.Accessory = itemID
	case 3:
		equipment.Gloves = itemID
	case 4:
		equipment.Belt = itemID
	case 5:
		equipment.Armor = itemID
	case 6:
		equipment.Necklace = itemID
	case 7:
		equipment.Hat = itemID
	case 8:
		equipment.Shield = itemID
	case 9:
		equipment.Weapon = itemID
	case 10, 11:
		index := subLoc - 10
		for len(equipment.Ring) <= index {
			equipment.Ring = append(equipment.Ring, 0)
		}
		equipment.Ring[index] = itemID
	case 12, 13:
		index := subLoc - 12
		for len(equipment.Armlet) <= index {
			equipment.Armlet = append(equipment.Armlet, 0)
		}
		equipment.Armlet[index] = itemID
	case 14, 15:
		index := subLoc - 14
		for len(equipment.Bracer) <= index {
			equipment.Bracer = append(equipment.Bracer, 0)
		}
		equipment.Bracer[index] = itemID
	}
}
