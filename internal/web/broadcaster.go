package web

import (
	"bytes"
	"sync"
)

type LogBroadcaster struct {
	mu          sync.RWMutex
	history     bytes.Buffer
	subscribers map[chan string]struct{}
	closed      bool
}

func NewLogBroadcaster() *LogBroadcaster {
	return &LogBroadcaster{
		subscribers: make(map[chan string]struct{}),
	}
}

func (b *LogBroadcaster) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return len(p), nil // or return error if we prefer
	}

	n, err = b.history.Write(p)
	chunk := string(p)

	for ch := range b.subscribers {
		select {
		case ch <- chunk:
		default:
			// Client too slow, skip chunk
		}
	}
	return n, err
}

func (b *LogBroadcaster) Subscribe() (chan string, string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan string, 100)

	// If it's already closed, we just return the history and a closed channel
	if b.closed {
		close(ch)
		return ch, b.history.String()
	}

	b.subscribers[ch] = struct{}{}
	return ch, b.history.String()
}

func (b *LogBroadcaster) Unsubscribe(ch chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.subscribers[ch]; ok {
		delete(b.subscribers, ch)
		close(ch)
	}
}

func (b *LogBroadcaster) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}
	b.closed = true
	for ch := range b.subscribers {
		close(ch)
	}
	b.subscribers = nil
}

func (b *LogBroadcaster) String() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.history.String()
}
