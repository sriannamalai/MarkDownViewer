package main

import (
	"errors"
	"testing"
)

func TestPanicError(t *testing.T) {
	if got := panicError("boom").Error(); got != "panic: boom" {
		t.Errorf("panicError(string) = %q", got)
	}
	wrapped := errors.New("boom")
	err := panicError(wrapped)
	if !errors.Is(err, wrapped) {
		t.Errorf("panicError(error) = %v, does not wrap the panic value", err)
	}
}
