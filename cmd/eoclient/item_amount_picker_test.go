package main

import "testing"

func TestItemAmountPickerValueClampsToMax(t *testing.T) {
	g := &Game{}
	g.overlay.itemAmountPicker = itemAmountPickerState{
		Active: true,
		Max:    5,
		Input:  "9",
	}

	if got := g.itemAmountPickerValue(); got != 5 {
		t.Fatalf("itemAmountPickerValue() = %d, want 5", got)
	}
}

func TestItemAmountPickerValueRejectsInvalidInput(t *testing.T) {
	g := &Game{}
	g.overlay.itemAmountPicker = itemAmountPickerState{
		Active: true,
		Max:    5,
		Input:  "",
	}

	if got := g.itemAmountPickerValue(); got != 0 {
		t.Fatalf("itemAmountPickerValue() = %d, want 0", got)
	}
}
