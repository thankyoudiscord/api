package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/go-chi/chi"
	"github.com/jackc/pgconn"
	"github.com/thankyoudiscord/api/auth"
	"github.com/thankyoudiscord/api/database"
	"github.com/thankyoudiscord/api/models"
)

type BannerRoutes struct{}

func (br BannerRoutes) Routes() chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth)

		r.Post("/sign", br.SignBanner)
	})

	return r
}

func (br BannerRoutes) SignBanner(w http.ResponseWriter, r *http.Request) {
	session := r.Context().Value("session").(*auth.Session)
	userId := session.UserID

	db := database.GetDatabase()
	sig := database.Signature{
		UserID: userId,
	}

	body := make(map[string]interface{})
	err := json.NewDecoder(r.Body).Decode(&body)

	if err == nil {
		ref, ok := body["referrer"].(string)
		if ok {
			matches, err := regexp.Match(`^\d{16,20}$`, []byte(ref))
			if err == nil && matches {
				sig.ReferrerID = &ref
			}
		}
	}

	res := db.Create(&sig)
	if res.Error != nil {
		var e *pgconn.PgError
		if errors.As(res.Error, &e) {
			if e.Code == "23505" {
				w.WriteHeader(http.StatusUnprocessableEntity)
				w.Write(models.CreateError("You have already signed the banner"))
				return
			}
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bytes, err := json.Marshal(sig)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(bytes)
}
