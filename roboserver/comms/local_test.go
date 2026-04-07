package comms

import (
	"roboserver/shared/event_bus"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestBus() *LocalBus {
	eb := event_bus.NewEventBus()
	return NewLocalBus(eb, nil)
}

func TestPublishEvent(t *testing.T) {
	bus := newTestBus()
	var received atomic.Bool

	cancel, err := bus.SubscribeEvent("test.event", func(eventType string, data any) {
		if eventType != "test.event" {
			t.Errorf("Expected test.event, got %s", eventType)
		}
		received.Store(true)
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer cancel()

	bus.PublishEvent("test.event", "hello")
	time.Sleep(50 * time.Millisecond)

	if !received.Load() {
		t.Error("Event handler was not called")
	}
}

func TestSubscribeCancel(t *testing.T) {
	bus := newTestBus()
	var count atomic.Int32

	cancel, err := bus.SubscribeEvent("cancel.test", func(eventType string, data any) {
		count.Add(1)
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	bus.PublishEvent("cancel.test", "1")
	time.Sleep(50 * time.Millisecond)

	cancel()

	bus.PublishEvent("cancel.test", "2")
	time.Sleep(50 * time.Millisecond)

	if count.Load() != 1 {
		t.Errorf("Expected 1 event after cancel, got %d", count.Load())
	}
}

func TestPublishToGroup_RoundRobin(t *testing.T) {
	bus := newTestBus()
	var counts [3]atomic.Int32

	for i := 0; i < 3; i++ {
		idx := i
		_, err := bus.SubscribeAsGroup("workers", "job.new", func(eventType string, data any) {
			counts[idx].Add(1)
		})
		if err != nil {
			t.Fatalf("SubscribeAsGroup failed: %v", err)
		}
	}

	// Publish 9 events — should be distributed round-robin
	for i := 0; i < 9; i++ {
		bus.PublishToGroup("workers", "job.new", i)
	}

	// Each worker should have received 3
	for i := range counts {
		if counts[i].Load() != 3 {
			t.Errorf("Worker %d received %d events, expected 3", i, counts[i].Load())
		}
	}
}

func TestPublishToGroup_NoSubscribers(t *testing.T) {
	bus := newTestBus()
	// Should not panic
	err := bus.PublishToGroup("empty", "job.new", "data")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestPublishToGroup_SingleSubscriber(t *testing.T) {
	bus := newTestBus()
	var count atomic.Int32

	_, err := bus.SubscribeAsGroup("solo", "task", func(eventType string, data any) {
		count.Add(1)
	})
	if err != nil {
		t.Fatalf("SubscribeAsGroup failed: %v", err)
	}

	for i := 0; i < 5; i++ {
		bus.PublishToGroup("solo", "task", i)
	}

	if count.Load() != 5 {
		t.Errorf("Expected 5, got %d", count.Load())
	}
}

func TestPublishToGroup_Concurrent(t *testing.T) {
	bus := newTestBus()
	var total atomic.Int32

	for i := 0; i < 4; i++ {
		_, err := bus.SubscribeAsGroup("concurrent", "work", func(eventType string, data any) {
			total.Add(1)
		})
		if err != nil {
			t.Fatalf("SubscribeAsGroup failed: %v", err)
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			bus.PublishToGroup("concurrent", "work", n)
		}(i)
	}
	wg.Wait()

	if total.Load() != 100 {
		t.Errorf("Expected 100 total events, got %d", total.Load())
	}
}
