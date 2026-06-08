package sample

import "testing"

func TestAdd(t *testing.T) {
	if Add(2, 3) != 5 {
		t.Error("expected 5")
	}
	// Intentionally weak: no test for negative numbers or zero
}

func TestIsPositive(t *testing.T) {
	if !IsPositive(1) {
		t.Error("1 should be positive")
	}
	if IsPositive(-1) {
		t.Error("-1 should not be positive")
	}
	// Intentionally missing: no test for 0 (boundary)
}

func TestMax(t *testing.T) {
	if Max(3, 5) != 5 {
		t.Error("expected 5")
	}
	// Intentionally weak: only tests one direction
}

func TestDivide(t *testing.T) {
	result, err := Divide(10, 2)
	if err != nil {
		t.Fatal(err)
	}
	if result != 5 {
		t.Error("expected 5")
	}
	// Intentionally weak: no test for division by zero error
}

// No test for IsEvenAndPositive at all — should survive all mutations
