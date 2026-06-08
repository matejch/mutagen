package sample

import "errors"

// Add returns the sum of a and b.
func Add(a, b int) int {
	return a + b
}

// IsPositive returns true if n > 0.
func IsPositive(n int) bool {
	return n > 0
}

// Max returns the larger of a and b.
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Divide divides a by b, returning an error if b is zero.
func Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

// IsEvenAndPositive returns true if n is even and positive.
func IsEvenAndPositive(n int) bool {
	return n > 0 && n%2 == 0
}
