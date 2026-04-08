package game

import (
	"path/filepath"
	"testing"

	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"
)

func TestHandleInitFilePubEmitsFileUpdatedEvent(t *testing.T) {
	tempDir := t.TempDir()
	c := NewClient()
	c.NpcPubPath = filepath.Join(tempDir, "pub", "dtn001.enf")

	if err := handleInitFilePub(c, client.File_Enf, []byte{1, 2, 3, 4}); err != nil {
		t.Fatalf("handleInitFilePub() error = %v", err)
	}

	select {
	case evt := <-c.Events:
		if evt.Type != EventFileUpdated {
			t.Fatalf("event type = %v, want %v", evt.Type, EventFileUpdated)
		}
		fileType, ok := evt.Data.(client.FileType)
		if !ok {
			t.Fatalf("event data type = %T, want client.FileType", evt.Data)
		}
		if fileType != client.File_Enf {
			t.Fatalf("file type = %v, want %v", fileType, client.File_Enf)
		}
	default:
		t.Fatal("expected file updated event")
	}
}
