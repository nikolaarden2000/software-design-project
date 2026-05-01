package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nikolaarden2000/software-design-project/backend/users"
)

var errNotConfigured = errors.New("mock: method not configured")

type mockRepo struct {
	getUserByEmail     func(ctx context.Context, email string) (*users.User, error)
	createUserWithRole func(ctx context.Context, username, email, hash, role string) (int, error)
	isEmailTaken       func(ctx context.Context, email string) (bool, error)
	getUserByID        func(ctx context.Context, id int) (*users.User, error)
}

func (m *mockRepo) CreateUserWithRole(ctx context.Context, username, email, hash, role string) (int, error) {
	if m.createUserWithRole != nil {
		return m.createUserWithRole(ctx, username, email, hash, role)
	}
	return 0, errNotConfigured
}

func (m *mockRepo) GetUserByEmail(ctx context.Context, email string) (*users.User, error) {
	if m.getUserByEmail != nil {
		return m.getUserByEmail(ctx, email)
	}
	return nil, errNotConfigured
}

func (m *mockRepo) IsEmailTaken(ctx context.Context, email string) (bool, error) {
	if m.isEmailTaken != nil {
		return m.isEmailTaken(ctx, email)
	}
	return false, errNotConfigured
}

func (m *mockRepo) GetUserByID(ctx context.Context, id int) (*users.User, error) {
	if m.getUserByID != nil {
		return m.getUserByID(ctx, id)
	}
	return nil, errNotConfigured
}

func newTestAuthService(repo UserRepo) *AuthService {
	return &AuthService{
		userRepo:       repo,
		sessionMgr:     newTestSM(time.Hour),
		cookieName:     "session_id",
		cookieSecure:   false,
		cookieSameSite: http.SameSiteLaxMode,
	}
}

func mustHashPassword(t *testing.T, password string) string {
	t.Helper()

	h, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword(%q): %v", password, err)
	}

	return h
}

func replacePart(hash string, idx int, val string) string {
	parts := strings.Split(hash, hashSeparator)
	parts[idx] = val
	return strings.Join(parts, hashSeparator)
}

func strPtr(s string) *string {
	return &s
}

// Техника тест-дизайна: граничные значения.
// Проверяем значения длины пароля около минимальной и максимальной границы.
func TestValidatePassword_BoundaryValues_Length(t *testing.T) {
	cases := []struct {
		name     string
		password string
		wantErr  error
	}{
		{"minLen-1 is invalid", strings.Repeat("a", minPasswordLen-1), ErrPasswordTooShort},
		{"minLen is valid", strings.Repeat("a", minPasswordLen), nil},
		{"maxLen is valid", strings.Repeat("a", maxPasswordLen), nil},
		{"maxLen+1 is invalid", strings.Repeat("a", maxPasswordLen+1), ErrPasswordTooLong},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			err := validatePassword(tc.password)

			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем классы допустимых и недопустимых символов пароля.
func TestValidatePassword_EquivalenceClasses_AllowedAndForbiddenCharacters(t *testing.T) {
	cases := []struct {
		name    string
		pass    string
		wantErr error
	}{
		{"letters and digits", "abcDEF12", nil},
		{"allowed special chars", "pass.w0rd", nil},
		{"forbidden char '!'", "password!", ErrPasswordInvalidChars},
		{"forbidden char '_'", "pass_word", ErrPasswordInvalidChars},
		{"space in password", "pass word1", ErrPasswordInvalidChars},
		{"cyrillic chars", "пароль123", ErrPasswordInvalidChars},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			err := validatePassword(tc.pass)

			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем полный цикл: хеширование пароля и успешная проверка тем же паролем.
func TestVerifyPassword_Scenario_RoundTrip(t *testing.T) {
	hash := mustHashPassword(t, "password123")

	ok, err := verifyPassword("password123", hash)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true for matching password")
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс несовпадающего пароля.
func TestVerifyPassword_EquivalenceClasses_WrongPasswordReturnsFalse(t *testing.T) {
	hash := mustHashPassword(t, "correct-password")

	ok, err := verifyPassword("wrong-password", hash)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected false for mismatched password")
	}
}

// Техника тест-дизайна: предположение об ошибке.
// Проверяем, что одинаковый пароль получает разные хеши из-за случайной соли.
func TestHashPassword_ErrorGuessing_SameInputProducesDifferentHashes(t *testing.T) {
	h1 := mustHashPassword(t, "password123")
	h2 := mustHashPassword(t, "password123")

	if h1 == h2 {
		t.Error("two hashes of the same password must differ because salt is random")
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс некорректно сохранённого хеша.
func TestVerifyPassword_EquivalenceClasses_InvalidStoredHashReturnsError(t *testing.T) {
	_, err := verifyPassword("password", "not-a-valid-hash")

	if err == nil {
		t.Error("expected error for invalid hash format")
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем классы корректной структуры хеша и разных вариантов некорректных полей.
func TestParseHashString_EquivalenceClasses(t *testing.T) {
	valid := mustHashPassword(t, "test-password")

	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid hash", valid, false},
		{"too few parts", "a$b$c", true},
		{"unsupported version", replacePart(valid, 0, "99"), true},
		{"non-numeric version", replacePart(valid, 0, "x"), true},
		{"non-numeric time", replacePart(valid, 1, "x"), true},
		{"non-numeric memory", replacePart(valid, 2, "x"), true},
		{"non-numeric threads", replacePart(valid, 3, "x"), true},
		{"non-numeric keyLen", replacePart(valid, 4, "x"), true},
		{"invalid salt hex", replacePart(valid, 5, "GG"), true},
		{"invalid hash hex", replacePart(valid, 6, "GG"), true},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			_, err := parseHashString(tc.input)

			if (err != nil) != tc.wantErr {
				t.Errorf("err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем классы невалидных входных данных регистрации до обращения к репозиторию.
func TestRegister_InputValidationEquivalenceClasses_NoDBCalls(t *testing.T) {
	cases := []struct {
		name     string
		username string
		email    string
		password string
		wantErr  error
	}{
		{"empty username", "", "a@b.com", "password1", ErrEmptyUsername},
		{"whitespace username", "   ", "a@b.com", "password1", ErrEmptyUsername},
		{"empty email", "alice", "", "password1", ErrEmptyEmail},
		{"whitespace email", "alice", "   ", "password1", ErrEmptyEmail},
		{"invalid email", "alice", "not-an-email", "password1", ErrInvalidEmail},
		{"email without domain", "alice", "alice@", "password1", ErrInvalidEmail},
		{"password too short", "alice", "a@b.com", "pass1", ErrPasswordTooShort},
		{"password too long", "alice", "a@b.com", strings.Repeat("a", maxPasswordLen+1), ErrPasswordTooLong},
		{"password invalid chars", "alice", "a@b.com", "pass!word", ErrPasswordInvalidChars},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			svc := newTestAuthService(&mockRepo{})

			_, err := svc.Register(context.Background(), tc.username, tc.email, tc.password)

			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс email, который уже занят.
func TestRegister_EquivalenceClasses_EmailAlreadyTakenReturnsErrEmailExists(t *testing.T) {
	repo := &mockRepo{
		isEmailTaken: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}
	svc := newTestAuthService(repo)

	_, err := svc.Register(context.Background(), "alice", "alice@example.com", "password1")

	if !errors.Is(err, ErrEmailExists) {
		t.Errorf("expected ErrEmailExists, got: %v", err)
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка проверки уникальности email возвращается вызывающему коду.
func TestRegister_ExceptionHandling_IsEmailTakenErrorPropagates(t *testing.T) {
	dbErr := fmt.Errorf("connection lost")
	repo := &mockRepo{
		isEmailTaken: func(_ context.Context, _ string) (bool, error) {
			return false, dbErr
		},
	}
	svc := newTestAuthService(repo)

	_, err := svc.Register(context.Background(), "alice", "alice@example.com", "password1")

	if !errors.Is(err, dbErr) {
		t.Errorf("expected wrapped dbErr, got: %v", err)
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем, что Register нормализует входные данные и создаёт пользователя с ролью user по умолчанию.
func TestRegister_Scenario_UsesDefaultUserRole(t *testing.T) {
	repo := &mockRepo{
		isEmailTaken: func(_ context.Context, email string) (bool, error) {
			if email != "alice@example.com" {
				t.Fatalf("email: got %q, want alice@example.com", email)
			}
			return false, nil
		},
		createUserWithRole: func(_ context.Context, username, email, hash, role string) (int, error) {
			if username != "alice" {
				t.Fatalf("username: got %q, want alice", username)
			}
			if email != "alice@example.com" {
				t.Fatalf("email: got %q, want alice@example.com", email)
			}
			if role != users.RoleUser {
				t.Fatalf("role: got %q, want %q", role, users.RoleUser)
			}

			ok, err := verifyPassword("password1", hash)
			if err != nil {
				t.Fatalf("verifyPassword returned error: %v", err)
			}
			if !ok {
				t.Fatal("password hash does not match original password")
			}

			return 42, nil
		},
	}

	svc := newTestAuthService(repo)

	id, err := svc.Register(context.Background(), " alice ", " alice@example.com ", "password1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Fatalf("id: got %d, want 42", id)
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем допустимые роли как класс валидных значений.
func TestRegisterWithRole_EquivalenceClasses_ValidRoles(t *testing.T) {
	cases := []struct {
		name string
		role string
	}{
		{"user role", users.RoleUser},
		{"admin role", users.RoleAdmin},
		{"superuser role", users.RoleSuperuser},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepo{
				isEmailTaken: func(_ context.Context, _ string) (bool, error) {
					return false, nil
				},
				createUserWithRole: func(_ context.Context, username, email, hash, role string) (int, error) {
					if username != "alice" {
						t.Fatalf("username: got %q, want alice", username)
					}
					if email != "alice@example.com" {
						t.Fatalf("email: got %q, want alice@example.com", email)
					}
					if role != tc.role {
						t.Fatalf("role: got %q, want %q", role, tc.role)
					}

					ok, err := verifyPassword("password1", hash)
					if err != nil {
						t.Fatalf("verifyPassword returned error: %v", err)
					}
					if !ok {
						t.Fatal("password hash does not match original password")
					}

					return 100, nil
				},
			}

			svc := newTestAuthService(repo)

			id, err := svc.RegisterWithRole(context.Background(), "alice", "alice@example.com", "password1", tc.role)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != 100 {
				t.Fatalf("id: got %d, want 100", id)
			}
		})
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс невалидных ролей, которые отклоняются до обращения к репозиторию.
func TestRegisterWithRole_EquivalenceClasses_InvalidRole(t *testing.T) {
	svc := newTestAuthService(&mockRepo{})

	_, err := svc.RegisterWithRole(context.Background(), "alice", "alice@example.com", "password1", "moderator")

	if err == nil {
		t.Fatal("expected error for invalid role, got nil")
	}
	if !strings.Contains(err.Error(), "invalid role") {
		t.Fatalf("expected invalid role error, got %v", err)
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка CreateUserWithRole возвращается вызывающему коду.
func TestRegisterWithRole_ExceptionHandling_CreateUserWithRoleErrorPropagates(t *testing.T) {
	createErr := fmt.Errorf("insert failed")

	repo := &mockRepo{
		isEmailTaken: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
		createUserWithRole: func(_ context.Context, _, _, _, _ string) (int, error) {
			return 0, createErr
		},
	}

	svc := newTestAuthService(repo)

	_, err := svc.RegisterWithRole(context.Background(), "alice", "alice@example.com", "password1", users.RoleAdmin)

	if !errors.Is(err, createErr) {
		t.Fatalf("got error %v, want wrapped %v", err, createErr)
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем классы невалидных входных данных входа до обращения к репозиторию.
func TestLogin_InputValidationEquivalenceClasses_NoDBCalls(t *testing.T) {
	cases := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{"empty email", "", "password1", ErrEmptyEmail},
		{"whitespace email", "   ", "password1", ErrEmptyEmail},
		{"empty password", "a@b.com", "", ErrInvalidCredentials},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			svc := newTestAuthService(&mockRepo{})
			w := httptest.NewRecorder()

			_, err := svc.Login(context.Background(), tc.email, tc.password, w)

			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка поиска пользователя возвращается вызывающему коду.
func TestLogin_ExceptionHandling_UserLookupErrorPropagates(t *testing.T) {
	dbErr := fmt.Errorf("user not found")
	repo := &mockRepo{
		getUserByEmail: func(_ context.Context, _ string) (*users.User, error) {
			return nil, dbErr
		},
	}
	svc := newTestAuthService(repo)

	_, err := svc.Login(context.Background(), "a@b.com", "password1", httptest.NewRecorder())

	if !errors.Is(err, dbErr) {
		t.Errorf("expected wrapped dbErr, got: %v", err)
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс неверного пароля.
func TestLogin_EquivalenceClasses_WrongPasswordReturnsInvalidCredentials(t *testing.T) {
	correctHash := mustHashPassword(t, "correct-password")
	repo := &mockRepo{
		getUserByEmail: func(_ context.Context, email string) (*users.User, error) {
			return &users.User{ID: 1, Email: email, Password: correctHash}, nil
		},
	}
	svc := newTestAuthService(repo)

	_, err := svc.Login(context.Background(), "a@b.com", "wrong-password", httptest.NewRecorder())

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем поведение при повреждённом сохранённом хеше.
func TestLogin_ExceptionHandling_CorruptedStoredHashReturnsError(t *testing.T) {
	repo := &mockRepo{
		getUserByEmail: func(_ context.Context, email string) (*users.User, error) {
			return &users.User{ID: 1, Email: email, Password: "corrupted-hash"}, nil
		},
	}
	svc := newTestAuthService(repo)

	_, err := svc.Login(context.Background(), "a@b.com", "password1", httptest.NewRecorder())

	if err == nil {
		t.Error("expected error for corrupted stored hash")
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем успешный вход, возврат пользователя, cookie и создание сессии.
func TestLogin_Scenario_ReturnsUserAndSetsCookie(t *testing.T) {
	correctHash := mustHashPassword(t, "password1")
	repo := &mockRepo{
		getUserByEmail: func(_ context.Context, email string) (*users.User, error) {
			return &users.User{
				ID:       7,
				Username: "alice",
				Email:    email,
				Password: correctHash,
				Role:     users.RoleUser,
			}, nil
		},
	}
	svc := newTestAuthService(repo)
	w := httptest.NewRecorder()

	u, err := svc.Login(context.Background(), "a@b.com", "password1", w)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u == nil {
		t.Fatal("expected user, got nil")
	}
	if u.ID != 7 {
		t.Errorf("user id: got %d, want 7", u.ID)
	}
	if u.Email != "a@b.com" {
		t.Errorf("user email: got %q, want a@b.com", u.Email)
	}
	if u.Username != "alice" {
		t.Errorf("username: got %q, want alice", u.Username)
	}
	if u.Role != users.RoleUser {
		t.Errorf("role: got %q, want %q", u.Role, users.RoleUser)
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	c := cookies[0]
	if c.Name != "session_id" {
		t.Errorf("cookie name: got %q, want session_id", c.Name)
	}
	if c.Value == "" {
		t.Error("expected non-empty session cookie value")
	}
	if !c.HttpOnly {
		t.Error("cookie must be HttpOnly")
	}

	uid, ok := svc.sessionMgr.Get(c.Value)
	if !ok {
		t.Error("expected created session to exist in session manager")
	}
	if uid != 7 {
		t.Errorf("session user id: got %d, want 7", uid)
	}
}

// Техника тест-дизайна: переходы состояний.
// Проверяем переход пользователя из состояния с активной сессией в состояние без сессии после Logout.
func TestLogout_StateTransition_WithCookieDeletesSessionAndClearsCookie(t *testing.T) {
	svc := newTestAuthService(&mockRepo{})
	sessionID, _ := svc.sessionMgr.Create(1)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	w := httptest.NewRecorder()

	svc.Logout(w, req)

	_, ok := svc.sessionMgr.Get(sessionID)
	if ok {
		t.Error("session must be deleted after Logout")
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 Set-Cookie header, got %d", len(cookies))
	}

	c := cookies[0]
	if c.MaxAge != -1 {
		t.Errorf("cookie MaxAge: got %d, want -1", c.MaxAge)
	}
	if c.Value != "" {
		t.Errorf("cookie value must be empty after Logout, got %q", c.Value)
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс выхода без входящей cookie.
func TestLogout_EquivalenceClasses_WithoutCookieClearsCookie(t *testing.T) {
	svc := newTestAuthService(&mockRepo{})
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	w := httptest.NewRecorder()

	svc.Logout(w, req)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 Set-Cookie header even without incoming cookie, got %d", len(cookies))
	}
	if cookies[0].MaxAge != -1 {
		t.Errorf("cookie MaxAge: got %d, want -1", cookies[0].MaxAge)
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем результат VerifyRequest в зависимости от наличия cookie и состояния сессии.
func TestVerifyRequest_DecisionTable(t *testing.T) {
	svc := newTestAuthService(&mockRepo{})
	validID, _ := svc.sessionMgr.Create(99)

	cases := []struct {
		name      string
		cookieVal *string
		wantOK    bool
		wantUID   int
	}{
		{"no cookie", nil, false, 0},
		{"unknown session id", strPtr("non-existent"), false, 0},
		{"valid session", &validID, true, 99},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.cookieVal != nil {
				req.AddCookie(&http.Cookie{Name: "session_id", Value: *tc.cookieVal})
			}

			uid, ok := svc.VerifyRequest(req)

			if ok != tc.wantOK || uid != tc.wantUID {
				t.Errorf("got (%d, %v), want (%d, %v)", uid, ok, tc.wantUID, tc.wantOK)
			}
		})
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем, что OptionalUserMiddleware кладёт user id и user в контекст.
func TestOptionalUserMiddleware_Scenario_AuthenticatedRequestInjectsUserIntoContext(t *testing.T) {
	repo := &mockRepo{
		getUserByID: func(_ context.Context, id int) (*users.User, error) {
			if id != 42 {
				t.Fatalf("GetUserByID id: got %d, want 42", id)
			}

			return &users.User{
				ID:       42,
				Username: "alice",
				Email:    "alice@example.com",
				Role:     users.RoleAdmin,
			}, nil
		},
	}

	svc := newTestAuthService(repo)
	sessionID, _ := svc.sessionMgr.Create(42)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})

	var gotUID int
	var gotUIDOK bool
	var gotUser *users.User
	var gotUserOK bool

	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotUID, gotUIDOK = UserIDFromContext(r.Context())
		gotUser, gotUserOK = UserFromContext(r.Context())
	})

	svc.OptionalUserMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)

	if !gotUIDOK || gotUID != 42 {
		t.Errorf("context user id: got uid=%d ok=%v, want uid=42 ok=true", gotUID, gotUIDOK)
	}

	if !gotUserOK || gotUser == nil {
		t.Fatal("expected user in context")
	}

	if gotUser.ID != 42 {
		t.Errorf("context user: got id=%d, want 42", gotUser.ID)
	}

	if gotUser.Role != users.RoleAdmin {
		t.Errorf("context user role: got %q, want %q", gotUser.Role, users.RoleAdmin)
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс запроса без авторизации.
func TestOptionalUserMiddleware_EquivalenceClasses_UnauthenticatedRequestCallsNextWithoutUser(t *testing.T) {
	svc := newTestAuthService(&mockRepo{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	nextCalled := false
	var gotUIDOK bool
	var gotUserOK bool

	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		nextCalled = true
		_, gotUIDOK = UserIDFromContext(r.Context())
		_, gotUserOK = UserFromContext(r.Context())
	})

	svc.OptionalUserMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)

	if !nextCalled {
		t.Error("next handler must be called even for unauthenticated request")
	}

	if gotUIDOK {
		t.Error("UserID must not be in context for unauthenticated request")
	}

	if gotUserOK {
		t.Error("User must not be in context for unauthenticated request")
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем, что RequireAuth пропускает авторизованный запрос дальше по цепочке обработчиков.
func TestRequireAuth_Scenario_AuthenticatedRequestCallsNext(t *testing.T) {
	repo := &mockRepo{
		getUserByID: func(_ context.Context, id int) (*users.User, error) {
			return &users.User{ID: id, Role: users.RoleUser}, nil
		},
	}
	svc := newTestAuthService(repo)
	sessionID, _ := svc.sessionMgr.Create(5)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	w := httptest.NewRecorder()

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	svc.RequireAuth(next).ServeHTTP(w, req)

	if !nextCalled {
		t.Error("next handler must be called for authenticated request")
	}
	if w.Code != http.StatusNoContent {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNoContent)
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс неавторизованного запроса к RequireAuth.
func TestRequireAuth_EquivalenceClasses_UnauthenticatedRequestReturnsUnauthorized(t *testing.T) {
	svc := newTestAuthService(&mockRepo{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		nextCalled = true
	})

	svc.RequireAuth(next).ServeHTTP(w, req)

	if nextCalled {
		t.Error("next handler must not be called for unauthenticated request")
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем RequireRole по комбинациям: наличие сессии, успешная загрузка пользователя и разрешённость роли.
func TestRequireRole_DecisionTable(t *testing.T) {
	lookupErr := fmt.Errorf("lookup failed")

	cases := []struct {
		name          string
		hasSession    bool
		repoUser      *users.User
		repoErr       error
		allowedRoles  []string
		wantStatus    int
		wantNext      bool
		wantRepoCall  bool
		sessionUserID int
	}{
		{
			name:         "no session returns unauthorized",
			allowedRoles: []string{users.RoleAdmin},
			wantStatus:   http.StatusUnauthorized,
			wantNext:     false,
			wantRepoCall: false,
		},
		{
			name:          "user lookup error returns unauthorized",
			hasSession:    true,
			repoErr:       lookupErr,
			allowedRoles:  []string{users.RoleAdmin},
			wantStatus:    http.StatusUnauthorized,
			wantNext:      false,
			wantRepoCall:  true,
			sessionUserID: 10,
		},
		{
			name:          "authenticated user with forbidden role returns forbidden",
			hasSession:    true,
			repoUser:      &users.User{ID: 11, Role: users.RoleUser},
			allowedRoles:  []string{users.RoleAdmin},
			wantStatus:    http.StatusForbidden,
			wantNext:      false,
			wantRepoCall:  true,
			sessionUserID: 11,
		},
		{
			name:          "authenticated user with empty allowed list returns forbidden",
			hasSession:    true,
			repoUser:      &users.User{ID: 12, Role: users.RoleAdmin},
			allowedRoles:  nil,
			wantStatus:    http.StatusForbidden,
			wantNext:      false,
			wantRepoCall:  true,
			sessionUserID: 12,
		},
		{
			name:          "authenticated user with allowed role calls next",
			hasSession:    true,
			repoUser:      &users.User{ID: 13, Role: users.RoleSuperuser},
			allowedRoles:  []string{users.RoleAdmin, users.RoleSuperuser},
			wantStatus:    http.StatusNoContent,
			wantNext:      true,
			wantRepoCall:  true,
			sessionUserID: 13,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			repoCalled := false
			repo := &mockRepo{
				getUserByID: func(_ context.Context, id int) (*users.User, error) {
					repoCalled = true

					if id != tc.sessionUserID {
						t.Fatalf("GetUserByID id: got %d, want %d", id, tc.sessionUserID)
					}
					if tc.repoErr != nil {
						return nil, tc.repoErr
					}
					if tc.repoUser == nil {
						t.Fatal("repoUser must be configured for this case")
					}

					return tc.repoUser, nil
				},
			}

			svc := newTestAuthService(repo)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.hasSession {
				sessionID, err := svc.sessionMgr.Create(tc.sessionUserID)
				if err != nil {
					t.Fatalf("Create session: %v", err)
				}
				req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
			}

			w := httptest.NewRecorder()
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusNoContent)
			})

			svc.RequireRole(tc.allowedRoles...)(next).ServeHTTP(w, req)

			if repoCalled != tc.wantRepoCall {
				t.Errorf("repoCalled: got %v, want %v", repoCalled, tc.wantRepoCall)
			}
			if nextCalled != tc.wantNext {
				t.Errorf("nextCalled: got %v, want %v", nextCalled, tc.wantNext)
			}
			if w.Code != tc.wantStatus {
				t.Errorf("status: got %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем извлечение user id из контекста для классов: отсутствует, неверный тип, корректный тип.
func TestUserIDFromContext_EquivalenceClasses(t *testing.T) {
	cases := []struct {
		name   string
		ctx    context.Context
		wantID int
		wantOK bool
	}{
		{"missing value", context.Background(), 0, false},
		{"wrong type", context.WithValue(context.Background(), KeyUserID, "42"), 0, false},
		{"valid value", context.WithValue(context.Background(), KeyUserID, 42), 42, true},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			id, ok := UserIDFromContext(tc.ctx)

			if id != tc.wantID || ok != tc.wantOK {
				t.Errorf("got (%d, %v), want (%d, %v)", id, ok, tc.wantID, tc.wantOK)
			}
		})
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем извлечение user из контекста для классов: отсутствует, неверный тип, корректный тип.
func TestUserFromContext_EquivalenceClasses(t *testing.T) {
	user := &users.User{ID: 42, Role: users.RoleAdmin}

	cases := []struct {
		name     string
		ctx      context.Context
		wantUser *users.User
		wantOK   bool
	}{
		{"missing value", context.Background(), nil, false},
		{"wrong type", context.WithValue(context.Background(), KeyUser, 42), nil, false},
		{"valid value", context.WithValue(context.Background(), KeyUser, user), user, true},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			gotUser, ok := UserFromContext(tc.ctx)

			if gotUser != tc.wantUser || ok != tc.wantOK {
				t.Errorf("got (%v, %v), want (%v, %v)", gotUser, ok, tc.wantUser, tc.wantOK)
			}
		})
	}
}
