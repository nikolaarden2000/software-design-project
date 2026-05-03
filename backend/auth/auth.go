package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gitlab.com/5130904-20104-teams/software-design-project/backend/users"
	"golang.org/x/crypto/argon2"
)

var (
	ErrEmailExists          = errors.New("auth: email address already exists")
	ErrEmptyEmail           = errors.New("auth: email must not be empty")
	ErrInvalidEmail         = errors.New("auth: email address has invalid format")
	ErrEmptyUsername        = errors.New("auth: username must not be empty")
	ErrPasswordTooShort     = fmt.Errorf("auth: password must be at least %d characters long", minPasswordLen)
	ErrPasswordTooLong      = fmt.Errorf("auth: password must not exceed %d characters", maxPasswordLen)
	ErrPasswordInvalidChars = errors.New("auth: password may only contain Latin letters, numbers and the following special characters: . , | / & % # @ < > : * + -")

	ErrNoUser             = errors.New("auth: user with given email not found")
	ErrInvalidCredentials = errors.New("auth: credentials provided are invalid")
)

const (
	argonTime    uint32 = 1
	argonMemory  uint32 = 64 * 1024
	argonThreads uint8  = 4
	argonKeyLen  uint32 = 32

	saltLen = 32

	hashVersion   = 1
	hashSeparator = "$"

	minPasswordLen = 8
	maxPasswordLen = 64
)

var allowedPasswordRe = regexp.MustCompile(`^[a-zA-Z0-9.,|/&%#@<>:*+-]+$`)

func validatePassword(password string) error {
	switch {
	case len(password) < minPasswordLen:
		return ErrPasswordTooShort
	case len(password) > maxPasswordLen:
		return ErrPasswordTooLong
	case !allowedPasswordRe.MatchString(password):
		return ErrPasswordInvalidChars
	}
	return nil
}

type UserRepo interface {
	CreateUserWithRole(ctx context.Context, username, email, passwordHash, role string) (int, error)
	GetUserByEmail(ctx context.Context, email string) (*users.User, error)
	IsEmailTaken(ctx context.Context, email string) (bool, error)
	GetUserByID(ctx context.Context, id int) (*users.User, error)
}

type AuthService struct {
	userRepo       UserRepo
	sessionMgr     *SessionManager
	cookieName     string
	cookieDomain   string
	cookieSecure   bool
	cookieSameSite http.SameSite
}

func NewAuthService(userRepo UserRepo, cookieName string, ttl, cleanupInterval time.Duration) *AuthService {
	if cookieName == "" {
		cookieName = "session_id"
	}
	return &AuthService{
		userRepo:       userRepo,
		sessionMgr:     NewSessionManager(ttl, cleanupInterval),
		cookieName:     cookieName,
		cookieSecure:   false,
		cookieSameSite: http.SameSiteLaxMode,
	}
}

func (s *AuthService) SetCookieSettings(domain string, secure bool, sameSite http.SameSite) {
	s.cookieDomain = domain
	s.cookieSecure = secure
	s.cookieSameSite = sameSite
}

func (s *AuthService) Shutdown() {
	s.sessionMgr.Stop()
}

func (s *AuthService) newCookie(value string, maxAge int) *http.Cookie {
	c := &http.Cookie{
		Name:     s.cookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: s.cookieSameSite,
		MaxAge:   maxAge,
	}
	if s.cookieDomain != "" {
		c.Domain = s.cookieDomain
	}
	return c
}

func (s *AuthService) Register(ctx context.Context, username, email, password string) (int, error) {
	return s.RegisterWithRole(ctx, username, email, password, users.RoleUser)
}

func (s *AuthService) RegisterWithRole(ctx context.Context, username, email, password, role string) (int, error) {
	if !users.IsValidRole(role) {
		return 0, fmt.Errorf("auth: invalid role: %s", role)
	}

	username = strings.TrimSpace(username)
	if username == "" {
		return 0, ErrEmptyUsername
	}

	email = strings.TrimSpace(email)
	if email == "" {
		return 0, ErrEmptyEmail
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return 0, ErrInvalidEmail
	}

	if err := validatePassword(password); err != nil {
		return 0, err
	}

	taken, err := s.userRepo.IsEmailTaken(ctx, email)
	if err != nil {
		return 0, fmt.Errorf("auth: checking email uniqueness: %w", err)
	}
	if taken {
		return 0, ErrEmailExists
	}

	hash, err := hashPassword(password)
	if err != nil {
		return 0, fmt.Errorf("auth: hashing password: %w", err)
	}

	id, err := s.userRepo.CreateUserWithRole(ctx, username, email, hash, role)
	if err != nil {
		return 0, fmt.Errorf("auth: creating user with role: %w", err)
	}

	log.Printf("[auth] user registered: id=%d email=%s role=%s", id, email, role)
	return id, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string, w http.ResponseWriter) (*users.User, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, ErrEmptyEmail
	}
	if password == "" {
		return nil, ErrInvalidCredentials
	}

	u, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("auth: fetching user: %w", err)
	}

	ok, err := verifyPassword(password, u.Password)
	if err != nil {
		return nil, fmt.Errorf("auth: verifying password: %w", err)
	}
	if !ok {
		return nil, ErrInvalidCredentials
	}

	sessionID, err := s.sessionMgr.Create(u.ID)
	if err != nil {
		return nil, fmt.Errorf("auth: creating session: %w", err)
	}

	http.SetCookie(w, s.newCookie(sessionID, int(s.sessionMgr.ttl.Seconds())))
	log.Printf("[auth] user logged in: id=%d email=%s", u.ID, u.Email)

	return u, nil
}

func (s *AuthService) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(s.cookieName); err == nil {
		s.sessionMgr.Delete(c.Value)
	}
	del := s.newCookie("", -1)
	del.Expires = time.Unix(1, 0)
	http.SetCookie(w, del)
}

func (s *AuthService) VerifyRequest(r *http.Request) (int, bool) {
	c, err := r.Cookie(s.cookieName)
	if err != nil {
		return 0, false
	}
	return s.sessionMgr.Get(c.Value)
}

func (s *AuthService) GetUserByID(ctx context.Context, id int) (*users.User, error) {
	return s.userRepo.GetUserByID(ctx, id)
}

type contextKey string

const KeyUserID contextKey = "auth_user_id"

func UserIDFromContext(ctx context.Context) (int, bool) {
	v := ctx.Value(KeyUserID)
	if v == nil {
		return 0, false
	}
	id, ok := v.(int)
	return id, ok
}

func hashPassword(password string) (string, error) {
	salt, err := genRandomBytes(saltLen)
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	encoded := strings.Join([]string{
		strconv.Itoa(hashVersion),
		strconv.FormatUint(uint64(argonTime), 10),
		strconv.FormatUint(uint64(argonMemory), 10),
		strconv.FormatUint(uint64(argonThreads), 10),
		strconv.FormatUint(uint64(argonKeyLen), 10),
		hex.EncodeToString(salt),
		hex.EncodeToString(hash),
	}, hashSeparator)

	return encoded, nil
}

type parsedHash struct {
	version int
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
	salt    []byte
	hash    []byte
}

func parseU32Field(s, field string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("auth: invalid %s field: %w", field, err)
	}
	return uint32(v), nil
}

func parseU8Field(s, field string) (uint8, error) {
	v, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("auth: invalid %s field: %w", field, err)
	}
	return uint8(v), nil
}

func parseHashString(stored string) (*parsedHash, error) {
	parts := strings.Split(stored, hashSeparator)
	if len(parts) != 7 {
		return nil, fmt.Errorf("auth: stored hash has invalid format: expected 7 fields, got %d", len(parts))
	}

	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("auth: invalid hash version field: %w", err)
	}
	if version != hashVersion {
		return nil, fmt.Errorf("auth: unsupported hash version %d (current: %d)", version, hashVersion)
	}

	t, err := parseU32Field(parts[1], "time")
	if err != nil {
		return nil, err
	}
	m, err := parseU32Field(parts[2], "memory")
	if err != nil {
		return nil, err
	}
	p, err := parseU8Field(parts[3], "threads")
	if err != nil {
		return nil, err
	}
	kl, err := parseU32Field(parts[4], "keyLen")
	if err != nil {
		return nil, err
	}

	salt, err := hex.DecodeString(parts[5])
	if err != nil {
		return nil, fmt.Errorf("auth: invalid salt encoding: %w", err)
	}
	hashBytes, err := hex.DecodeString(parts[6])
	if err != nil {
		return nil, fmt.Errorf("auth: invalid hash encoding: %w", err)
	}

	return &parsedHash{
		version: version,
		time:    t,
		memory:  m,
		threads: p,
		keyLen:  kl,
		salt:    salt,
		hash:    hashBytes,
	}, nil
}

func verifyPassword(password, stored string) (bool, error) {
	ph, err := parseHashString(stored)
	if err != nil {
		return false, err
	}

	candidate := argon2.IDKey([]byte(password), ph.salt, ph.time, ph.memory, ph.threads, ph.keyLen)
	if subtle.ConstantTimeCompare(ph.hash, candidate) != 1 {
		return false, nil
	}
	return true, nil
}

func genRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}
