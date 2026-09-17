package idempotency

import (
	"context"
	"sync"
	"time"
)

type memoryItem struct {
	record    *Record
	expiresAt time.Time
}

// MemoryStorage implements Storage using an in-memory thread-safe map.
// It is ideal for local development, unit testing, and lightweight single-node deployments.
type MemoryStorage struct {
	mu    sync.RWMutex
	items map[string]memoryItem
}

// NewMemoryStorage initializes and returns a new MemoryStorage instance.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		items: make(map[string]memoryItem),
	}
}

// Lock atomically acquires a lock for key with a TTL if it does not already exist.
func (m *MemoryStorage) Lock(_ context.Context, key string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	if item, exists := m.items[key]; exists {
		if now.Before(item.expiresAt) {
			return false, nil // Key is currently locked or active
		}
	}

	m.items[key] = memoryItem{
		record: &Record{
			Status:    StatusInProgress,
			CreatedAt: now,
		},
		expiresAt: now.Add(ttl),
	}

	return true, nil
}

// Set stores the completed record for key with a TTL.
func (m *MemoryStorage) Set(_ context.Context, key string, record *Record, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items[key] = memoryItem{
		record:    record,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Get retrieves the record for key if it exists and has not expired.
func (m *MemoryStorage) Get(_ context.Context, key string) (*Record, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.items[key]
	if !exists {
		return nil, nil
	}

	if time.Now().After(item.expiresAt) {
		return nil, nil
	}

	return item.record, nil
}

// Delete removes the key from the in-memory map.
func (m *MemoryStorage) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.items, key)
	return nil
}
