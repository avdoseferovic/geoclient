package gfx

import (
	"image"
	"image/color"
	"testing"
)

func TestApplyTransparencyHatClearsBothBackgroundKeys(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 8, G: 0, B: 0, A: 255})
	img.SetNRGBA(2, 0, color.NRGBA{R: 8, G: 8, B: 8, A: 255})

	applyTransparency(img, 15)

	if got := img.NRGBAAt(0, 0).A; got != 0 {
		t.Fatalf("black background alpha = %d, want 0", got)
	}
	if got := img.NRGBAAt(1, 0).A; got != 0 {
		t.Fatalf("8,0,0 background alpha = %d, want 0", got)
	}
	if got := img.NRGBAAt(2, 0).A; got != 255 {
		t.Fatalf("visible dark pixel alpha = %d, want 255", got)
	}
}

func TestApplyTransparencyNonHatKeepsDarkRed(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 8, G: 0, B: 0, A: 255})

	applyTransparency(img, 8)

	if got := img.NRGBAAt(0, 0).A; got != 0 {
		t.Fatalf("black background alpha = %d, want 0", got)
	}
	if got := img.NRGBAAt(1, 0).A; got != 255 {
		t.Fatalf("non-hat 8,0,0 alpha = %d, want 255", got)
	}
}
