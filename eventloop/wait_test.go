package eventloop

import "testing"

// BeforeWait runs when the loop has to wait, not when work is ready
func TestBeforeWait(t *testing.T) {
	waits := 0
	BeforeWait = func() { waits++ }
	defer func() { BeforeWait = nil }()

	l := New()
	ran := 0
	l.After(20, func() { ran++ })
	l.Run(func() bool { return ran == 1 })
	if ran != 1 || waits != 1 {
		t.Fatalf("ran %d, waited %d times, want 1 and 1", ran, waits)
	}

	l.Hold()
	l.Post(func() { ran++ })
	// Post put the work there before the loop looks
	waits = 0
	l.Run(func() bool { return ran == 2 })
	if waits != 0 {
		t.Errorf("waited %d times for work that was ready", waits)
	}
}
