package pubdata

import (
	"fmt"

	"github.com/avdo/eoweb/internal/assets"
	"github.com/ethanmoffat/eolib-go/v3/data"
	"github.com/ethanmoffat/eolib-go/v3/protocol/pub"
)

type NPCDef struct {
	ID        int
	Name      string
	GraphicID int
}

type NPCDB struct {
	byID map[int]NPCDef
}

func LoadNPCDB(path string) (*NPCDB, error) {
	return LoadNPCDBFromReader(assets.NewOSReader(), path)
}

func LoadNPCDBFromReader(reader assets.Reader, path string) (*NPCDB, error) {
	raw, err := reader.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read ENF: %w", err)
	}
	return LoadNPCDBFromBytes(raw)
}

func LoadNPCDBFromBytes(raw []byte) (*NPCDB, error) {
	reader := data.NewEoReader(raw)
	var enf pub.Enf
	if err := enf.Deserialize(reader); err != nil {
		return nil, fmt.Errorf("deserialize ENF: %w", err)
	}

	db := &NPCDB{byID: make(map[int]NPCDef, len(enf.Npcs))}
	for i, npc := range enf.Npcs {
		id := i + 1
		db.byID[id] = NPCDef{
			ID:        id,
			Name:      npc.Name,
			GraphicID: npc.GraphicId,
		}
	}
	return db, nil
}

func (db *NPCDB) Get(id int) (NPCDef, bool) {
	if db == nil || id <= 0 {
		return NPCDef{}, false
	}
	npc, ok := db.byID[id]
	return npc, ok
}

func (db *NPCDB) Name(id int) string {
	if npc, ok := db.Get(id); ok {
		return npc.Name
	}
	return ""
}

func (db *NPCDB) GraphicID(id int) int {
	if npc, ok := db.Get(id); ok {
		return npc.GraphicID
	}
	return 0
}
