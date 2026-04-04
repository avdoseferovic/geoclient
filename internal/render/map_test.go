package render

import "testing"

func TestTileLayerOffsetMatchesReference(t *testing.T) {
	tests := []struct {
		name         string
		layer        int
		imgW, imgH   float64
		wantX, wantY float64
	}{
		{name: "objects", layer: 1, imgW: 64, imgH: 96, wantX: -2, wantY: -66},
		{name: "down wall", layer: 3, imgW: 64, imgH: 64, wantX: 0, wantY: -33},
		{name: "right wall", layer: 4, imgW: 64, imgH: 64, wantX: 32, wantY: -33},
		{name: "roof", layer: 5, imgW: 64, imgH: 64, wantX: 0, wantY: -64},
		{name: "top", layer: 6, imgW: 64, imgH: 64, wantX: 0, wantY: -32},
		{name: "shadow", layer: 7, imgW: 64, imgH: 32, wantX: -24, wantY: -12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tileLayerOffset(tt.layer, tt.imgW, tt.imgH)
			if got.x != tt.wantX || got.y != tt.wantY {
				t.Fatalf("tileLayerOffset(%d, %.0f, %.0f) = (%v, %v), want (%v, %v)", tt.layer, tt.imgW, tt.imgH, got.x, got.y, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestTileDrawPositionMatchesReferenceAnchors(t *testing.T) {
	sx, sy := tileDrawPosition(4, 10, 12, 64, 64, 0, 0, 0, 0)
	if sx != -64 {
		t.Fatalf("right wall sx = %v, want -64", sx)
	}
	if sy != 303 {
		t.Fatalf("right wall sy = %v, want 303", sy)
	}

	sx, sy = tileDrawPosition(5, 10, 12, 64, 64, 0, 0, 0, 0)
	if sx != -96 {
		t.Fatalf("roof sx = %v, want -96", sx)
	}
	if sy != 272 {
		t.Fatalf("roof sy = %v, want 272", sy)
	}
}

func TestLayerDepthMatchesReferenceOrdering(t *testing.T) {
	if got := layerDepth(4, 10, 12); got != 0.01100099 {
		t.Fatalf("layerDepth(right wall) = %.8f, want 0.01100099", got)
	}
	if got := layerDepth(1, 10, 12); got != 0.01200094 {
		t.Fatalf("layerDepth(objects) = %.8f, want 0.01200094", got)
	}
}
