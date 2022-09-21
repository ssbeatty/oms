package task

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"time"
)

func NewSyncBuffer(fd *os.File) *syncBuffer {

	buf := &syncBuffer{
		fd:   fd,
		quit: make(chan struct{}, 1),
	}

	go buf.flushComboOutput()

	return buf
}

type syncBuffer struct {
	bytes.Buffer
	fd   *os.File
	quit chan struct{}
	mu   sync.Mutex
}

func (w *syncBuffer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.Buffer.Write(p)
}

func (w *syncBuffer) WriteWithMsg(output []byte, msg string) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	_, _ = fmt.Fprintf(&w.Buffer, msg)

	return w.Buffer.Write(output)
}

func (w *syncBuffer) flush() {
	if w.Buffer.Len() != 0 {
		_, err := w.fd.Write(w.Buffer.Bytes())
		if err != nil {
			return
		}
		w.Buffer.Reset()
	}
}

func (w *syncBuffer) Close() {
	w.flush()

	w.quit <- struct{}{}
	defer w.fd.Close()
}

func (w *syncBuffer) flushComboOutput() {
	tick := time.NewTicker(time.Millisecond * time.Duration(120))

	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			w.flush()
		case <-w.quit:
			return
		}
	}
}
