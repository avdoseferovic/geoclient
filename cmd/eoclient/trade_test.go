package main

import (
	"image"
	"testing"
)

func TestTradeColumnRectsSplitDialogIntoTwoColumns(t *testing.T) {
	g := &Game{}
	rect := image.Rect(100, 80, 440, 340)

	left, right := g.tradeColumnRects(rect)

	if left.Min.X != 110 || left.Min.Y != 108 {
		t.Fatalf("left min = %v, want (110,108)", left.Min)
	}
	if right.Max.X != 430 || right.Max.Y != 300 {
		t.Fatalf("right max = %v, want (430,300)", right.Max)
	}
	if left.Max.X >= right.Min.X {
		t.Fatalf("columns overlap: left=%v right=%v", left, right)
	}
}
