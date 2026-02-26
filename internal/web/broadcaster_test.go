package web

import (
	"strings"
	"testing"
	"time"
)

func TestLogBroadcaster_WriteAndHistory(t *testing.T) {
	b := NewLogBroadcaster()

	// Write before anyone subscribes
	b.Write([]byte("first line\n"))

	ch, history := b.Subscribe()
	if history != "first line\n" {
		t.Errorf("Expected history to be 'first line\\n', got %q", history)
	}

	// Write after subscribe
	b.Write([]byte("second line\n"))

	select {
	case msg := <-ch:
		if msg != "second line\n" {
			t.Errorf("Expected msg 'second line\\n', got %q", msg)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for message")
	}

	b.Unsubscribe(ch)

	// Subscribing again should get full history
	ch2, history2 := b.Subscribe()
	if history2 != "first line\nsecond line\n" {
		t.Errorf("Expected full history, got %q", history2)
	}
	b.Unsubscribe(ch2)
}

func TestLogBroadcaster_Close(t *testing.T) {
	b := NewLogBroadcaster()
	ch, _ := b.Subscribe()

	b.Write([]byte("data"))

	// read the data
	<-ch

	b.Close()

	// Write after close should not panic or error ideally
	b.Write([]byte("ignored"))

	// The channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("Expected channel to be closed")
	}

	// Subscribing to a closed broadcaster
	ch2, history := b.Subscribe()
	if !strings.Contains(history, "data") {
		t.Errorf("Expected history to contain 'data', got %q", history)
	}

	_, ok = <-ch2
	if ok {
		t.Error("Expected new channel from closed broadcaster to be closed")
	}
}
