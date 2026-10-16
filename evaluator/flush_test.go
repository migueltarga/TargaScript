package evaluator

import (
	"bytes"
	"testing"
)

type flushingWriter struct {
	bytes.Buffer
	flushes int
}

func (w *flushingWriter) Flush() error {
	w.flushes++
	return nil
}

// FlushOutput flushes the writer print uses when it keeps output
func TestFlushOutput(t *testing.T) {
	previous := Output()
	defer SetOutput(previous)

	w := &flushingWriter{}
	SetOutput(w)
	FlushOutput()
	if w.flushes != 1 {
		t.Errorf("flushed %d times, want 1", w.flushes)
	}
	SetOutput(&bytes.Buffer{})
	FlushOutput()
}
