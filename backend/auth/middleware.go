package auth

import (
	"context"
	"log"
	"net/http"

	"gitlab.com/5130904-20104-teams/software-design-project/backend/httpapi"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/users"
)

const KeyUser contextKey = "auth_user"

func (s *AuthService) OptionalUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := s.VerifyRequest(r)
		if !ok || uid <= 0 {
			next.ServeHTTP(w, r)
			return
		}

		u, err := s.GetUserByID(r.Context(), uid)
		if err != nil {
			log.Printf("[auth] OptionalUserMiddleware: GetUserByID(%d): %v", uid, err)
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), KeyUserID, uid)
		ctx = context.WithValue(ctx, KeyUser, u)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *AuthService) RequireAuth(next http.Handler) http.Handler {
	return s.OptionalUserMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := UserFromContext(r.Context())
		if !ok || u == nil {
			httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized", "Необходимо войти в аккаунт")
			return
		}

		next.ServeHTTP(w, r)
	}))
}

func (s *AuthService) RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return s.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := UserFromContext(r.Context())
			if !ok || u == nil {
				httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized", "Необходимо войти в аккаунт")
				return
			}

			if _, ok := allowed[u.Role]; !ok {
				httpapi.WriteError(w, http.StatusForbidden, "forbidden", "Недостаточно прав")
				return
			}

			next.ServeHTTP(w, r)
		}))
	}
}

func UserFromContext(ctx context.Context) (*users.User, bool) {
	v := ctx.Value(KeyUser)
	if v == nil {
		return nil, false
	}

	u, ok := v.(*users.User)
	return u, ok
}
