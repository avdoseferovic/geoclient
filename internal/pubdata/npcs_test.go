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
