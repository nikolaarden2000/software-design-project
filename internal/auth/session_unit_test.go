package auth

import (
	"math"
	"testing"
	"time"
)

func newTestSM(ttl time.Duration) *SessionManager {
	return &SessionManager{
		store:           make(map[string]*Session),
		ttl:             ttl,
		cleanupInterval: time.Hour,
		stopCh:          make(chan struct{}),
	}
}

func seedSession(sm *SessionManager, id string, userID int, expiresAt time.Time) {
	sm.mu.Lock()
	sm.store[id] = &Session{
		UserID:    userID,
		LastSeen:  time.Now().UTC(),
		ExpiresAt: expiresAt,
	}
	sm.mu.Unlock()
}

// Create

func TestCreate_ReturnsNonEmptyHexID(t *testing.T) {
	sm := newTestSM(time.Hour)
	id, err := sm.Create(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty session ID")
	}

	if len(id) != sessionIDLen*2 {
		t.Errorf("id length: got %d, want %d", len(id), sessionIDLen*2)
	}
}

func TestCreate_SessionStoredWithCorrectUserID(t *testing.T) {
	sm := newTestSM(time.Hour)
	id, _ := sm.Create(99)

	sm.mu.Lock()
	s := sm.store[id]
	sm.mu.Unlock()

	if s == nil {
		t.Fatal("session not found in store after Create")
	}
	if s.UserID != 99 {
		t.Errorf("UserID: got %d, want 99", s.UserID)
	}
}

func TestCreate_ExpiresAtWithinExpectedWindow(t *testing.T) {
	ttl := time.Hour
	sm := newTestSM(ttl)

	before := time.Now().UTC()
	id, _ := sm.Create(1)
	after := time.Now().UTC()

	sm.mu.Lock()
	s := sm.store[id]
	sm.mu.Unlock()

	lo, hi := before.Add(ttl), after.Add(ttl)
	if s.ExpiresAt.Before(lo) || s.ExpiresAt.After(hi) {
		t.Errorf("ExpiresAt %v not in [%v, %v]", s.ExpiresAt, lo, hi)
	}
}

func TestCreate_TwoCalls_ReturnDifferentIDs(t *testing.T) {
	sm := newTestSM(time.Hour)
	id1, _ := sm.Create(1)
	id2, _ := sm.Create(2)
	if id1 == id2 {
		t.Error("expected different session IDs")
	}
}

func TestCreate_BoundaryUserIDs(t *testing.T) {
	cases := []struct {
		name   string
		userID int
	}{
		{"zero", 0},
		{"negative", -1},
		{"max int32", math.MaxInt32},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sm := newTestSM(time.Hour)
			id, err := sm.Create(tc.userID)
			if err != nil {
				t.Fatalf("Create(%d): unexpected error: %v", tc.userID, err)
			}
			gotID, ok := sm.Get(id)
			if !ok {
				t.Fatal("Get after Create returned false")
			}
			if gotID != tc.userID {
				t.Errorf("userID: got %d, want %d", gotID, tc.userID)
			}
		})
	}
}

func TestGet_DecisionTable(t *testing.T) {
	cases := []struct {
		name      string
		seed      bool
		expiresIn time.Duration
		wantOK    bool
	}{
		{"not exists", false, 0, false},
		{"exists, expired", true, -time.Second, false},
		{"exists, valid", true, time.Hour, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sm := newTestSM(time.Hour)
			if tc.seed {
				seedSession(sm, "s", 42, time.Now().UTC().Add(tc.expiresIn))
			}

			_, ok := sm.Get("s")

			if ok != tc.wantOK {
				t.Errorf("ok: got %v, want %v", ok, tc.wantOK)
			}
		})
	}
}

func TestGet_BVA_ExpiredByOneNanosecond(t *testing.T) {
	sm := newTestSM(time.Hour)
	seedSession(sm, "s", 1, time.Now().UTC().Add(-time.Nanosecond))

	_, ok := sm.Get("s")

	if ok {
		t.Error("session expired 1ns ago should not be valid")
	}
}

func TestGet_BVA_ValidByOneNanosecond(t *testing.T) {
	sm := newTestSM(time.Hour)
	seedSession(sm, "s", 1, time.Now().UTC().Add(time.Millisecond*100))

	_, ok := sm.Get("s")

	if !ok {
		t.Error("session expiring in 100ms should still be valid")
	}
}

func TestGet_ValidSession_ReturnsCorrectUserID(t *testing.T) {
	sm := newTestSM(time.Hour)
	seedSession(sm, "s", 77, time.Now().UTC().Add(time.Hour))

	gotID, ok := sm.Get("s")

	if !ok {
		t.Fatal("expected ok=true")
	}
	if gotID != 77 {
		t.Errorf("userID: got %d, want 77", gotID)
	}
}

func TestGet_UnknownID_ReturnsZeroUserID(t *testing.T) {
	sm := newTestSM(time.Hour)

	gotID, ok := sm.Get("no-such-id")

	if ok || gotID != 0 {
		t.Errorf("got (%d, %v), want (0, false)", gotID, ok)
	}
}

func TestGet_ExpiredSession_RemovedFromStore(t *testing.T) {
	sm := newTestSM(time.Hour)
	seedSession(sm, "s", 1, time.Now().UTC().Add(-time.Second))

	sm.Get("s")

	sm.mu.Lock()
	_, exists := sm.store["s"]
	sm.mu.Unlock()

	if exists {
		t.Error("expired session must be removed from store after Get")
	}
}

func TestGet_RenewsTTL(t *testing.T) {
	ttl := time.Hour
	sm := newTestSM(ttl)

	seedSession(sm, "s", 1, time.Now().UTC().Add(30*time.Minute))

	before := time.Now().UTC()
	sm.Get("s")
	after := time.Now().UTC()

	sm.mu.Lock()
	s := sm.store["s"]
	sm.mu.Unlock()

	lo, hi := before.Add(ttl), after.Add(ttl)
	if s.ExpiresAt.Before(lo) || s.ExpiresAt.After(hi) {
		t.Errorf("ExpiresAt after Get: %v not in [%v, %v]", s.ExpiresAt, lo, hi)
	}
}

// Delete

func TestDelete_EquivalenceClasses(t *testing.T) {
	t.Run("existing session is removed", func(t *testing.T) {
		sm := newTestSM(time.Hour)
		seedSession(sm, "s", 1, time.Now().UTC().Add(time.Hour))

		sm.Delete("s")

		_, ok := sm.Get("s")
		if ok {
			t.Error("session should be gone after Delete")
		}
	})

	t.Run("non-existing session does not panic", func(t *testing.T) {
		sm := newTestSM(time.Hour)

		sm.Delete("ghost")
	})
}

func TestSessionLifecycle_StateTransitions(t *testing.T) {
	sm := newTestSM(time.Hour)

	_, ok := sm.Get("s")
	if ok {
		t.Fatal("state=non-existent: expected ok=false")
	}

	id, err := sm.Create(42)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	gotID, ok := sm.Get(id)
	if !ok || gotID != 42 {
		t.Fatalf("state=active: got ok=%v userID=%d, want ok=true userID=42", ok, gotID)
	}

	sm.Delete(id)
	_, ok = sm.Get(id)
	if ok {
		t.Fatal("state=deleted: expected ok=false after Delete")
	}

	id2, _ := sm.Create(7)
	sm.mu.Lock()
	sm.store[id2].ExpiresAt = time.Now().UTC().Add(-time.Second)
	sm.mu.Unlock()

	_, ok = sm.Get(id2)
	if ok {
		t.Fatal("state=expired: expected ok=false")
	}

	sm.mu.Lock()
	_, exists := sm.store[id2]
	sm.mu.Unlock()
	if exists {
		t.Fatal("state=non-existent (after expire): session must be deleted from store")
	}
}

// cleanupLoop
func TestCleanupLoop_RemovesExpiredKeepsValid(t *testing.T) {
	sm := &SessionManager{
		store:           make(map[string]*Session),
		ttl:             time.Hour,
		cleanupInterval: 20 * time.Millisecond,
		stopCh:          make(chan struct{}),
	}
	go sm.cleanupLoop()
	defer sm.Stop()

	seedSession(sm, "exp1", 1, time.Now().UTC().Add(-time.Second))
	seedSession(sm, "exp2", 2, time.Now().UTC().Add(-time.Second))
	seedSession(sm, "valid", 3, time.Now().UTC().Add(time.Hour))

	time.Sleep(60 * time.Millisecond)

	sm.mu.Lock()
	_, e1 := sm.store["exp1"]
	_, e2 := sm.store["exp2"]
	_, valid := sm.store["valid"]
	sm.mu.Unlock()

	if e1 || e2 {
		t.Error("cleanupLoop must remove expired sessions")
	}
	if !valid {
		t.Error("cleanupLoop must not remove valid sessions")
	}
}

func TestStop_CleanupLoopTerminates(t *testing.T) {
	sm := NewSessionManager(time.Hour, 10*time.Millisecond)
	done := make(chan struct{})
	go func() {
		sm.Stop()
		close(done)
	}()
	select {
	case <-done:

	case <-time.After(time.Second):
		t.Error("Stop() did not return within 1s — possible goroutine leak")
	}
}
