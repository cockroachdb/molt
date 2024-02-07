package utils

import (
	"bytes"
	"io"
	"sync"
)

// The code in pipe.go is adapted from
// https://cs.opensource.google/go/x/net/+/master:http2/pipe.go

type Pipe struct {
	b   *bytes.Buffer
	c   sync.Cond
	m   sync.Mutex
	err error
}

func NewPipe(b *bytes.Buffer) *Pipe {
	p := &Pipe{
		b: b,
	}
	p.c.L = &p.m
	return p
}

// Read from the buffer into b. It blocks if the buffer is empty
// and waits for the p.c.Signal call from write to
// wake and check the condition again.
func (p *Pipe) Read(b []byte) (int, error) {
	p.c.L.Lock()
	defer p.c.L.Unlock()
	for p.b.Len() == 0 {
		// Only allow reads if the pipe is not closed.
		if p.err != nil {
			return 0, p.err
		}
		p.c.Wait()
	}
	return p.b.Read(b)
}

// Write copies bytes from b into the buffer and sends
// a singnal to wake the reader that is in a wait state.
func (p *Pipe) Write(b []byte) (int, error) {
	p.c.L.Lock()
	defer p.c.L.Unlock()
	// If you atttempt a write on a closed pipe,
	// ensure an error is returned.
	if p.err != nil {
		return 0, io.ErrClosedPipe
	}
	defer p.c.Signal()
	return p.b.Write(b)
}

func (p *Pipe) Close() error {
	return p.CloseWithError(nil)
}

func (p *Pipe) CloseWithError(err error) error {
	if err == nil {
		// Set an error signifying we are done.
		err = io.EOF
	}

	p.c.L.Lock()
	defer p.c.L.Unlock()
	if p.err != nil {
		// If we already closed with an error,
		// just simply return that same error back.
		return p.err
	}

	defer p.c.Signal()
	p.err = err
	return nil
}
