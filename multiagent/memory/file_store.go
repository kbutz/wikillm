package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kbutz/wikillm/multiagent"
)

// FileMemoryStore implements MemoryStore using the filesystem
type FileMemoryStore struct {
	baseDir    string
	mu         sync.RWMutex
	index      map[string]*indexEntry
	tagIndex   map[string][]string
	cleanupMu  sync.Mutex
}

type indexEntry struct {
	Key        string
	Category   string
	Tags       []string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	AccessedAt time.Time
	ExpiresAt  *time.Time
}

// NewFileMemoryStore creates a new file-based memory store
func NewFileMemoryStore(baseDir string) (*FileMemoryStore, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create memory directory: %w", err)
	}

	store := &FileMemoryStore{
		baseDir:  baseDir,
		index:    make(map[string]*indexEntry),
		tagIndex: make(map[string][]string),
	}

	// Load existing index
	if err := store.loadIndex(); err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	// Start cleanup routine
	go store.cleanupRoutine()

	return store, nil
}

// Store saves a value with the given key
func (s *FileMemoryStore) Store(ctx context.Context, key string, value interface{}) error {
	return s.StoreWithTTL(ctx, key, value, 0)
}

// StoreWithTTL saves a value with the given key and TTL
func (s *FileMemoryStore) StoreWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create memory entry
	entry := multiagent.MemoryEntry{
		Key:         key,
		Value:       value,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 0,
	}

	// Set TTL if provided
	if ttl > 0 {
		entry.TTL = &ttl
		expiresAt := time.Now().Add(ttl)
		entry.ExpiresAt = &expiresAt
	}

	// Extract category and tags from key
	entry.Category, entry.Tags = s.extractMetadata(key)

	// Marshal entry to JSON
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	// Write to file
	filename := s.getFilename(key)
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Update index
	indexEntry := &indexEntry{
		Key:       key,
		Category:  entry.Category,
		Tags:      entry.Tags,
		CreatedAt: entry.CreatedAt,
		UpdatedAt: entry.UpdatedAt,
		ExpiresAt: entry.ExpiresAt,
	}
	
	s.index[key] = indexEntry
	s.updateTagIndex(key, entry.Tags)
	
	// Save index
	return s.saveIndex()
}

// Get retrieves a value by key
func (s *FileMemoryStore) Get(ctx context.Context, key string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if key exists in index
	indexEntry, exists := s.index[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	// Check if expired
	if indexEntry.ExpiresAt != nil && time.Now().After(*indexEntry.ExpiresAt) {
		return nil, fmt.Errorf("key expired: %s", key)
	}

	// Read file
	filename := s.getFilename(key)
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal entry
	var entry multiagent.MemoryEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal entry: %w", err)
	}

	// Update access time and count
	entry.AccessedAt = time.Now()
	entry.AccessCount++
	
	// Save updated entry (in background)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		
		if data, err := json.MarshalIndent(entry, "", "  "); err == nil {
			os.WriteFile(filename, data, 0644)
		}
	}()

	return entry.Value, nil
}

// GetMultiple retrieves multiple values by keys
func (s *FileMemoryStore) GetMultiple(ctx context.Context, keys []string) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	
	for _, key := range keys {
		value, err := s.Get(ctx, key)
		if err == nil {
			results[key] = value
		}
	}
	
	return results, nil
}

// Search searches for entries matching a query
func (s *FileMemoryStore) Search(ctx context.Context, query string, limit int) ([]multiagent.MemoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	queryLower := strings.ToLower(query)
	results := make([]multiagent.MemoryEntry, 0, limit)
	
	// Search through all entries
	for key, indexEntry := range s.index {
		// Skip expired entries
		if indexEntry.ExpiresAt != nil && time.Now().After(*indexEntry.ExpiresAt) {
			continue
		}
		
		// Check if key or category matches
		if strings.Contains(strings.ToLower(key), queryLower) ||
		   strings.Contains(strings.ToLower(indexEntry.Category), queryLower) {
			
			// Load full entry
			filename := s.getFilename(key)
			data, err := os.ReadFile(filename)
			if err != nil {
				continue
			}
			
			var entry multiagent.MemoryEntry
			if err := json.Unmarshal(data, &entry); err != nil {
				continue
			}
			
			// Check if value contains query
			valueStr := fmt.Sprintf("%v", entry.Value)
			if strings.Contains(strings.ToLower(valueStr), queryLower) {
				results = append(results, entry)
				if len(results) >= limit {
					break
				}
			}
		}
	}
	
	return results, nil
}

// SearchByTags searches for entries with specific tags
func (s *FileMemoryStore) SearchByTags(ctx context.Context, tags []string, limit int) ([]multiagent.MemoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Find keys that have all specified tags
	keySet := make(map[string]int)
	for _, tag := range tags {
		if keys, exists := s.tagIndex[tag]; exists {
			for _, key := range keys {
				keySet[key]++
			}
		}
	}
	
	// Collect keys that have all tags
	results := make([]multiagent.MemoryEntry, 0, limit)
	for key, count := range keySet {
		if count == len(tags) {
			// Load entry
			filename := s.getFilename(key)
			data, err := os.ReadFile(filename)
			if err != nil {
				continue
			}
			
			var entry multiagent.MemoryEntry
			if err := json.Unmarshal(data, &entry); err != nil {
				continue
			}
			
			// Skip expired entries
			if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
				continue
			}
			
			results = append(results, entry)
			if len(results) >= limit {
				break
			}
		}
	}
	
	return results, nil
}

// Delete removes an entry by key
func (s *FileMemoryStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from index
	if indexEntry, exists := s.index[key]; exists {
		delete(s.index, key)
		
		// Remove from tag index
		for _, tag := range indexEntry.Tags {
			s.removeFromTagIndex(key, tag)
		}
	}
	
	// Delete file
	filename := s.getFilename(key)
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	
	// Save index
	return s.saveIndex()
}

// Update updates an existing entry
func (s *FileMemoryStore) Update(ctx context.Context, key string, updater func(interface{}) (interface{}, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load current value
	value, err := s.Get(ctx, key)
	if err != nil {
		return err
	}
	
	// Apply update
	newValue, err := updater(value)
	if err != nil {
		return err
	}
	
	// Store updated value
	return s.Store(ctx, key, newValue)
}

// List returns keys matching a prefix
func (s *FileMemoryStore) List(ctx context.Context, prefix string, limit int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, limit)
	
	for key := range s.index {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
			if len(keys) >= limit {
				break
			}
		}
	}
	
	return keys, nil
}

// Cleanup removes expired entries
func (s *FileMemoryStore) Cleanup(ctx context.Context) error {
	s.cleanupMu.Lock()
	defer s.cleanupMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	toDelete := []string{}
	
	// Find expired entries
	for key, indexEntry := range s.index {
		if indexEntry.ExpiresAt != nil && now.After(*indexEntry.ExpiresAt) {
			toDelete = append(toDelete, key)
		}
	}
	
	// Delete expired entries
	for _, key := range toDelete {
		delete(s.index, key)
		
		// Remove file
		filename := s.getFilename(key)
		os.Remove(filename)
	}
	
	if len(toDelete) > 0 {
		return s.saveIndex()
	}
	
	return nil
}

// Internal helper methods

func (s *FileMemoryStore) getFilename(key string) string {
	// Replace special characters to make valid filename
	safeKey := strings.ReplaceAll(key, "/", "_")
	safeKey = strings.ReplaceAll(safeKey, ":", "_")
	return filepath.Join(s.baseDir, safeKey+".json")
}

func (s *FileMemoryStore) extractMetadata(key string) (category string, tags []string) {
	// Extract category from key pattern (e.g., "agent:id:data" -> "agent")
	parts := strings.Split(key, ":")
	if len(parts) > 0 {
		category = parts[0]
	}
	
	// Extract tags based on key patterns
	tags = []string{}
	if strings.Contains(key, "msg") {
		tags = append(tags, "message")
	}
	if strings.Contains(key, "task") {
		tags = append(tags, "task")
	}
	if strings.Contains(key, "memory") {
		tags = append(tags, "memory")
	}
	
	return category, tags
}

func (s *FileMemoryStore) updateTagIndex(key string, tags []string) {
	for _, tag := range tags {
		if s.tagIndex[tag] == nil {
			s.tagIndex[tag] = []string{}
		}
		
		// Check if key already exists
		exists := false
		for _, k := range s.tagIndex[tag] {
			if k == key {
				exists = true
				break
			}
		}
		
		if !exists {
			s.tagIndex[tag] = append(s.tagIndex[tag], key)
		}
	}
}

func (s *FileMemoryStore) removeFromTagIndex(key string, tag string) {
	if keys, exists := s.tagIndex[tag]; exists {
		newKeys := []string{}
		for _, k := range keys {
			if k != key {
				newKeys = append(newKeys, k)
			}
		}
		
		if len(newKeys) > 0 {
			s.tagIndex[tag] = newKeys
		} else {
			delete(s.tagIndex, tag)
		}
	}
}

func (s *FileMemoryStore) loadIndex() error {
	indexFile := filepath.Join(s.baseDir, "_index.json")
	
	data, err := os.ReadFile(indexFile)
	if err != nil {
		if os.IsNotExist(err) {
			// No index file yet
			return nil
		}
		return err
	}
	
	var savedIndex struct {
		Entries  map[string]*indexEntry `json:"entries"`
		TagIndex map[string][]string    `json:"tag_index"`
	}
	
	if err := json.Unmarshal(data, &savedIndex); err != nil {
		return err
	}
	
	s.index = savedIndex.Entries
	s.tagIndex = savedIndex.TagIndex
	
	return nil
}

func (s *FileMemoryStore) saveIndex() error {
	indexFile := filepath.Join(s.baseDir, "_index.json")
	
	savedIndex := struct {
		Entries  map[string]*indexEntry `json:"entries"`
		TagIndex map[string][]string    `json:"tag_index"`
	}{
		Entries:  s.index,
		TagIndex: s.tagIndex,
	}
	
	data, err := json.MarshalIndent(savedIndex, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(indexFile, data, 0644)
}

func (s *FileMemoryStore) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		ctx := context.Background()
		if err := s.Cleanup(ctx); err != nil {
			fmt.Printf("Memory cleanup error: %v\n", err)
		}
	}
}
