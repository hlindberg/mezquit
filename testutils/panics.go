package testutils

import "testing"

// ShouldNotPanic is used to assert that a function does not panic
func ShouldNotPanic(t *testing.T) {
	t.Helper()
	if r := recover(); r != nil {
		t.Errorf("Unexpected panic")
	}
}

// ShouldPanic is used to assert that a function does panic
func ShouldPanic(t *testing.T) {
	t.Helper()
	if r := recover(); r == nil {
		t.Errorf("Expected panic")
	}
}
