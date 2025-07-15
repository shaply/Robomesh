package event_bus

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Test event implementation
type TestEvent struct {
	eventType string
	data      interface{}
}

func (te *TestEvent) GetType() string {
	return te.eventType
}

func (te *TestEvent) GetData() interface{} {
	return te.data
}

// Basic functionality tests
func TestEventBusSubscribe(t *testing.T) {
	eb := NewEventBus()

	eventReceived := false
	var receivedData interface{}

	subscriber := eb.Subscribe("test_event", nil, func(event Event) {
		eventReceived = true
		receivedData = event.GetData()
	})

	if subscriber == nil {
		t.Error("Expected subscriber to be returned")
	}

	// Publish event
	eb.Publish(&TestEvent{
		eventType: "test_event",
		data:      "test_data",
	})

	// Give goroutine time to process
	time.Sleep(10 * time.Millisecond)

	if !eventReceived {
		t.Error("Expected event to be received")
	}

	if receivedData != "test_data" {
		t.Errorf("Expected 'test_data', got %v", receivedData)
	}
}

func TestEventBusUnsubscribe(t *testing.T) {
	eb := NewEventBus()

	var count int32

	subscriber := eb.Subscribe("test_event", nil, func(event Event) {
		atomic.AddInt32(&count, 1)
	})

	// Publish first event
	eb.Publish(&TestEvent{eventType: "test_event", data: "data1"})
	time.Sleep(10 * time.Millisecond)

	// Unsubscribe
	eb.Unsubscribe("test_event", subscriber)

	// Publish second event
	eb.Publish(&TestEvent{eventType: "test_event", data: "data2"})
	time.Sleep(10 * time.Millisecond)

	finalCount := atomic.LoadInt32(&count)
	if finalCount != 1 {
		t.Errorf("Expected count to be 1 after unsubscribe, got %d", finalCount)
	}
}

func TestEventBusMultipleSubscribers(t *testing.T) {
	eb := NewEventBus()

	var count1, count2, count3 int32

	eb.Subscribe("test_event", nil, func(event Event) {
		atomic.AddInt32(&count1, 1)
	})

	eb.Subscribe("test_event", nil, func(event Event) {
		atomic.AddInt32(&count2, 1)
	})

	eb.Subscribe("test_event", nil, func(event Event) {
		atomic.AddInt32(&count3, 1)
	})

	// Publish event
	eb.Publish(&TestEvent{eventType: "test_event", data: "broadcast"})
	time.Sleep(20 * time.Millisecond)

	if atomic.LoadInt32(&count1) != 1 {
		t.Errorf("Expected subscriber1 count to be 1, got %d", count1)
	}
	if atomic.LoadInt32(&count2) != 1 {
		t.Errorf("Expected subscriber2 count to be 1, got %d", count2)
	}
	if atomic.LoadInt32(&count3) != 1 {
		t.Errorf("Expected subscriber3 count to be 1, got %d", count3)
	}
}

func TestEventBusDifferentEventTypes(t *testing.T) {
	eb := NewEventBus()

	var robotCount, userCount int32

	eb.Subscribe("robot_event", nil, func(event Event) {
		atomic.AddInt32(&robotCount, 1)
	})

	eb.Subscribe("user_event", nil, func(event Event) {
		atomic.AddInt32(&userCount, 1)
	})

	// Publish different event types
	eb.Publish(&TestEvent{eventType: "robot_event", data: "robot_data"})
	eb.Publish(&TestEvent{eventType: "user_event", data: "user_data"})
	eb.Publish(&TestEvent{eventType: "robot_event", data: "robot_data2"})

	time.Sleep(20 * time.Millisecond)

	if atomic.LoadInt32(&robotCount) != 2 {
		t.Errorf("Expected robot event count to be 2, got %d", robotCount)
	}
	if atomic.LoadInt32(&userCount) != 1 {
		t.Errorf("Expected user event count to be 1, got %d", userCount)
	}
}

// Edge case tests
func TestEventBusPublishToNoSubscribers(t *testing.T) {
	eb := NewEventBus()

	// Should not panic when publishing to event with no subscribers
	eb.Publish(&TestEvent{eventType: "nonexistent_event", data: "data"})

	// Test passes if no panic occurs
}

func TestEventBusUnsubscribeNonexistent(t *testing.T) {
	eb := NewEventBus()

	// Should not panic when unsubscribing nonexistent subscriber
	fakeSubscriber := &Subscriber{ID: "fake"}
	eb.Unsubscribe("nonexistent_event", fakeSubscriber)

	// Add a real subscriber
	var count int32
	realSubscriber := eb.Subscribe("real_event", nil, func(event Event) {
		atomic.AddInt32(&count, 1)
	})

	// Unsubscribe nonexistent subscriber from real event
	eb.Unsubscribe("real_event", fakeSubscriber)

	// Real subscriber should still work
	eb.Publish(&TestEvent{eventType: "real_event", data: "data"})
	time.Sleep(10 * time.Millisecond)

	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("Expected real subscriber to still work, count: %d", count)
	}

	// Now unsubscribe real subscriber
	eb.Unsubscribe("real_event", realSubscriber)
	eb.Publish(&TestEvent{eventType: "real_event", data: "data2"})
	time.Sleep(10 * time.Millisecond)

	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("Expected count to remain 1 after real unsubscribe, got %d", count)
	}
}

func TestEventBusDuplicateSubscribers(t *testing.T) {
	eb := NewEventBus()

	var count int32
	handler := func(event Event) {
		atomic.AddInt32(&count, 1)
	}

	// Create specific subscriber
	subscriber := &Subscriber{ID: "duplicate_id"}

	// Subscribe same subscriber multiple times
	eb.Subscribe("test_event", subscriber, handler)
	eb.Subscribe("test_event", subscriber, handler)
	eb.Subscribe("test_event", subscriber, handler)

	// Publish event
	eb.Publish(&TestEvent{eventType: "test_event", data: "data"})
	time.Sleep(10 * time.Millisecond)

	// Should handle duplicates gracefully (exact behavior depends on Set implementation)
	finalCount := atomic.LoadInt32(&count)
	if finalCount < 1 {
		t.Errorf("Expected at least 1 event delivery, got %d", finalCount)
	}

	t.Logf("Duplicate subscriber resulted in %d event deliveries", finalCount)
}

// Concurrency tests
func TestEventBusConcurrentSubscribers(t *testing.T) {
	eb := NewEventBus()

	var totalCount int64
	numSubscribers := 100
	var wg sync.WaitGroup

	// Add subscribers concurrently
	for i := 0; i < numSubscribers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			eb.Subscribe("concurrent_event", nil, func(event Event) {
				atomic.AddInt64(&totalCount, 1)
			})
		}(i)
	}

	wg.Wait()

	// Publish event
	eb.Publish(&TestEvent{eventType: "concurrent_event", data: "concurrent_data"})
	time.Sleep(100 * time.Millisecond)

	finalCount := atomic.LoadInt64(&totalCount)
	if finalCount != int64(numSubscribers) {
		t.Errorf("Expected %d event deliveries, got %d", numSubscribers, finalCount)
	}
}

func TestEventBusConcurrentPublish(t *testing.T) {
	eb := NewEventBus()

	var count int64

	eb.Subscribe("publish_event", nil, func(event Event) {
		atomic.AddInt64(&count, 1)
	})

	numPublishers := 50
	eventsPerPublisher := 10
	var wg sync.WaitGroup

	// Publish concurrently
	for i := 0; i < numPublishers; i++ {
		wg.Add(1)
		go func(publisherID int) {
			defer wg.Done()
			for j := 0; j < eventsPerPublisher; j++ {
				eb.Publish(&TestEvent{
					eventType: "publish_event",
					data:      fmt.Sprintf("data_%d_%d", publisherID, j),
				})
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	expectedCount := int64(numPublishers * eventsPerPublisher)
	finalCount := atomic.LoadInt64(&count)

	if finalCount != expectedCount {
		t.Errorf("Expected %d events, got %d", expectedCount, finalCount)
	}
}

func TestEventBusConcurrentSubscribeUnsubscribe(t *testing.T) {
	eb := NewEventBus()

	var subscribeCount, unsubscribeCount int64
	var eventCount int64

	var wg sync.WaitGroup
	var subscribers []*Subscriber
	var subscribersMu sync.Mutex
	numOperations := 100

	// Concurrent subscribe operations
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			subscriber := eb.Subscribe("mixed_event", nil, func(event Event) {
				atomic.AddInt64(&eventCount, 1)
			})

			subscribersMu.Lock()
			subscribers = append(subscribers, subscriber)
			subscribersMu.Unlock()

			atomic.AddInt64(&subscribeCount, 1)

			// Sometimes unsubscribe immediately
			if id%2 == 0 {
				time.Sleep(time.Microsecond) // Small delay
				eb.Unsubscribe("mixed_event", subscriber)
				atomic.AddInt64(&unsubscribeCount, 1)
			}
		}(i)
	}

	// Concurrent publish operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			time.Sleep(time.Millisecond) // Let some subscribers register
			eb.Publish(&TestEvent{
				eventType: "mixed_event",
				data:      fmt.Sprintf("data_%d", id),
			})
		}(i)
	}

	wg.Wait()
	time.Sleep(50 * time.Millisecond)

	t.Logf("Subscribes: %d, Unsubscribes: %d, Events delivered: %d",
		atomic.LoadInt64(&subscribeCount),
		atomic.LoadInt64(&unsubscribeCount),
		atomic.LoadInt64(&eventCount))

	// Test passes if no races or panics occurred
}

// Performance tests
func TestEventBusPerformance(t *testing.T) {
	eb := NewEventBus()

	numSubscribers := 1000
	numEvents := 1000

	// Add many subscribers
	for i := 0; i < numSubscribers; i++ {
		eb.Subscribe("perf_event", nil, func(event Event) {
			// Minimal work
			_ = event.GetData()
		})
	}

	// Measure publish performance
	start := time.Now()

	for i := 0; i < numEvents; i++ {
		eb.Publish(&TestEvent{
			eventType: "perf_event",
			data:      fmt.Sprintf("data_%d", i),
		})
	}

	publishDuration := time.Since(start)

	// Wait for all events to be processed
	time.Sleep(100 * time.Millisecond)

	avgTimePerEvent := publishDuration / time.Duration(numEvents)

	t.Logf("Published %d events to %d subscribers in %v (avg: %v per event)",
		numEvents, numSubscribers, publishDuration, avgTimePerEvent)

	// Performance check (adjust threshold as needed)
	if avgTimePerEvent > time.Millisecond {
		t.Errorf("Average time per event too high: %v", avgTimePerEvent)
	}
}

// Benchmark tests
func BenchmarkEventBusSubscribe(b *testing.B) {
	eb := NewEventBus()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eb.Subscribe("bench_event", nil, func(event Event) {})
	}
}

func BenchmarkEventBusPublish(b *testing.B) {
	eb := NewEventBus()

	// Pre-add subscribers
	for i := 0; i < 100; i++ {
		eb.Subscribe("bench_event", nil, func(event Event) {})
	}

	event := &TestEvent{eventType: "bench_event", data: "benchmark_data"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eb.Publish(event)
	}
}

func BenchmarkEventBusUnsubscribe(b *testing.B) {
	eb := NewEventBus()

	// Pre-add subscribers
	var subscribers []*Subscriber
	for i := 0; i < b.N; i++ {
		subscriber := eb.Subscribe("bench_event", nil, func(event Event) {})
		subscribers = append(subscribers, subscriber)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eb.Unsubscribe("bench_event", subscribers[i])
	}
}

// Integration tests for robot scenarios
func TestEventBusRobotScenarios(t *testing.T) {
	eb := NewEventBus()

	var robotAddedCount, robotRemovedCount, robotStatusCount int32

	// WebSocket subscriber
	eb.Subscribe("robot_added", nil, func(event Event) {
		atomic.AddInt32(&robotAddedCount, 1)
		robotData := event.GetData().(map[string]interface{})
		if robotData["deviceID"] == nil {
			t.Error("Expected deviceID in robot_added event")
		}
	})

	// Database logger
	eb.Subscribe("robot_added", nil, func(event Event) {
		atomic.AddInt32(&robotAddedCount, 1)
	})

	// Status monitor
	eb.Subscribe("robot_status_changed", nil, func(event Event) {
		atomic.AddInt32(&robotStatusCount, 1)
	})

	eb.Subscribe("robot_removed", nil, func(event Event) {
		atomic.AddInt32(&robotRemovedCount, 1)
	})

	// Simulate robot lifecycle events
	eb.Publish(&TestEvent{
		eventType: "robot_added",
		data: map[string]interface{}{
			"deviceID": "robot_001",
			"ip":       "192.168.1.100",
			"type":     "trash_can",
		},
	})

	eb.Publish(&TestEvent{
		eventType: "robot_status_changed",
		data: map[string]interface{}{
			"deviceID": "robot_001",
			"status":   "active",
		},
	})

	eb.Publish(&TestEvent{
		eventType: "robot_removed",
		data: map[string]interface{}{
			"deviceID": "robot_001",
		},
	})

	time.Sleep(20 * time.Millisecond)

	if atomic.LoadInt32(&robotAddedCount) != 2 { // 2 subscribers
		t.Errorf("Expected 2 robot_added events, got %d", robotAddedCount)
	}
	if atomic.LoadInt32(&robotStatusCount) != 1 {
		t.Errorf("Expected 1 robot_status_changed event, got %d", robotStatusCount)
	}
	if atomic.LoadInt32(&robotRemovedCount) != 1 {
		t.Errorf("Expected 1 robot_removed event, got %d", robotRemovedCount)
	}
}
