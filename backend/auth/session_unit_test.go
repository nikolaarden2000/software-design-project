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
// Проверяем, что Create возвращает непустой hex-идентификатор ожидаемой длины.
func TestCreate_Scenario_ReturnsNonEmptyHexID(t *testing.T) {
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

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем, что созданная сессия сохраняется с корректным userID.
func TestCreate_Scenario_StoresCorrectUserID(t *testing.T) {
	sm := newTestSM(time.Hour)

	id, err := sm.Create(99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
}

// Техника тест-дизайна: граничные значения.
// Проверяем, что ExpiresAt попадает в ожидаемое временное окно.
func TestCreate_BoundaryValues_ExpiresAtWithinExpectedWindow(t *testing.T) {
	ttl := time.Hour
	sm := newTestSM(ttl)

	before := time.Now().UTC()
	id, err := sm.Create(1)
	after := time.Now().UTC()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sm.mu.Lock()
	s := sm.store[id]
	sm.mu.Unlock()

	lo, hi := before.Add(ttl), after.Add(ttl)
	if s.ExpiresAt.Before(lo) || s.ExpiresAt.After(hi) {
		t.Errorf("ExpiresAt %v not in [%v, %v]", s.ExpiresAt, lo, hi)
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем, что два создания сессии дают разные идентификаторы.
func TestCreate_EquivalenceClasses_TwoCallsReturnDifferentIDs(t *testing.T) {
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

// Техника тест-дизайна: граничные значения.
// Проверяем userID на границах допустимого для SessionManager поведения.
func TestCreate_BoundaryValues_UserIDs(t *testing.T) {
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

// Техника тест-дизайна: таблица решений.
// Проверяем результат Get в зависимости от наличия сессии и срока её действия.
func TestGet_DecisionTable(t *testing.T) {
	cases := []struct {
		name      string
		seed      bool
		expiresIn time.Duration
		wantOK    bool
	}{
		{"not exists", false, 0, false},
		{"exists and expired", true, -time.Second, false},
		{"exists and valid", true, time.Hour, true},
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

// Техника тест-дизайна: граничные значения.
// Проверяем сессию, которая истекла совсем недавно.
func TestGet_BoundaryValues_ExpiredByOneNanosecond(t *testing.T) {
	sm := newTestSM(time.Hour)
	seedSession(sm, "s", 1, time.Now().UTC().Add(-time.Nanosecond))

	_, ok := sm.Get("s")

	if ok {
		t.Error("session expired 1ns ago should not be valid")
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем сессию, которая ещё действует минимально малое время.
func TestGet_BoundaryValues_ValidBySmallPositiveDuration(t *testing.T) {
	sm := newTestSM(time.Hour)
	seedSession(sm, "s", 1, time.Now().UTC().Add(100*time.Millisecond))

	_, ok := sm.Get("s")

	if !ok {
		t.Error("session expiring soon should still be valid")
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем, что валидная сессия возвращает корректный userID.
func TestGet_Scenario_ValidSessionReturnsCorrectUserID(t *testing.T) {
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

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс неизвестного идентификатора сессии.
func TestGet_EquivalenceClasses_UnknownIDReturnsZeroUserID(t *testing.T) {
	sm := newTestSM(time.Hour)

	gotID, ok := sm.Get("no-such-id")

	if ok || gotID != 0 {
		t.Errorf("got (%d, %v), want (0, false)", gotID, ok)
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

// Техника тест-дизайна: переходы состояний.
// Проверяем жизненный цикл сессии: отсутствует, активна, удалена, истекла.
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

	id2, err := sm.Create(7)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

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
		t.Fatal("state=non-existent after expire: session must be deleted from store")
	}
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
