// Package eventloop runs the asynchronous work of programs outside the
// browser: work happens in other goroutines, and what it finishes with runs
// on the loop's goroutine, the only one that touches the VM.
package eventloop

import (
	"math"
	"strings"
	"time"
)

type Loop struct {
	tasks         chan func()
	pending       int
	restricted    bool
	allowedHosts  map[string]bool
	maxFetchBytes int64
}

func New() *Loop {
	return &Loop{tasks: make(chan func(), 64)}
}

// NewRestricted creates a loop for embedded scripts. Network requests are
// denied until the embedding application allows specific hosts.
func NewRestricted() *Loop {
	l := New()
	l.restricted = true
	l.maxFetchBytes = 1 << 20
	return l
}

// AllowFetchHosts replaces the exact hostnames embedded scripts may contact.
// An empty list disables fetch.
func (l *Loop) AllowFetchHosts(hosts ...string) {
	l.allowedHosts = make(map[string]bool, len(hosts))
	for _, host := range hosts {
		l.allowedHosts[strings.ToLower(strings.TrimSpace(host))] = true
	}
}

// SetMaxFetchBytes limits the response body of an embedded fetch.
func (l *Loop) SetMaxFetchBytes(n int64) {
	if n > 0 {
		if n == math.MaxInt64 {
			n--
		}
		l.maxFetchBytes = n
	}
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
		var fn func()
		select {
		case fn = <-l.tasks:
		default:
			// Nothing is ready, so the program waits
			if BeforeWait != nil {
				BeforeWait()
			}
			fn = <-l.tasks
		}
		l.pending--
		fn()
	}
	return true
}

// BeforeWait, when set, runs before the loop waits for work that isn't
// finished, as a program that prints while it serves does to show it.
var BeforeWait func()
