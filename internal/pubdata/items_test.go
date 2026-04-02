package pubdata

import (
	"strings"
	"testing"

	"github.com/ethanmoffat/eolib-go/v3/protocol/pub"
)

func TestGraphicResourceID(t *testing.T) {
	if got := GraphicResourceID(100, 7, 1); got != 13 {
		t.Fatalf("GraphicResourceID(normal) = %d, want 13", got)
	}

	tests := []struct {
		amount int
		want   int
	}{
		{1, 269},
		{2, 271},
		{100, 273},
		{10000, 275},
		{100000, 277},
	}

	for _, tt := range tests {
		if got := GraphicResourceID(1, 999, tt.amount); got != tt.want {
			t.Fatalf("GraphicResourceID(gold,%d) = %d, want %d", tt.amount, got, tt.want)
		}
	}
}

func TestGridGraphicResourceID(t *testing.T) {
	db := &ItemDB{
		byID: map[int]ItemDef{
			7: {ID: 7, GraphicID: 11},
		},
	}

	if got := db.GridGraphicResourceID(7); got != 22 {
		t.Fatalf("GridGraphicResourceID = %d, want 22", got)
	}
	if got := db.GridGraphicResourceID(99); got != 0 {
		t.Fatalf("GridGraphicResourceID(miss) = %d, want 0", got)
	}
}

func TestMetaLines(t *testing.T) {
	db := &ItemDB{
		byID: map[int]ItemDef{
			5: {
				ID:               5,
				Name:             "Bronze Sword",
				Type:             pub.Item_Weapon,
				MinDamage:        3,
				MaxDamage:        7,
				Accuracy:         2,
				Str:              1,
				LevelRequirement: 4,
				StrRequirement:   2,
			},
		},
	}

	lines := db.MetaLines(5)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"normal weapon", "damage: 3-7", "plus+ 2hit", "stat+ 1str", "req: 4LVL 2str"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("tooltip metadata missing %q in %q", want, joined)
		}
	}
}
