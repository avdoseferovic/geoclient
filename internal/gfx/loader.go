package gfx

import (
	"container/list"
	"fmt"
	"image"
	"path/filepath"
	"sync"

	"github.com/avdo/eoweb/internal/assets"
	ebimg "github.com/hajimehoshi/ebiten/v2"
)

const lruMaxSize = 500

type cacheKey struct {
	fileID     int
	resourceID int
}

// Loader loads and caches sprites from EGF files.
type Loader struct {
	dataDir string
	reader  assets.Reader

	mu   sync.Mutex
	egfs map[int]*PEReader

	cacheMu  sync.Mutex
	cache    map[cacheKey]*ebimg.Image
	lruList  *list.List                 // front = most recent, back = least recent
	lruIndex map[cacheKey]*list.Element // key -> list element
}

// NewLoader creates a GFX loader that reads .egf files from dataDir.
func NewLoader(dataDir string) *Loader {
	return NewLoaderWithReader(dataDir, assets.NewOSReader())
}

func NewLoaderWithReader(dataDir string, reader assets.Reader) *Loader {
	return &Loader{
		dataDir:  dataDir,
		reader:   reader,
		egfs:     make(map[int]*PEReader),
		cache:    make(map[cacheKey]*ebimg.Image),
		lruList:  list.New(),
		lruIndex: make(map[cacheKey]*list.Element),
	}
}

// LoadEGF loads and parses an EGF file, caching the PE reader.
func (l *Loader) LoadEGF(fileID int) error {
	l.mu.Lock()
	if _, ok := l.egfs[fileID]; ok {
		l.mu.Unlock()
		return nil
	}
	l.mu.Unlock()

	// Perform I/O outside the lock
	path := filepath.Join(l.dataDir, fmt.Sprintf("gfx%03d.egf", fileID))
	data, err := l.reader.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read egf %d: %w", fileID, err)
	}

	pe, err := NewPEReader(data)
	if err != nil {
		return fmt.Errorf("parse egf %d: %w", fileID, err)
	}

	// Re-acquire lock and store (double-check pattern)
	l.mu.Lock()
	if _, ok := l.egfs[fileID]; !ok {
		l.egfs[fileID] = pe
	}
	l.mu.Unlock()
	return nil
}

// GetImage loads a sprite from an EGF file. Returns nil if not found.
// Resource IDs in EGF files are offset by +100 from the game's graphic IDs.
func (l *Loader) GetImage(fileID, resourceID int) (*ebimg.Image, error) {
	resourceID += 100 // EGF resource offset
	key := cacheKey{fileID, resourceID}

	l.cacheMu.Lock()
	if img, ok := l.cache[key]; ok {
		// O(1) LRU touch: move to front
		if elem, ok := l.lruIndex[key]; ok {
			l.lruList.MoveToFront(elem)
		}
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
	// Check if another goroutine already cached this
	if existing, ok := l.cache[key]; ok {
		l.cacheMu.Unlock()
		return existing, nil
	}
	l.evictLRU()
	l.cache[key] = img
	elem := l.lruList.PushFront(key)
	l.lruIndex[key] = elem
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

func (l *Loader) evictLRU() {
	for len(l.cache) >= lruMaxSize && l.lruList.Len() > 0 {
		back := l.lruList.Back()
		if back == nil {
			break
		}
		key := back.Value.(cacheKey)
		l.lruList.Remove(back)
		delete(l.lruIndex, key)
		delete(l.cache, key)
	}
}

func nrgbaToEbiten(img *image.NRGBA) *ebimg.Image {
	return ebimg.NewImageFromImage(img)
}
