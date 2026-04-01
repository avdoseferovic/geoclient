package gfx

import (
	"encoding/binary"
	"fmt"
	"math"
)

const resourceTypeBitmap = 2

// ResourceInfo describes a bitmap resource inside a PE/EGF file.
type ResourceInfo struct {
	Start  int
	Size   int
	Width  int
	Height int
}

// PEReader parses PE executables (.egf files) to extract bitmap resource metadata.
type PEReader struct {
	data        []byte
	resourceRVA uint32 // virtual address of resource section
	resourceRaw uint32 // raw file offset of resource section
	Resources   map[int]ResourceInfo
}

// NewPEReader parses a PE file and indexes its bitmap resources.
func NewPEReader(data []byte) (*PEReader, error) {
	p := &PEReader{data: data, Resources: make(map[int]ResourceInfo)}
	if err := p.readHeader(); err != nil {
		return nil, err
	}
	if err := p.readBitmapTable(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *PEReader) u16(pos int) uint16 {
	return binary.LittleEndian.Uint16(p.data[pos:])
}

func (p *PEReader) u32(pos int) uint32 {
	return binary.LittleEndian.Uint32(p.data[pos:])
}

func (p *PEReader) i32(pos int) int32 {
	return int32(binary.LittleEndian.Uint32(p.data[pos:]))
}

func (p *PEReader) readHeader() error {
	if len(p.data) < 0x40 {
		return fmt.Errorf("file too small")
	}

	// PE header offset at 0x3C
	peOff := int(p.u32(0x3C))
	if peOff+4 > len(p.data) || string(p.data[peOff:peOff+4]) != "PE\x00\x00" {
		return fmt.Errorf("invalid PE signature")
	}

	// COFF header: peOff+4
	coffOff := peOff + 4
	numSections := int(p.u16(coffOff + 2))
	optHeaderSize := int(p.u16(coffOff + 16))

	// Optional header: coffOff+20
	optOff := coffOff + 20
	if optOff+optHeaderSize > len(p.data) {
		return fmt.Errorf("optional header truncated")
	}

	// Resource directory is data directory entry #2
	// Data directories start at optOff+96 for PE32 (magic 0x10b)
	// or optOff+112 for PE32+ (magic 0x20b)
	magic := p.u16(optOff)
	var ddOff int
	switch magic {
	case 0x10b: // PE32
		ddOff = optOff + 96
	case 0x20b: // PE32+
		ddOff = optOff + 112
	default:
		return fmt.Errorf("unknown optional header magic: 0x%x", magic)
	}

	// Entry #2 = resource table (each entry is 8 bytes: RVA + Size)
	resEntryOff := ddOff + 2*8
	if resEntryOff+8 > len(p.data) {
		return fmt.Errorf("resource data directory entry truncated")
	}
	p.resourceRVA = p.u32(resEntryOff)
	if p.resourceRVA == 0 {
		return fmt.Errorf("no resource directory")
	}

	// Section headers start after optional header
	secOff := optOff + optHeaderSize
	for i := 0; i < numSections; i++ {
		off := secOff + i*40
		if off+40 > len(p.data) {
			break
		}
		sectionVA := p.u32(off + 12)
		sectionRawPtr := p.u32(off + 20)
		if sectionVA == p.resourceRVA {
			p.resourceRaw = sectionRawPtr
			break
		}
	}

	if p.resourceRaw == 0 {
		return fmt.Errorf("resource section not found")
	}

	return nil
}

func (p *PEReader) readDirectoryEntryCount(pos int) (int, int) {
	named := int(p.u16(pos + 12))
	id := int(p.u16(pos + 14))
	return named + id, pos + 16
}

func (p *PEReader) readBitmapTable() error {
	root := int(p.resourceRaw)

	count, pos := p.readDirectoryEntryCount(root)
	var bitmapSubdirOffset uint32
	for i := 0; i < count; i++ {
		resType := p.u32(pos)
		subdirOff := p.u32(pos + 4)
		if resType == resourceTypeBitmap {
			bitmapSubdirOffset = subdirOff & 0x7FFFFFFF
			break
		}
		pos += 8
	}

	if bitmapSubdirOffset == 0 {
		return fmt.Errorf("no bitmap resource directory found")
	}

	bitmapDir := root + int(bitmapSubdirOffset)
	count, pos = p.readDirectoryEntryCount(bitmapDir)

	type entry struct {
		id     int
		offset uint32
	}
	var entries []entry

	for i := 0; i < count; i++ {
		resID := p.u32(pos)
		subdirOff := p.u32(pos + 4)
		if subdirOff > 0x80000000 {
			entries = append(entries, entry{int(resID), subdirOff & 0x7FFFFFFF})
		}
		pos += 8
	}

	for _, e := range entries {
		// Language subdirectory
		langDirPos := root + int(e.offset)
		_, langEntryPos := p.readDirectoryEntryCount(langDirPos)
		if langEntryPos+8 > len(p.data) {
			continue
		}
		dataEntryOff := p.u32(langEntryPos + 4)
		// Data entries are NOT subdirectories (high bit not set)
		if dataEntryOff > 0x80000000 {
			dataEntryOff &= 0x7FFFFFFF
		}

		dataPos := root + int(dataEntryOff)
		if dataPos+16 > len(p.data) {
			continue
		}
		dataRVA := p.u32(dataPos)
		dataSize := p.u32(dataPos + 4)

		// Convert RVA to file offset
		start := int(dataRVA) - int(p.resourceRVA) + root
		size := int(dataSize)

		if start < 0 || start+4 > len(p.data) {
			continue
		}

		headerSize := p.u32(start)
		var w, h int
		if headerSize == 12 {
			w = int(p.u16(start + 4))
			h = int(p.u16(start + 6))
		} else {
			w = int(math.Abs(float64(p.i32(start + 4))))
			h = int(math.Abs(float64(p.i32(start + 8))))
		}

		p.Resources[e.id] = ResourceInfo{Start: start, Size: size, Width: w, Height: h}
	}

	return nil
}

// ResourceData returns the raw bytes of a resource.
func (p *PEReader) ResourceData(info ResourceInfo) []byte {
	end := info.Start + info.Size
	if end > len(p.data) {
		end = len(p.data)
	}
	return p.data[info.Start:end]
}
