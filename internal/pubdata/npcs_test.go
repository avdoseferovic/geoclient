package pubdata

import "testing"

func TestNPCName(t *testing.T) {
	db := &NPCDB{
		byID: map[int]NPCDef{
			28: {ID: 28, Name: "Guild Bob"},
		},
	}

	if got := db.Name(28); got != "Guild Bob" {
		t.Fatalf("Name(28) = %q, want %q", got, "Guild Bob")
	}
	if got := db.Name(99); got != "" {
		t.Fatalf("Name(99) = %q, want empty", got)
	}
}

func TestNPCType(t *testing.T) {
	db := &NPCDB{
		byID: map[int]NPCDef{
			15: {ID: 15, Name: "Shop Bob", Type: 6},
		},
	}

	if got := db.Type(15); got != 6 {
		t.Fatalf("Type(15) = %v, want 6", got)
	}
	if got := db.Type(99); got != 0 {
		t.Fatalf("Type(99) = %v, want 0", got)
	}
}
