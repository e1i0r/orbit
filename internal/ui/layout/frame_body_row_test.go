package layout

import (
	"testing"
)

func TestFrameAtAndBodyRow(t *testing.T) {
	frame, err := Fit(80, 24)
	if err != nil {
		t.Fatalf("Fit(80, 24) failed: %v", err)
	}

	// Test RegionHeader
	if reg := frame.At(0); reg != RegionHeader {
		t.Errorf("At(0) = %v, want %v", reg, RegionHeader)
	}

	// Test RegionBody
	bodyY := frame.Body.Y
	if reg := frame.At(bodyY); reg != RegionBody {
		t.Errorf("At(%d) = %v, want %v", bodyY, reg, RegionBody)
	}

	// Test BodyRow within body
	row, ok := frame.BodyRow(bodyY)
	if !ok || row != 0 {
		t.Errorf("BodyRow(%d) = (%d, %v), want (0, true)", bodyY, row, ok)
	}

	row, ok = frame.BodyRow(bodyY + 2)
	if !ok || row != 2 {
		t.Errorf("BodyRow(%d) = (%d, %v), want (2, true)", bodyY+2, row, ok)
	}

	// Test BodyRow outside body (e.g. Header row 0)
	if _, ok := frame.BodyRow(0); ok {
		t.Error("BodyRow(0) returned true for header line")
	}

	// Test At outside screen bounds
	if reg := frame.At(-5); reg != RegionNone {
		t.Errorf("At(-5) = %v, want %v", reg, RegionNone)
	}
	if reg := frame.At(100); reg != RegionNone {
		t.Errorf("At(100) = %v, want %v", reg, RegionNone)
	}
}
