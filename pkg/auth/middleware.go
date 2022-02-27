package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	tyderrors "github.com/thankyoudiscord/api/pkg/errors"
	"github.com/thankyoudiscord/api/pkg/models"
)

func Authenticated(next http.Handler) http.Handler {
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
			return
		}

		if session == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// TODO: is there a better way to check if the application was revoked?
		user, err := models.GetUser(session.AccessToken)
		if err != nil {
			// The oauth token was revoked, so force the user to logout and delete the session
			if errors.Is(err, tyderrors.DiscordAPIUnauthorized) {
				mgr.DeleteSession(sessionId)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "session_id", sessionId)
		ctx = context.WithValue(ctx, "session", session)
		ctx = context.WithValue(ctx, "user", user)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
