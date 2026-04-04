package gfx

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"math"
)

// Hat file IDs use (8,0,0) as transparent color instead of (0,0,0).
var hatFileIDs = map[int]bool{15: true, 16: true}

// ReadDIB decodes a DIB (Device Independent Bitmap) into an RGBA image.
// fileID is used to determine the transparency color for hat files.
func ReadDIB(data []byte, fileID int) (*image.NRGBA, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("DIB data too short")
	}

	headerSize := le32(data, 0)
	var w, h int
	var depth uint16
	var compressionType uint32
	var topDown bool

	if headerSize == 12 {
		// BITMAPCOREHEADER
		w = int(le16(data, 4))
		h = int(le16(data, 6))
		depth = le16(data, 10)
	} else {
		w = int(math.Abs(float64(int32(le32(data, 4)))))
		rawH := int32(le32(data, 8))
		if rawH < 0 {
			topDown = true
			h = int(-rawH)
		} else {
			h = int(rawH)
		}
		depth = le16(data, 14)
		if headerSize >= 20 {
			compressionType = le32(data, 16)
		}
	}

	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("invalid dimensions: %dx%d", w, h)
	}

	// Read palette for indexed images
	var palette []color.NRGBA
	if depth <= 8 {
		paletteCount := 1 << depth
		if headerSize >= 36 {
			colorsUsed := int(le32(data, 32))
			if colorsUsed > 0 {
				paletteCount = colorsUsed
			}
		}
		bytesPerEntry := 4
		if headerSize == 12 {
			bytesPerEntry = 3
		}
		paletteStart := int(headerSize)
		palette = make([]color.NRGBA, paletteCount)
		for i := range palette {
			off := paletteStart + i*bytesPerEntry
			if off+3 > len(data) {
				break
			}
			palette[i] = color.NRGBA{R: data[off+2], G: data[off+1], B: data[off], A: 255}
		}
	}

	// Calculate pixel data start
	pixelStart := int(headerSize)
	if depth <= 8 {
		bytesPerEntry := 4
		if headerSize == 12 {
			bytesPerEntry = 3
		}
		paletteCount := len(palette)
		pixelStart += paletteCount * bytesPerEntry

		// Handle optional bitmasks for BITMAPINFOHEADER with bitfields
		if headerSize == 40 && (compressionType == 3 || compressionType == 6) {
			optSize := 12
			if compressionType == 6 {
				optSize = 16
			}
			pixelStart += optSize
		}
	} else {
		// For 16/24/32 bit, pixel data follows header + optional bitmasks
		if headerSize == 40 && (compressionType == 3 || compressionType == 6) {
			optSize := 12
			if compressionType == 6 {
				optSize = 16
			}
			pixelStart += optSize
		}
	}

	stride := ((w*int(depth) + 31) & ^31) >> 3

	img := image.NewNRGBA(image.Rect(0, 0, w, h))

	switch {
	case compressionType == 1 && depth == 8:
		decodeRLE8(data, pixelStart, w, h, topDown, palette, img)
	case compressionType == 0 && depth <= 8:
		decodePaletted(data, pixelStart, stride, w, h, int(depth), topDown, palette, img)
	case compressionType == 0 || compressionType == 3:
		decodeRGB(data, pixelStart, stride, w, h, int(depth), topDown, headerSize, compressionType, img)
	default:
		return nil, fmt.Errorf("unsupported compression %d with depth %d", compressionType, depth)
	}

	// Apply transparency: (0,0,0) or (8,0,0) for hat files
	applyTransparency(img, fileID)

	return img, nil
}

func decodePaletted(data []byte, pixelStart, stride, w, h, depth int, topDown bool, palette []color.NRGBA, img *image.NRGBA) {
	pixelsPerByte := 8 / depth
	mask := (1 << depth) - 1

	for row := 0; row < h; row++ {
		line := row
		if !topDown {
			line = h - 1 - row
		}
		lineStart := pixelStart + stride*line

		x := 0
		for byteIdx := 0; byteIdx < (w+pixelsPerByte-1)/pixelsPerByte; byteIdx++ {
			pos := lineStart + byteIdx
			if pos >= len(data) {
				break
			}
			b := data[pos]
			for bit := pixelsPerByte - 1; bit >= 0 && x < w; bit-- {
				idx := int(b>>uint(bit*depth)) & mask
				if idx < len(palette) {
					img.SetNRGBA(x, row, palette[idx])
				}
				x++
			}
		}
	}
}

func decodeRGB(data []byte, pixelStart, stride, w, h, depth int, topDown bool, headerSize, compression uint32, img *image.NRGBA) {
	bytesPerPixel := depth >> 3

	// Read bitmasks for 16/32-bit with bitfields
	var rMask, gMask, bMask uint32
	if compression == 3 && len(data) >= 52 {
		// BI_BITFIELDS: masks stored right after the 40-byte header
		rMask = le32(data, 40)
		gMask = le32(data, 44)
		bMask = le32(data, 48)
	} else {
		switch depth {
		case 16:
			rMask, gMask, bMask = 0x7C00, 0x03E0, 0x001F
		case 24, 32:
			rMask, gMask, bMask = 0x00FF0000, 0x0000FF00, 0x000000FF
		}
	}

	rShift, rLen := maskShiftLen(rMask)
	gShift, gLen := maskShiftLen(gMask)
	bShift, bLen := maskShiftLen(bMask)

	for row := 0; row < h; row++ {
		line := row
		if !topDown {
			line = h - 1 - row
		}
		lineStart := pixelStart + stride*line

		for x := 0; x < w; x++ {
			pos := lineStart + x*bytesPerPixel
			if pos+bytesPerPixel > len(data) {
				break
			}

			var pixel uint32
			switch bytesPerPixel {
			case 2:
				pixel = uint32(le16(data, pos))
			case 3:
				pixel = uint32(data[pos]) | uint32(data[pos+1])<<8 | uint32(data[pos+2])<<16
			case 4:
				pixel = le32(data, pos)
			}

			r := scaleTo8((pixel>>rShift)&((1<<rLen)-1), rLen)
			g := scaleTo8((pixel>>gShift)&((1<<gLen)-1), gLen)
			b := scaleTo8((pixel>>bShift)&((1<<bLen)-1), bLen)

			img.SetNRGBA(x, row, color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255})
		}
	}
}

func decodeRLE8(data []byte, pixelStart, w, h int, topDown bool, palette []color.NRGBA, img *image.NRGBA) {
	pos := pixelStart
	x, y := 0, 0

	getRow := func(y int) int {
		if topDown {
			return y
		}
		return h - 1 - y
	}

	for pos < len(data) && y < h {
		if pos+1 >= len(data) {
			break
		}
		count := int(data[pos])
		value := data[pos+1]
		pos += 2

		if count == 0 {
			switch value {
			case 0: // end of line
				x = 0
				y++
			case 1: // end of bitmap
				return
			case 2: // delta
				if pos+1 >= len(data) {
					return
				}
				x += int(data[pos])
				y += int(data[pos+1])
				pos += 2
			default: // absolute mode
				for i := 0; i < int(value) && pos < len(data); i++ {
					idx := int(data[pos])
					pos++
					if idx < len(palette) && x < w && y < h {
						img.SetNRGBA(x, getRow(y), palette[idx])
					}
					x++
				}
				if int(value)%2 != 0 {
					pos++ // padding
				}
			}
		} else {
			idx := int(value)
			for i := 0; i < count; i++ {
				if idx < len(palette) && x < w && y < h {
					img.SetNRGBA(x, getRow(y), palette[idx])
				}
				x++
			}
		}
	}
}

func applyTransparency(img *image.NRGBA, fileID int) {
	isHat := hatFileIDs[fileID]

	has800 := false
	if isHat {
		for i := 0; i < len(img.Pix); i += 4 {
			if img.Pix[i] == 8 && img.Pix[i+1] == 0 && img.Pix[i+2] == 0 {
				has800 = true
				break
			}
		}
	}

	for i := 0; i < len(img.Pix); i += 4 {
		r, g, b := img.Pix[i], img.Pix[i+1], img.Pix[i+2]
		if r == 0 && g == 0 && b == 0 {
			img.Pix[i+3] = 0
			continue
		}
		if isHat && has800 && r == 8 && g == 0 && b == 0 {
			img.Pix[i+3] = 0
		}
	}
}

func maskShiftLen(mask uint32) (shift, length uint32) {
	if mask == 0 {
		return 0, 0
	}
	for mask&1 == 0 {
		mask >>= 1
		shift++
	}
	for mask&1 == 1 {
		mask >>= 1
		length++
	}
	return
}

func scaleTo8(val, bits uint32) uint32 {
	if bits == 0 {
		return 0
	}
	if bits >= 8 {
		return val >> (bits - 8)
	}
	return (val * 255) / ((1 << bits) - 1)
}

func le16(data []byte, pos int) uint16 {
	return binary.LittleEndian.Uint16(data[pos:])
}

func le32(data []byte, pos int) uint32 {
	return binary.LittleEndian.Uint32(data[pos:])
}
