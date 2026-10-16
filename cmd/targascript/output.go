package main

import (
	"bufio"
	"io"
	"os"
	"os/signal"
	"sync"

	"golang.org/x/term"

	"github.com/migueltarga/TargaScript/evaluator"
	"github.com/migueltarga/TargaScript/eventloop"
)

// bufferedOutput keeps what a script prints to write it in large pieces,
// for a file or a pipe, where a write for every line is slow.
type bufferedOutput struct {
	mu sync.Mutex
	w  *bufio.Writer
}

func (o *bufferedOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.w.Write(p)
}

func (o *bufferedOutput) Flush() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.w.Flush()
}

var (
	stdout     io.Writer
	stdoutOnce sync.Once
)

// scriptOutput returns where scripts print: the terminal as it is, or
// standard output with a buffer, written out when the script waits, reads
// input, runs a command, ends or is interrupted.
func scriptOutput() io.Writer {
	stdoutOnce.Do(func() {
		if term.IsTerminal(int(os.Stdout.Fd())) {
			stdout = os.Stdout
			return
		}
		stdout = &bufferedOutput{w: bufio.NewWriterSize(os.Stdout, 32*1024)}
		eventloop.BeforeWait = evaluator.FlushOutput
		interrupted := make(chan os.Signal, 1)
		signal.Notify(interrupted, os.Interrupt)
		go func() {
			<-interrupted
			stdout.(*bufferedOutput).Flush()
			os.Exit(130)
		}()
	})
	return stdout
}
