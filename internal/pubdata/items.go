package pubdata

import (
	"fmt"
	"strings"

	"github.com/avdoseferovic/geoclient/internal/assets"
	"github.com/ethanmoffat/eolib-go/v3/data"
	"github.com/ethanmoffat/eolib-go/v3/protocol/pub"
)

type ItemDef struct {
	ID               int
	Name             string
	GraphicID        int
	Size             pub.ItemSize
	Type             pub.ItemType
	Special          pub.ItemSpecial
	HP               int
	TP               int
	MinDamage        int
	MaxDamage        int
	Accuracy         int
	Evade            int
	Armor            int
	Str              int
	Intl             int
	Wis              int
	Agi              int
	Con              int
	Cha              int
	Spec2            int
	LevelRequirement int
	ClassRequirement int
	StrRequirement   int
	IntRequirement   int
	WisRequirement   int
	AgiRequirement   int
	ConRequirement   int
	ChaRequirement   int
}

type ItemDB struct {
	byID map[int]ItemDef
}

func LoadItemDB(path string) (*ItemDB, error) {
	return LoadItemDBFromReader(assets.NewOSReader(), path)
}

func LoadItemDBFromReader(reader assets.Reader, path string) (*ItemDB, error) {
	raw, err := reader.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read EIF: %w", err)
	}
	return LoadItemDBFromBytes(raw)
}

func LoadItemDBFromBytes(raw []byte) (*ItemDB, error) {
	reader := data.NewEoReader(raw)
	var eif pub.Eif
	if err := eif.Deserialize(reader); err != nil {
		return nil, fmt.Errorf("deserialize EIF: %w", err)
	}

	db := &ItemDB{byID: make(map[int]ItemDef, len(eif.Items))}
	for i, item := range eif.Items {
		id := i + 1
		db.byID[id] = ItemDef{
			ID:               id,
			Name:             item.Name,
			GraphicID:        item.GraphicId,
			Size:             item.Size,
			Type:             item.Type,
			Special:          item.Special,
			HP:               item.Hp,
			TP:               item.Tp,
			MinDamage:        item.MinDamage,
			MaxDamage:        item.MaxDamage,
			Accuracy:         item.Accuracy,
			Evade:            item.Evade,
			Armor:            item.Armor,
			Str:              item.Str,
			Intl:             item.Intl,
			Wis:              item.Wis,
			Agi:              item.Agi,
			Con:              item.Con,
			Cha:              item.Cha,
			Spec2:            item.Spec2,
			LevelRequirement: item.LevelRequirement,
			ClassRequirement: item.ClassRequirement,
			StrRequirement:   item.StrRequirement,
			IntRequirement:   item.IntRequirement,
			WisRequirement:   item.WisRequirement,
			AgiRequirement:   item.AgiRequirement,
			ConRequirement:   item.ConRequirement,
			ChaRequirement:   item.ChaRequirement,
		}
	}

	return db, nil
}

func (db *ItemDB) Get(id int) (ItemDef, bool) {
	if db == nil || id <= 0 {
		return ItemDef{}, false
	}
	item, ok := db.byID[id]
	return item, ok
}

func (db *ItemDB) Name(id int) string {
	if item, ok := db.Get(id); ok && item.Name != "" {
		return item.Name
	}
	return ""
}

func (db *ItemDB) MetaLines(id int) []string {
	item, ok := db.Get(id)
	if !ok {
		return nil
	}

	lines := []string{item.typeLabel()}
	if item.Type >= pub.Item_Weapon && item.Type <= pub.Item_Bracer {
		if item.MinDamage != 0 || item.MaxDamage != 0 {
			lines = append(lines, fmt.Sprintf("damage: %d-%d", item.MinDamage, item.MaxDamage))
		}
		if item.HP != 0 || item.TP != 0 {
			line := "add+"
			if item.HP != 0 {
				line += fmt.Sprintf(" %dhp", item.HP)
			}
			if item.TP != 0 {
				line += fmt.Sprintf(" %dtp", item.TP)
			}
			lines = append(lines, line)
		}
		if item.Accuracy != 0 {
			lines = append(lines, fmt.Sprintf("plus+ %dhit", item.Accuracy))
		}
		if item.Evade != 0 || item.Armor != 0 {
			line := "def+"
			if item.Evade != 0 {
				line += fmt.Sprintf(" %deva", item.Evade)
			}
			if item.Armor != 0 {
				line += fmt.Sprintf(" %darm", item.Armor)
			}
			lines = append(lines, line)
		}
		if item.Str != 0 || item.Intl != 0 || item.Wis != 0 || item.Agi != 0 || item.Cha != 0 || item.Con != 0 {
			line := "stat+"
			if item.Str != 0 {
				line += fmt.Sprintf(" %dstr", item.Str)
			}
			if item.Intl != 0 {
				line += fmt.Sprintf(" %dint", item.Intl)
			}
			if item.Wis != 0 {
				line += fmt.Sprintf(" %dwis", item.Wis)
			}
			if item.Agi != 0 {
				line += fmt.Sprintf(" %dagi", item.Agi)
			}
			if item.Cha != 0 {
				line += fmt.Sprintf(" %dcha", item.Cha)
			}
			if item.Con != 0 {
				line += fmt.Sprintf(" %dcon", item.Con)
			}
			lines = append(lines, line)
		}
	}

	req := make([]string, 0, 8)
	if item.LevelRequirement != 0 {
		req = append(req, fmt.Sprintf("%dLVL", item.LevelRequirement))
	}
	if item.ClassRequirement != 0 {
		req = append(req, fmt.Sprintf("Class %d", item.ClassRequirement))
	}
	if item.StrRequirement != 0 {
		req = append(req, fmt.Sprintf("%dstr", item.StrRequirement))
	}
	if item.IntRequirement != 0 {
		req = append(req, fmt.Sprintf("%dint", item.IntRequirement))
	}
	if item.WisRequirement != 0 {
		req = append(req, fmt.Sprintf("%dwis", item.WisRequirement))
	}
	if item.AgiRequirement != 0 {
		req = append(req, fmt.Sprintf("%dagi", item.AgiRequirement))
	}
	if item.ChaRequirement != 0 {
		req = append(req, fmt.Sprintf("%dcha", item.ChaRequirement))
	}
	if item.ConRequirement != 0 {
		req = append(req, fmt.Sprintf("%dcon", item.ConRequirement))
	}
	if len(req) > 0 {
		lines = append(lines, "req: "+strings.Join(req, " "))
	}

	return lines
}

func (item ItemDef) typeLabel() string {
	switch item.Type {
	case pub.Item_General:
		return "general item"
	case pub.Item_Currency:
		return "currency"
	case pub.Item_Heal:
		label := "potion"
		if item.HP != 0 {
			label += fmt.Sprintf(" + %dhp", item.HP)
		}
		if item.TP != 0 {
			label += fmt.Sprintf(" + %dtp", item.TP)
		}
		return label
	case pub.Item_Teleport:
		return "teleport"
	case pub.Item_ExpReward:
		return "exp reward"
	case pub.Item_Key:
		return "key"
	case pub.Item_Alcohol:
		return "beverage"
	case pub.Item_EffectPotion:
		return "effect"
	case pub.Item_HairDye:
		return "hairdye"
	case pub.Item_CureCurse:
		return "cure"
	}

	prefix := "normal"
	switch item.Special {
	case pub.ItemSpecial_Cursed:
		prefix = "cursed"
	case pub.ItemSpecial_Lore:
		prefix = "lore"
	}
	if item.Type == pub.Item_Armor {
		if item.Spec2 == 1 {
			prefix += " male"
		} else {
			prefix += " female"
		}
	}
	switch item.Type {
	case pub.Item_Weapon:
		return prefix + " weapon"
	case pub.Item_Shield:
		return prefix + " shield"
	case pub.Item_Armor:
		return prefix + " clothing"
	case pub.Item_Hat:
		return prefix + " hat"
	case pub.Item_Boots:
		return prefix + " boots"
	case pub.Item_Gloves:
		return prefix + " gloves"
	case pub.Item_Accessory:
		return prefix + " accessory"
	case pub.Item_Belt:
		return prefix + " belt"
	case pub.Item_Necklace:
		return prefix + " necklace"
	case pub.Item_Ring:
		return prefix + " ring"
	case pub.Item_Armlet:
		return prefix + " bracelet"
	case pub.Item_Bracer:
		return prefix + " bracer"
	default:
		return prefix
	}
}

func (db *ItemDB) GraphicResourceID(id, amount int) int {
	item, ok := db.Get(id)
	if !ok || item.GraphicID <= 0 {
		return 0
	}
	return GraphicResourceID(id, item.GraphicID, amount)
}

func (db *ItemDB) GridGraphicResourceID(id int) int {
	item, ok := db.Get(id)
	if !ok || item.GraphicID <= 0 {
		return 0
	}
	return item.GraphicID * 2
}

func GraphicResourceID(itemID, graphicID, amount int) int {
	if graphicID <= 0 {
		return 0
	}
	if itemID == 1 {
		offset := 0
		switch {
		case amount >= 100_000:
			offset = 4
		case amount >= 10_000:
			offset = 3
		case amount >= 100:
			offset = 2
		case amount >= 2:
			offset = 1
		}
		return 269 + 2*offset
	}
	return graphicID*2 - 1
}
