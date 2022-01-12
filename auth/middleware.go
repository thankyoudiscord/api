package auth

import (
	"context"
	"fmt"
	"net/http"
)

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil {
			if err == http.ErrNoCookie {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			w.WriteHeader(http.StatusBadRequest)
			return
		}

		sessionId := c.Value

		session, err := mgr.GetSession(sessionId)
		if err != nil {
			fmt.Println("failed to get session:", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		if session == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "session_id", sessionId)
		ctx = context.WithValue(ctx, "session", session)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
