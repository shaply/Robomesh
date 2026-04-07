package handler_engine

import (
	"fmt"
	"sync"
)

// HandlerManager tracks all active handler processes globally.
// It is safe for concurrent access.
var HandlerManager = &handlerManager{
	handlers: make(map[string]*HandlerProcess),
	spawning: make(map[string]bool),
}

type handlerManager struct {
	mu       sync.RWMutex
	handlers map[string]*HandlerProcess
	spawning map[string]bool
}

// Register adds a handler process to the global map.
func (m *handlerManager) Register(hp *HandlerProcess) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[hp.UUID] = hp
}

// Unregister removes a handler process from the global map.
func (m *handlerManager) Unregister(uuid string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.handlers, uuid)
}

// Get retrieves a handler process by robot UUID.
func (m *handlerManager) Get(uuid string) (*HandlerProcess, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	hp, ok := m.handlers[uuid]
	return hp, ok
}

// Has checks if a handler process exists for the given UUID.
func (m *handlerManager) Has(uuid string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.handlers[uuid]
	return ok
}

// TryStartSpawning atomically checks if a handler is running or being spawned,
// and marks the UUID as spawning if neither. Returns true if spawning can proceed.
func (m *handlerManager) TryStartSpawning(uuid string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.handlers[uuid]; ok {
		return false // already running
	}
	if m.spawning[uuid] {
		return false // already being spawned by another request
	}
	m.spawning[uuid] = true
	return true
}

// FinishSpawning removes the spawning mark for a UUID.
func (m *handlerManager) FinishSpawning(uuid string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.spawning, uuid)
}

// Kill stops and removes a handler process by UUID.
func (m *handlerManager) Kill(uuid string) error {
	m.mu.Lock()
	hp, ok := m.handlers[uuid]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("no handler running for robot %s", uuid)
	}
	delete(m.handlers, uuid)
	m.mu.Unlock()

	hp.Stop("killed")
	return nil
}

// StopAll gracefully stops all running handlers.
func (m *handlerManager) StopAll(reason string) {
	m.mu.Lock()
	handlers := make(map[string]*HandlerProcess, len(m.handlers))
	for k, v := range m.handlers {
		handlers[k] = v
	}
	m.handlers = make(map[string]*HandlerProcess)
	m.mu.Unlock()

	for _, hp := range handlers {
		hp.Stop(reason)
	}
}

// ListAll returns a snapshot of all running handler UUIDs and their PIDs.
func (m *handlerManager) ListAll() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]int, len(m.handlers))
	for uuid, hp := range m.handlers {
		result[uuid] = hp.PID
	}
	return result
}
