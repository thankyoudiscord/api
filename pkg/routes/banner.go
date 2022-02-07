package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/go-chi/chi"
	"github.com/jackc/pgconn"
	"github.com/thankyoudiscord/api/pkg/auth"
	"github.com/thankyoudiscord/api/pkg/database"
	"github.com/thankyoudiscord/api/pkg/models"
)

type BannerRoutes struct{}

func (br BannerRoutes) Routes() chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(auth.Authenticated)

		r.Post("/sign", br.SignBanner)
		r.Delete("/sign", br.UnsignBanner)
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

		log.Printf("Failed to create signature: %v\n", err)

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

func (br BannerRoutes) UnsignBanner(w http.ResponseWriter, r *http.Request) {
	session := r.Context().Value("session").(*auth.Session)
	userId := session.UserID

	db := database.GetDatabase()

	res := db.Where("user_id = ?", userId).Unscoped().Delete(&database.Signature{})
	if res.Error != nil {
		fmt.Printf("failed to delete from database: %v\n", res.Error)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}