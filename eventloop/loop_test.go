package eventloop

import "testing"

func TestRunsTasksInOrderOfCompletion(t *testing.T) {
	l := New()
	order := []int{}
	l.After(30, func() { order = append(order, 3) })
	l.After(10, func() { order = append(order, 1) })
	l.Go(func() func() {
		return func() {
			order = append(order, 2)
			l.After(40, func() { order = append(order, 4) })
		}
	})

	if !l.Run(func() bool { return len(order) == 4 }) {
		t.Fatal("stopped before all tasks ran")
	}
	if order[0] != 2 || order[1] != 1 || order[2] != 3 || order[3] != 4 {
		t.Errorf("wrong order: %v", order)
	}
}

func TestRunStopsWhenNothingIsPending(t *testing.T) {
	l := New()
	if l.Run(func() bool { return false }) {
		t.Error("reported done with nothing to wait for")
	}
}

func TestPostRunsWorkFromOtherGoroutines(t *testing.T) {
	l := New()
	l.Hold()
	ran := 0
	go func() {
		for range 3 {
			l.Post(func() { ran++ })
		}
	}()
	if !l.Run(func() bool { return ran == 3 }) {
		t.Fatal("stopped before the posted work ran")
	}
}
