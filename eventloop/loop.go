// Package eventloop runs the asynchronous work of programs outside the
// browser: work happens in other goroutines, and what it finishes with runs
// on the loop's goroutine, the only one that touches the VM.
package eventloop

import "time"

type Loop struct {
	tasks   chan func()
	pending int
}

func New() *Loop {
	return &Loop{tasks: make(chan func(), 64)}
}

// After runs fn on the loop after the given number of milliseconds.
func (l *Loop) After(milliseconds int64, fn func()) {
	l.pending++
	go func() {
		time.Sleep(time.Duration(milliseconds) * time.Millisecond)
		l.tasks <- fn
	}()
}

// Go runs work in another goroutine, then runs the function it returns on
// the loop.
func (l *Loop) Go(work func() func()) {
	l.pending++
	go func() {
		l.tasks <- work()
	}()
}

// Hold keeps Run going for work with no end of its own, like a server, so
// Post can be used. Call it on the loop.
func (l *Loop) Hold() {
	l.pending++
}

// Post runs fn on the loop. Unlike the other methods it can be called from
// any goroutine, while something holds the loop.
func (l *Loop) Post(fn func()) {
	l.tasks <- func() {
		// Run counts this as finished work; it wasn't pending
		l.pending++
		fn()
	}
}

// Run runs finished work until done returns true or nothing is pending. It
// reports whether done returned true.
func (l *Loop) Run(done func() bool) bool {
	for !done() {
		if l.pending == 0 {
			return false
		}
		fn := <-l.tasks
		l.pending--
		fn()
	}
	return true
}
