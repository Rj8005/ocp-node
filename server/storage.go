package server

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type StoredMessage struct {
	ID        string    `json:"id"`
	ToOCP     string    `json:"to_ocp"`
	FromOCP   string    `json:"from_ocp"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type MessageStore struct {
	mu       sync.RWMutex
	messages map[string][]StoredMessage
	filePath string
}

func NewMessageStore(filePath string) *MessageStore {
	ms := &MessageStore{
		messages: make(map[string][]StoredMessage),
		filePath: filePath,
	}
	ms.load()
	go ms.cleanupLoop()
	return ms
}

func (ms *MessageStore) Store(toOCP, fromOCP, body string, ttlDays int) string {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	msg := StoredMessage{
		ID:        generateID(),
		ToOCP:     toOCP,
		FromOCP:   fromOCP,
		Body:      body,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Duration(ttlDays) * 24 * time.Hour),
	}

	ms.messages[toOCP] = append(ms.messages[toOCP], msg)
	ms.save()
	log.Printf("[storage] stored message %s for %s", msg.ID, truncate(toOCP, 16))
	return msg.ID
}

func (ms *MessageStore) GetPending(toOCP string) []StoredMessage {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	msgs := ms.messages[toOCP]
	if len(msgs) == 0 {
		return nil
	}

	delete(ms.messages, toOCP)
	ms.save()
	log.Printf("[storage] delivered %d messages to %s", len(msgs), truncate(toOCP, 16))
	return msgs
}

func (ms *MessageStore) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		ms.cleanup()
	}
}

func (ms *MessageStore) cleanup() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	cleaned := 0
	for ocp, msgs := range ms.messages {
		var active []StoredMessage
		for _, msg := range msgs {
			if now.Before(msg.ExpiresAt) {
				active = append(active, msg)
			} else {
				cleaned++
			}
		}
		if len(active) == 0 {
			delete(ms.messages, ocp)
		} else {
			ms.messages[ocp] = active
		}
	}
	if cleaned > 0 {
		log.Printf("[storage] cleaned %d expired messages", cleaned)
		ms.save()
	}
}

func (ms *MessageStore) save() {
	if ms.filePath == "" {
		return
	}
	data, _ := json.Marshal(ms.messages)
	os.WriteFile(ms.filePath, data, 0644)
}

func (ms *MessageStore) load() {
	if ms.filePath == "" {
		return
	}
	data, err := os.ReadFile(ms.filePath)
	if err != nil {
		return
	}
	json.Unmarshal(data, &ms.messages)
	log.Printf("[storage] loaded message store from %s", ms.filePath)
}

func generateID() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = byte(time.Now().UnixNano() >> (uint(i) * 8))
	}
	return fmt.Sprintf("%x", b)
}

// truncate returns s[:n] safely, appending "..." if truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
