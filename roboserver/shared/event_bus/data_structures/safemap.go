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
