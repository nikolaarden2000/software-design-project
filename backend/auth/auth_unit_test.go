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
	createUser     func(ctx context.Context, username, email, hash string) (int, error)
	getUserByEmail func(ctx context.Context, email string) (*users.User, error)
	isEmailTaken   func(ctx context.Context, email string) (bool, error)
	getUserByID    func(ctx context.Context, id int) (*users.User, error)
}

func (m *mockRepo) CreateUser(ctx context.Context, username, email, hash string) (int, error) {
	if m.createUser != nil {
		return m.createUser(ctx, username, email, hash)
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

func TestValidatePassword_BVA_Length(t *testing.T) {
	cases := []struct {
		name     string
		password string
		wantErr  error
	}{
		{"minLen-1 (7 chars)", strings.Repeat("a", minPasswordLen-1), ErrPasswordTooShort},
		{"minLen (8 chars)", strings.Repeat("a", minPasswordLen), nil},
		{"maxLen (64 chars)", strings.Repeat("a", maxPasswordLen), nil},
		{"maxLen+1 (65 chars)", strings.Repeat("a", maxPasswordLen+1), ErrPasswordTooLong},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePassword(tc.password)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidatePassword_EquivalenceClasses_Characters(t *testing.T) {
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
		t.Run(tc.name, func(t *testing.T) {
			err := validatePassword(tc.pass)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestHashVerify_RoundTrip(t *testing.T) {
	hash := mustHashPassword(t, "password123")

	ok, err := verifyPassword("password123", hash)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true for matching password")
	}
}

func TestHashVerify_WrongPassword_ReturnsFalse(t *testing.T) {
	hash := mustHashPassword(t, "correct-password")

	ok, err := verifyPassword("wrong-password", hash)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected false for mismatched password")
	}
}

func TestHashPassword_SameInput_ProducesDifferentHashes(t *testing.T) {
	h1 := mustHashPassword(t, "password123")
	h2 := mustHashPassword(t, "password123")

	if h1 == h2 {
		t.Error("two hashes of the same password must differ (random salt)")
	}
}

func TestVerifyPassword_InvalidStoredHash_ReturnsError(t *testing.T) {
	_, err := verifyPassword("password", "not-a-valid-hash")

	if err == nil {
		t.Error("expected error for invalid hash format")
	}
}

func TestParseHashString_DecisionTable(t *testing.T) {
	valid := mustHashPassword(t, "test-password")

	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid hash", valid, false},
		{"too few parts (3)", "a$b$c", true},
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
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseHashString(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestRegister_InputValidation_NoDBCalls(t *testing.T) {
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
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestAuthService(&mockRepo{})

			_, err := svc.Register(context.Background(), tc.username, tc.email, tc.password)

			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestRegister_EmailAlreadyTaken_ReturnsErrEmailExists(t *testing.T) {
	repo := &mockRepo{
		isEmailTaken: func(_ context.Context, _ string) (bool, error) { return true, nil },
	}
	svc := newTestAuthService(repo)

	_, err := svc.Register(context.Background(), "alice", "alice@example.com", "password1")

	if !errors.Is(err, ErrEmailExists) {
		t.Errorf("expected ErrEmailExists, got: %v", err)
	}
}

func TestRegister_DBErrorOnIsEmailTaken_PropagatesError(t *testing.T) {
	dbErr := fmt.Errorf("connection lost")
	repo := &mockRepo{
		isEmailTaken: func(_ context.Context, _ string) (bool, error) { return false, dbErr },
	}
	svc := newTestAuthService(repo)

	_, err := svc.Register(context.Background(), "alice", "alice@example.com", "password1")

	if !errors.Is(err, dbErr) {
		t.Errorf("expected wrapped dbErr, got: %v", err)
	}
}

func TestRegister_DBErrorOnCreateUser_PropagatesError(t *testing.T) {
	dbErr := fmt.Errorf("insert failed")
	repo := &mockRepo{
		isEmailTaken: func(_ context.Context, _ string) (bool, error) { return false, nil },
		createUser:   func(_ context.Context, _, _, _ string) (int, error) { return 0, dbErr },
	}
	svc := newTestAuthService(repo)

	_, err := svc.Register(context.Background(), "alice", "alice@example.com", "password1")

	if !errors.Is(err, dbErr) {
		t.Errorf("expected wrapped dbErr, got: %v", err)
	}
}

func TestRegister_Success_ReturnsUserID(t *testing.T) {
	repo := &mockRepo{
		isEmailTaken: func(_ context.Context, _ string) (bool, error) { return false, nil },
		createUser:   func(_ context.Context, _, _, _ string) (int, error) { return 42, nil },
	}
	svc := newTestAuthService(repo)

	id, err := svc.Register(context.Background(), "alice", "alice@example.com", "password1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Errorf("id: got %d, want 42", id)
	}
}

func TestRegister_Pairwise_TrimSpaceApplied(t *testing.T) {
	cases := []struct {
		name     string
		username string
		email    string
	}{
		{"both already trimmed", "alice", "alice@example.com"},
		{"both with whitespace", "  alice ", "  alice@example.com  "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedUsername, capturedEmail string
			repo := &mockRepo{
				isEmailTaken: func(_ context.Context, email string) (bool, error) {
					capturedEmail = email
					return false, nil
				},
				createUser: func(_ context.Context, username, email, _ string) (int, error) {
					capturedUsername = username
					return 1, nil
				},
			}
			svc := newTestAuthService(repo)

			_, _ = svc.Register(context.Background(), tc.username, tc.email, "password1")

			if capturedUsername != "alice" {
				t.Errorf("username not trimmed: got %q", capturedUsername)
			}
			if capturedEmail != "alice@example.com" {
				t.Errorf("email not trimmed: got %q", capturedEmail)
			}
		})
	}
}

func TestLogin_InputValidation_NoDBCalls(t *testing.T) {
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

func TestLogin_UserNotFound_PropagatesError(t *testing.T) {
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

func TestLogin_WrongPassword_ReturnsErrInvalidCredentials(t *testing.T) {
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

func TestLogin_CorruptedStoredHash_ReturnsError(t *testing.T) {
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

func TestLogin_Success_ReturnsUserAndSetsCookie(t *testing.T) {
	correctHash := mustHashPassword(t, "password1")
	repo := &mockRepo{
		getUserByEmail: func(_ context.Context, email string) (*users.User, error) {
			return &users.User{
				ID:       7,
				Username: "alice",
				Email:    email,
				Password: correctHash,
				Role:     "user",
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
	if u.Role != "user" {
		t.Errorf("role: got %q, want user", u.Role)
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

func TestLogout_WithCookie_DeletesSessionAndClearsCookie(t *testing.T) {
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

func TestLogout_WithoutCookie_DoesNotPanicAndClearsCookie(t *testing.T) {
	svc := newTestAuthService(&mockRepo{})
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	w := httptest.NewRecorder()

	svc.Logout(w, req)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 Set-Cookie header even without incoming cookie, got %d", len(cookies))
	}
}

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

func TestOptionalUserMiddleware_AuthenticatedRequest_InjectsUserIntoContext(t *testing.T) {
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

func TestOptionalUserMiddleware_UnauthenticatedRequest_CallsNextWithoutUser(t *testing.T) {
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

func TestRequireAuth_UnauthenticatedRequest_ReturnsUnauthorized(t *testing.T) {
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

func TestRequireRole_ForWrongRole_ReturnsForbidden(t *testing.T) {
	repo := &mockRepo{
		getUserByID: func(_ context.Context, id int) (*users.User, error) {
			return &users.User{
				ID:       id,
				Username: "alice",
				Email:    "alice@example.com",
				Role:     users.RoleUser,
			}, nil
		},
	}

	svc := newTestAuthService(repo)
	sessionID, _ := svc.sessionMgr.Create(42)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
	w := httptest.NewRecorder()

	nextCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		nextCalled = true
	})

	svc.RequireRole(users.RoleAdmin, users.RoleSuperuser)(next).ServeHTTP(w, req)

	if nextCalled {
		t.Error("next handler must not be called for forbidden request")
	}

	if w.Code != http.StatusForbidden {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestUserIDFromContext_EquivalenceClasses(t *testing.T) {
	cases := []struct {
		name   string
		ctx    context.Context
		wantID int
		wantOK bool
	}{
		{
			"with valid int value",
			context.WithValue(context.Background(), KeyUserID, 42),
			42, true,
		},
		{
			"without value",
			context.Background(),
			0, false,
		},
		{
			"with wrong type value",
			context.WithValue(context.Background(), KeyUserID, "not-an-int"),
			0, false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, ok := UserIDFromContext(tc.ctx)
			if id != tc.wantID || ok != tc.wantOK {
				t.Errorf("got (%d, %v), want (%d, %v)", id, ok, tc.wantID, tc.wantOK)
			}
		})
	}
}

func TestGetUserByID_DelegatesSuccessToRepo(t *testing.T) {
	want := &users.User{ID: 5, Username: "alice", Email: "a@b.com"}
	repo := &mockRepo{
		getUserByID: func(_ context.Context, id int) (*users.User, error) {
			return want, nil
		},
	}
	svc := newTestAuthService(repo)

	got, err := svc.GetUserByID(context.Background(), 5)

	if err != nil || got != want {
		t.Errorf("got (%v, %v), want (%v, nil)", got, err, want)
	}
}

func TestGetUserByID_DelegatesErrorToRepo(t *testing.T) {
	dbErr := fmt.Errorf("not found")
	repo := &mockRepo{
		getUserByID: func(_ context.Context, id int) (*users.User, error) {
			return nil, dbErr
		},
	}
	svc := newTestAuthService(repo)

	_, err := svc.GetUserByID(context.Background(), 99)

	if !errors.Is(err, dbErr) {
		t.Errorf("expected dbErr, got: %v", err)
	}
}

func strPtr(s string) *string { return &s }
