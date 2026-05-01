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

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем, что Create создаёт сессию с непустым hex-идентификатором, корректным userID и ожидаемым сроком действия.
func TestCreate_Scenario_StoresSessionWithExpectedFields(t *testing.T) {
	ttl := time.Hour
	sm := newTestSM(ttl)

	before := time.Now().UTC()
	id, err := sm.Create(99)
	after := time.Now().UTC()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty session ID")
	}
	if len(id) != sessionIDLen*2 {
		t.Errorf("id length: got %d, want %d", len(id), sessionIDLen*2)
	}

	sm.mu.Lock()
	s := sm.store[id]
	sm.mu.Unlock()

	if s == nil {
		t.Fatal("session not found in store after Create")
	}
	if s.UserID != 99 {
		t.Errorf("UserID: got %d, want 99", s.UserID)
	}

	lo, hi := before.Add(ttl), after.Add(ttl)
	if s.ExpiresAt.Before(lo) || s.ExpiresAt.After(hi) {
		t.Errorf("ExpiresAt %v not in [%v, %v]", s.ExpiresAt, lo, hi)
	}
}

// Техника тест-дизайна: предположение об ошибке.
// Проверяем, что два создания сессии не возвращают одинаковый идентификатор.
func TestCreate_ErrorGuessing_TwoCallsReturnDifferentIDs(t *testing.T) {
	sm := newTestSM(time.Hour)

	id1, err := sm.Create(1)
	if err != nil {
		t.Fatalf("unexpected error on first Create: %v", err)
	}

	id2, err := sm.Create(2)
	if err != nil {
		t.Fatalf("unexpected error on second Create: %v", err)
	}

	if id1 == id2 {
		t.Error("expected different session IDs")
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем разные классы целочисленных userID, которые SessionManager сохраняет без валидации.
func TestCreate_EquivalenceClasses_UserIDValues(t *testing.T) {
	cases := []struct {
		name   string
		userID int
	}{
		{"zero", 0},
		{"negative", -1},
		{"max int32", math.MaxInt32},
	}

	for _, tc := range cases {
		tc := tc

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

// Техника тест-дизайна: таблица решений.
// Проверяем результат Get в зависимости от наличия сессии и срока её действия.
func TestGet_DecisionTable(t *testing.T) {
	cases := []struct {
		name      string
		seed      bool
		userID    int
		expiresIn time.Duration
		wantID    int
		wantOK    bool
	}{
		{"not exists", false, 0, 0, 0, false},
		{"exists and expired", true, 42, -time.Second, 0, false},
		{"exists and valid", true, 42, time.Hour, 42, true},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			sm := newTestSM(time.Hour)
			if tc.seed {
				seedSession(sm, "s", tc.userID, time.Now().UTC().Add(tc.expiresIn))
			}

			gotID, ok := sm.Get("s")

			if gotID != tc.wantID || ok != tc.wantOK {
				t.Errorf("got (%d, %v), want (%d, %v)", gotID, ok, tc.wantID, tc.wantOK)
			}
		})
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем поведение Get около границы истечения срока действия сессии.
func TestGet_BoundaryValues_ExpirationBoundary(t *testing.T) {
	cases := []struct {
		name      string
		expiresAt time.Time
		wantOK    bool
	}{
		{"expired by one nanosecond", time.Now().UTC().Add(-time.Nanosecond), false},
		{"valid by small positive duration", time.Now().UTC().Add(100 * time.Millisecond), true},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			sm := newTestSM(time.Hour)
			seedSession(sm, "s", 1, tc.expiresAt)

			_, ok := sm.Get("s")

			if ok != tc.wantOK {
				t.Errorf("ok: got %v, want %v", ok, tc.wantOK)
			}
		})
	}
}

// Техника тест-дизайна: переходы состояний.
// Проверяем переход истёкшей сессии в удалённое состояние после Get.
func TestGet_StateTransition_ExpiredSessionRemovedFromStore(t *testing.T) {
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

// Техника тест-дизайна: переходы состояний.
// Проверяем, что валидная сессия продлевает срок действия после Get.
func TestGet_StateTransition_RenewsTTL(t *testing.T) {
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

// Техника тест-дизайна: классы эквивалентности.
// Проверяем удаление существующей и несуществующей сессии.
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

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем, что Stop завершает работу cleanupLoop без блокировки.
func TestStop_Scenario_CleanupLoopTerminates(t *testing.T) {
	sm := NewSessionManager(time.Hour, 10*time.Millisecond)

	done := make(chan struct{})
	go func() {
		sm.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("Stop did not return within 1s, possible goroutine leak")
	}
}
