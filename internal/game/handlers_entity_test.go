package game

import (
	"testing"

	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
	"github.com/ethanmoffat/eolib-go/v3/protocol/pub"
)

func TestHandlePaperdollRemoveClearsEquippedItemWhenSubLocIsRelative(t *testing.T) {
	c := NewClient()
	c.Equipment.Hat = 123

	err := handlePaperdollRemove(c, packetReader(t, &server.PaperdollRemoveServerPacket{
		ItemId: 123,
		SubLoc: 0,
	}))
	if err != nil {
		t.Fatalf("handlePaperdollRemove returned error: %v", err)
	}
	if c.Equipment.Hat != 0 {
		t.Fatalf("hat slot = %d, want 0", c.Equipment.Hat)
	}
	if len(c.Inventory) != 1 || c.Inventory[0].ID != 123 || c.Inventory[0].Amount != 1 {
		t.Fatalf("inventory = %#v", c.Inventory)
	}
}

func TestFindEquippedPaperdollSlotUsesSubLocForSecondDualSlot(t *testing.T) {
	equipment := server.EquipmentPaperdoll{
		Ring: []int{456, 456},
	}

	if got := findEquippedPaperdollSlot(equipment, 456, 0); got != 10 {
		t.Fatalf("findEquippedPaperdollSlot(subLoc=0) = %d, want 10", got)
	}
	if got := findEquippedPaperdollSlot(equipment, 456, 1); got != 11 {
		t.Fatalf("findEquippedPaperdollSlot(subLoc=1) = %d, want 11", got)
	}
}

func TestItemTypeToEquipSlotUsesPubItemTypeValues(t *testing.T) {
	c := NewClient()

	tests := []struct {
		name     string
		itemType pub.ItemType
		subLoc   int
		want     int
	}{
		{name: "hat", itemType: pub.Item_Hat, subLoc: 0, want: 7},
		{name: "boots", itemType: pub.Item_Boots, subLoc: 0, want: 1},
		{name: "weapon", itemType: pub.Item_Weapon, subLoc: 0, want: 9},
		{name: "ring second", itemType: pub.Item_Ring, subLoc: 1, want: 11},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c.ItemTypeFunc = func(int) int { return int(tt.itemType) }
			if got := itemTypeToEquipSlot(c, 999, tt.subLoc); got != tt.want {
				t.Fatalf("itemTypeToEquipSlot(%v, %d) = %d, want %d", tt.itemType, tt.subLoc, got, tt.want)
			}
		})
	}
}
