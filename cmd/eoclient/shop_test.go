package main

import (
	"slices"
	"testing"

	"github.com/ethanmoffat/eolib-go/v3/data"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"
	"github.com/ethanmoffat/eolib-go/v3/protocol/pub"

	"github.com/avdo/eoweb/internal/game"
	internalnet "github.com/avdo/eoweb/internal/net"
	"github.com/avdo/eoweb/internal/pubdata"
)

func TestTryOpenShopAtAdjacentVendorSendsPacket(t *testing.T) {
	conn := &fakePacketConn{}
	bus := internalnet.NewPacketBus(conn)
	c := game.NewClient()
	c.SetBus(bus)
	c.Character.X = 10
	c.Character.Y = 10
	c.NearbyNpcs = []game.NearbyNPC{{Index: 4, ID: 15, X: 11, Y: 10}}

	g := &Game{
		client: c,
		npcDB:  pubdata.NewNPCDB(pubdata.NPCDef{ID: 15, Name: "Vendor", Type: pub.Npc_Shop}),
	}

	if !g.tryOpenShopAt(11, 10) {
		t.Fatal("tryOpenShopAt() = false, want true")
	}
	if len(conn.writes) != 1 {
		t.Fatalf("writes = %d, want 1", len(conn.writes))
	}

	var sent client.ShopOpenClientPacket
	if err := sent.Deserialize(packetPayload(t, conn.writes[0])); err != nil {
		t.Fatalf("deserialize sent packet: %v", err)
	}
	if sent.NpcIndex != 4 {
		t.Fatalf("npc index = %d, want 4", sent.NpcIndex)
	}
}

func TestTryOpenShopAtNonVendorDoesNothing(t *testing.T) {
	conn := &fakePacketConn{}
	bus := internalnet.NewPacketBus(conn)
	c := game.NewClient()
	c.SetBus(bus)
	c.Character.X = 10
	c.Character.Y = 10
	c.NearbyNpcs = []game.NearbyNPC{{Index: 4, ID: 15, X: 11, Y: 10}}

	g := &Game{
		client: c,
		npcDB:  pubdata.NewNPCDB(pubdata.NPCDef{ID: 15, Name: "Wolf", Type: pub.Npc_Aggressive}),
	}

	if g.tryOpenShopAt(11, 10) {
		t.Fatal("tryOpenShopAt() = true, want false")
	}
	if len(conn.writes) != 0 {
		t.Fatalf("writes = %d, want 0", len(conn.writes))
	}
}

func TestClickedNpcNeverAttacks(t *testing.T) {
	g := &Game{
		client: &game.Client{
			Character: game.Character{X: 10, Y: 10},
			NearbyNpcs: []game.NearbyNPC{
				{Index: 2, ID: 7, X: 11, Y: 10},
			},
		},
	}

	if g.isAttackTargetTile(11, 10) {
		t.Fatal("isAttackTargetTile() = true for npc tile, want false")
	}
}

func TestNormalizeShopStateKeepsValidSectionAndSelectsFirstItem(t *testing.T) {
	g := &Game{}
	g.overlay.shopView = shopViewSection
	g.overlay.shopTab = shopTabSell

	snapshot := game.UISnapshot{
		Inventory: []game.InventoryItem{{ID: 1, Amount: 150}, {ID: 2, Amount: 3}},
		Shop: game.ShopState{
			TradeItems: []game.ShopTradeItem{
				{ItemID: 4, BuyPrice: 25, MaxBuyAmount: 99},
				{ItemID: 2, SellPrice: 6},
			},
		},
	}

	g.normalizeShopState(snapshot)

	if g.overlay.shopTab != shopTabSell {
		t.Fatalf("shopTab = %d, want %d", g.overlay.shopTab, shopTabSell)
	}
	if g.overlay.shopSelectedItem != 2 {
		t.Fatalf("shopSelectedItem = %d, want 2", g.overlay.shopSelectedItem)
	}
}

func TestNormalizeShopStateFallsBackToFirstAvailableSection(t *testing.T) {
	g := &Game{}
	g.overlay.shopView = shopViewSection
	g.overlay.shopTab = shopTabSell

	snapshot := game.UISnapshot{
		Inventory: []game.InventoryItem{{ID: 1, Amount: 150}},
		Shop: game.ShopState{
			TradeItems: []game.ShopTradeItem{
				{ItemID: 4, BuyPrice: 25, MaxBuyAmount: 99},
			},
		},
	}

	g.normalizeShopState(snapshot)

	if g.overlay.shopTab != shopTabBuy {
		t.Fatalf("shopTab = %d, want %d", g.overlay.shopTab, shopTabBuy)
	}
	if g.overlay.shopSelectedItem != 4 {
		t.Fatalf("shopSelectedItem = %d, want 4", g.overlay.shopSelectedItem)
	}
}

func TestNormalizeShopStateReturnsToMenuWhenNothingAvailable(t *testing.T) {
	g := &Game{}
	g.overlay.shopView = shopViewSection
	g.overlay.shopTab = shopTabBuy
	g.overlay.shopSelectedItem = 99

	g.normalizeShopState(game.UISnapshot{})

	if g.overlay.shopView != shopViewMenu {
		t.Fatalf("shopView = %d, want %d", g.overlay.shopView, shopViewMenu)
	}
	if g.overlay.shopSelectedItem != 0 {
		t.Fatalf("shopSelectedItem = %d, want 0", g.overlay.shopSelectedItem)
	}
}

func TestShopSectionButtonsExposeAvailableSections(t *testing.T) {
	g := &Game{}

	snapshot := game.UISnapshot{
		Inventory: []game.InventoryItem{{ID: 7, Amount: 2}},
		Shop: game.ShopState{
			TradeItems: []game.ShopTradeItem{
				{ItemID: 5, BuyPrice: 50, MaxBuyAmount: 99},
				{ItemID: 7, SellPrice: 12},
			},
			CraftItems: []game.ShopCraftItem{{ItemID: 9}},
		},
	}

	buttons := g.shopSectionButtons(snapshot)
	labels := make([]string, 0, len(buttons))
	for _, button := range buttons {
		labels = append(labels, button.Label)
	}

	if !slices.Equal(labels, []string{"Buy", "Sell", "Craft"}) {
		t.Fatalf("labels = %v, want [Buy Sell Craft]", labels)
	}
	if buttons[0].Count != 1 || buttons[1].Count != 1 || buttons[2].Count != 1 {
		t.Fatalf("counts = %v, want [1 1 1]", []int{buttons[0].Count, buttons[1].Count, buttons[2].Count})
	}
}

func packetPayload(t *testing.T, buf []byte) *data.EoReader {
	t.Helper()
	return data.NewEoReader(buf[5:])
}
