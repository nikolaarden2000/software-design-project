package auth

import (
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

const (
	sessionIDLen        = 32
	maxSessionIDRetries = 5
)

type Session struct {
	UserID    int
	LastSeen  time.Time
	ExpiresAt time.Time
}

type SessionManager struct {
	store map[string]*Session
	mu    sync.Mutex

	ttl             time.Duration
	cleanupInterval time.Duration
	stopCh          chan struct{}
}

func NewSessionManager(ttl, cleanupInterval time.Duration) *SessionManager {
	sm := &SessionManager{
		store:           make(map[string]*Session),
		ttl:             ttl,
		cleanupInterval: cleanupInterval,
		stopCh:          make(chan struct{}),
	}
	go sm.cleanupLoop()
	return sm
}

func (sm *SessionManager) Stop() {
	close(sm.stopCh)
}

func (sm *SessionManager) Create(userID int) (string, error) {
	for i := 0; i < maxSessionIDRetries; i++ {
		b, err := genRandomBytes(sessionIDLen)
		if err != nil {
			return "", err
		}
		id := hex.EncodeToString(b)

		sm.mu.Lock()
		if _, exists := sm.store[id]; !exists {
			now := time.Now().UTC()
			sm.store[id] = &Session{
				UserID:    userID,
				LastSeen:  now,
				ExpiresAt: now.Add(sm.ttl),
			}
			sm.mu.Unlock()
			return id, nil
		}
		sm.mu.Unlock()
	}
	return "", errors.New("auth: failed to generate unique session id")
}

func (sm *SessionManager) Get(sessionID string) (int, bool) {
	now := time.Now().UTC()

	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.store[sessionID]
	if !ok {
		return 0, false
	}
	if now.After(s.ExpiresAt) {
		delete(sm.store, sessionID)
		return 0, false
	}
	s.LastSeen = time.Now().UTC()
	s.ExpiresAt = s.LastSeen.Add(sm.ttl)
	return s.UserID, true
}

func (sm *SessionManager) Delete(sessionID string) {
	sm.mu.Lock()
	delete(sm.store, sessionID)
	sm.mu.Unlock()
}

func (sm *SessionManager) cleanupLoop() {
	ticker := time.NewTicker(sm.cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now().UTC()
			sm.mu.Lock()
			for id, s := range sm.store {
				if now.After(s.ExpiresAt) {
					delete(sm.store, id)
				}
			}
			sm.mu.Unlock()
		case <-sm.stopCh:
			return
		}
	}
}
