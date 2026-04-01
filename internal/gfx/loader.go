package gfx

import (
	"fmt"
	"image"
	"os"
	"sync"

	ebimg "github.com/hajimehoshi/ebiten/v2"
)

const lruMaxSize = 500

// Loader loads and caches sprites from EGF files.
type Loader struct {
	dataDir string

	mu   sync.Mutex
	egfs map[int]*PEReader

	cacheMu  sync.Mutex
	cache    map[string]*ebimg.Image
	cacheSeq []string // LRU order, most recent at end
}

// NewLoader creates a GFX loader that reads .egf files from dataDir.
func NewLoader(dataDir string) *Loader {
	return &Loader{
		dataDir: dataDir,
		egfs:    make(map[int]*PEReader),
		cache:   make(map[string]*ebimg.Image),
	}
}

// LoadEGF loads and parses an EGF file, caching the PE reader.
func (l *Loader) LoadEGF(fileID int) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.egfs[fileID]; ok {
		return nil
	}

	path := fmt.Sprintf("%s/gfx%03d.egf", l.dataDir, fileID)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read egf %d: %w", fileID, err)
	}

	pe, err := NewPEReader(data)
	if err != nil {
		return fmt.Errorf("parse egf %d: %w", fileID, err)
	}

	l.egfs[fileID] = pe
	return nil
}

// GetImage loads a sprite from an EGF file. Returns nil if not found.
// Resource IDs in EGF files are offset by +100 from the game's graphic IDs.
func (l *Loader) GetImage(fileID, resourceID int) (*ebimg.Image, error) {
	resourceID += 100 // EGF resource offset
	key := fmt.Sprintf("%d:%d", fileID, resourceID)

	l.cacheMu.Lock()
	if img, ok := l.cache[key]; ok {
		l.touchLRU(key)
		l.cacheMu.Unlock()
		return img, nil
	}
	l.cacheMu.Unlock()

	if err := l.LoadEGF(fileID); err != nil {
		return nil, err
	}

	l.mu.Lock()
	pe := l.egfs[fileID]
	l.mu.Unlock()

	info, ok := pe.Resources[resourceID]
	if !ok {
		return nil, nil
	}

	rawData := pe.ResourceData(info)
	nrgba, err := ReadDIB(rawData, fileID)
	if err != nil {
		return nil, fmt.Errorf("decode dib %d:%d: %w", fileID, resourceID, err)
	}

	img := nrgbaToEbiten(nrgba)

	l.cacheMu.Lock()
	l.evictLRU()
	l.cache[key] = img
	l.cacheSeq = append(l.cacheSeq, key)
	l.cacheMu.Unlock()

	return img, nil
}

// GetRawImage loads a sprite as a Go image.NRGBA (no Ebitengine dependency).
func (l *Loader) GetRawImage(fileID, resourceID int) (*image.NRGBA, error) {
	resourceID += 100 // EGF resource offset
	if err := l.LoadEGF(fileID); err != nil {
		return nil, err
	}

	l.mu.Lock()
	pe := l.egfs[fileID]
	l.mu.Unlock()

	info, ok := pe.Resources[resourceID]
	if !ok {
		return nil, nil
	}

	return ReadDIB(pe.ResourceData(info), fileID)
}

func (l *Loader) touchLRU(key string) {
	for i, k := range l.cacheSeq {
		if k == key {
			l.cacheSeq = append(l.cacheSeq[:i], l.cacheSeq[i+1:]...)
			l.cacheSeq = append(l.cacheSeq, key)
			return
		}
	}
}

func (l *Loader) evictLRU() {
	for len(l.cache) >= lruMaxSize && len(l.cacheSeq) > 0 {
		oldest := l.cacheSeq[0]
		l.cacheSeq = l.cacheSeq[1:]
		delete(l.cache, oldest)
	}
}

func nrgbaToEbiten(img *image.NRGBA) *ebimg.Image {
	return ebimg.NewImageFromImage(img)
}
