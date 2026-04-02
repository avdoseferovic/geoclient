package main

type storedInventoryPos struct {
	Page int `json:"page"`
	X    int `json:"x"`
	Y    int `json:"y"`
}

type inventoryLayoutFile struct {
	Positions map[int]storedInventoryPos `json:"positions"`
}
