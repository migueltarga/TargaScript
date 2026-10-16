package main

import (
	"bytes"
	"testing"
)

func TestStrip(t *testing.T) {
	custom := []byte{0, 5, 4, 'n', 'a', 'm', 'e'}
	typeSection := []byte{1, 4, 1, 0x60, 0, 0}

	input := append(append(append([]byte{}, magic...), custom...), typeSection...)
	output, err := strip(input)
	if err != nil {
		t.Fatal(err)
	}

	expected := append(append([]byte{}, magic...), typeSection...)
	if !bytes.Equal(output, expected) {
		t.Errorf("got %v, want %v", output, expected)
	}

	if _, err := strip([]byte("not wasm")); err == nil {
		t.Error("expected an error for a non-WebAssembly file")
	}
	if _, err := strip(append(append([]byte{}, magic...), 1, 9, 0)); err == nil {
		t.Error("expected an error for a truncated section")
	}
}
