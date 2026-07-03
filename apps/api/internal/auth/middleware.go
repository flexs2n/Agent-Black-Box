package auth

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/blackbox-agentdiff/api/internal/store"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const ProjectIDKey = contextKey("projectID")

func ProjectIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ProjectIDKey).(string)
	return v, ok
}

func Middleware(st store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, "missing Authorization header", http.StatusUnauthorized)
				return
			}
			raw := strings.TrimPrefix(auth, "Bearer ")
			prefix := raw
			if len(raw) > 12 {
				prefix = raw[:12]
			}
			key, err := st.APIKeyGetByPrefix(r.Context(), prefix)
			if err != nil {
				if err == sql.ErrNoRows {
					http.Error(w, "invalid API key", http.StatusUnauthorized)
					return
				}
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if err := bcrypt.CompareHashAndPassword([]byte(key.KeyHash), []byte(raw)); err != nil {
				http.Error(w, "invalid API key", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ProjectIDKey, key.ProjectID)
			_ = st.APIKeyMarkUsed(ctx, key.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
