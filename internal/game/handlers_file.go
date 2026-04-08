package game

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ethanmoffat/eolib-go/v3/protocol/net/client"
	"github.com/ethanmoffat/eolib-go/v3/protocol/net/server"
)

func (c *Client) requestNextFile() error {
	c.mu.Lock()
	if len(c.PendingFiles) == 0 {
		state := c.State
		c.mu.Unlock()
		if state == StateLoadingFiles {
			c.SetState(StateLoggedIn)
			return c.enterGame()
		}
		// If we were already in game (warp), just emit warp event
		c.Emit(Event{Type: EventWarp, Data: c.Character.MapID})
		return nil
	}

	file := c.PendingFiles[0]
	c.PendingFiles = c.PendingFiles[1:]
	c.mu.Unlock()

	bus := c.GetBus()
	if bus == nil {
		return fmt.Errorf("bus missing during file request")
	}

	slog.Info("requesting file", "type", file.Type, "id", file.ID)
	return bus.SendSequenced(&client.WelcomeAgreeClientPacket{
		FileType:     file.Type,
		SessionId:    c.SessionID,
		FileTypeData: fileTypeData(file.Type, file.ID),
	})
}

func fileTypeData(t client.FileType, id int) client.WelcomeAgreeFileTypeData {
	switch t {
	case client.File_Emf:
		return &client.WelcomeAgreeFileTypeDataEmf{FileId: id}
	case client.File_Eif:
		return &client.WelcomeAgreeFileTypeDataEif{FileId: id}
	case client.File_Enf:
		return &client.WelcomeAgreeFileTypeDataEnf{FileId: id}
	case client.File_Esf:
		return &client.WelcomeAgreeFileTypeDataEsf{FileId: id}
	case client.File_Ecf:
		return &client.WelcomeAgreeFileTypeDataEcf{FileId: id}
	default:
		return nil
	}
}

func (c *Client) enterGame() error {
	bus := c.GetBus()
	if bus == nil {
		return fmt.Errorf("bus missing during enter game")
	}

	return bus.SendSequenced(&client.WelcomeMsgClientPacket{
		SessionId:   c.SessionID,
		CharacterId: c.Character.ID,
	})
}

func (c *Client) checkFile(fileType client.FileType, id int, rid []int, size int) bool {
	if c.AssetReader == nil {
		return false
	}
	path := c.filePath(fileType, id)
	if path == "" {
		return true
	}

	raw, err := c.AssetReader.ReadFile(path)
	if err != nil {
		return false
	}

	if len(raw) != size {
		return false
	}

	if len(raw) >= 4 {
		for i := range 4 {
			if i < len(rid) && int(raw[i]) != rid[i] {
				return false
			}
		}
	}

	return true
}

func (c *Client) filePath(fileType client.FileType, id int) string {
	switch fileType {
	case client.File_Emf:
		return filepath.Join(c.MapsDir, fmt.Sprintf("%05d.emf", id))
	case client.File_Eif:
		return c.ItemPubPath
	case client.File_Enf:
		return c.NpcPubPath
	case client.File_Esf:
		return c.SpellPubPath
	case client.File_Ecf:
		return c.ClassPubPath
	default:
		return ""
	}
}

func handleInitFileEmf(c *Client, d *server.InitInitReplyCodeDataFileEmf) error {
	path := c.filePath(client.File_Emf, c.Character.MapID)
	if err := saveFile(path, d.MapFile.Content); err != nil {
		return err
	}
	return c.requestNextFile()
}

func handleInitFilePub(c *Client, fileType client.FileType, content []byte) error {
	path := c.filePath(fileType, 1)
	if err := saveFile(path, content); err != nil {
		return err
	}
	c.Emit(Event{Type: EventFileUpdated, Data: fileType})
	return c.requestNextFile()
}

func saveFile(path string, content []byte) error {
	if path == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	slog.Info("saving file", "path", path, "size", len(content))
	return os.WriteFile(path, content, 0o644)
}
