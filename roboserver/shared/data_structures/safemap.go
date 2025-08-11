package data_structures

func NewSafeMap[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		m: make(map[K]V),
	}
}

func (sm *SafeMap[K, V]) Set(key K, value V) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.m == nil {
		sm.m = make(map[K]V)
	}
	sm.m[key] = value
}

func (sm *SafeMap[K, V]) Get(key K) (V, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	val, ok := sm.m[key]
	return val, ok
}

func (sm *SafeMap[K, V]) Pop(key K) (V, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	val, ok := sm.m[key]
	if ok {
		delete(sm.m, key)
	}
	return val, ok
}

func (sm *SafeMap[K, V]) GetOrDefault(key K, defaultValue V) V {
	sm.mu.RLock()
	if val, ok := sm.m[key]; ok {
		defer sm.mu.RUnlock()
		return val
	}
	sm.mu.RUnlock()
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if val, ok := sm.m[key]; ok {
		return val
	}
	sm.m[key] = defaultValue
	return defaultValue
}

func (sm *SafeMap[K, V]) Delete(key K) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.m, key)
}

func (sm *SafeMap[K, V]) GetKeys() []K {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	keys := make([]K, 0, len(sm.m))
	for k := range sm.m {
		keys = append(keys, k)
	}
	return keys
}

// DeleteIfEmpty removes the key if the value implements IsEmpty() and returns true
// Returns false if key doesn't exist, value doesn't implement IsEmpty(), or value is not empty
func (sm *SafeMap[K, V]) DeleteIfEmpty(key K) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if val, ok := sm.m[key]; ok {
		// Check if the value implements the IsEmpty() method
		if emptyable, hasMethod := any(val).(interface{ IsEmpty() bool }); hasMethod {
			if emptyable.IsEmpty() {
				delete(sm.m, key)
				return true
			}
		}
	}
	return false
}

func (sm *SafeMap[K, V]) IsEmpty() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.m) == 0
}
