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

func TestOpenTradeAmountPickerConfiguresTradeAction(t *testing.T) {
	g := &Game{}

	g.openTradeAmountPicker(200, 7)

	if !g.overlay.itemAmountPicker.Active {
		t.Fatal("itemAmountPicker.Active = false, want true")
	}
	if g.overlay.itemAmountPicker.Action != itemAmountActionTradeAdd {
		t.Fatalf("itemAmountPicker.Action = %d, want %d", g.overlay.itemAmountPicker.Action, itemAmountActionTradeAdd)
	}
	if g.overlay.itemAmountPicker.ItemID != 200 {
		t.Fatalf("itemAmountPicker.ItemID = %d, want 200", g.overlay.itemAmountPicker.ItemID)
	}
	if g.overlay.itemAmountPicker.Max != 7 {
		t.Fatalf("itemAmountPicker.Max = %d, want 7", g.overlay.itemAmountPicker.Max)
	}
}
