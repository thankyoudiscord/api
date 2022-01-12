package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/thankyoudiscord/api/auth"
	"github.com/thankyoudiscord/api/database"
	"github.com/thankyoudiscord/api/models"
	"gorm.io/gorm"
)

type UserRoutes struct{}

func (ur UserRoutes) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(auth.RequireAuth)

	r.Get("/@me", getSelf)

	return r
}

type (
	GetUserPayloadSignature struct {
		HasSigned     bool    `json:"has_signed"`
		Position      *int64  `json:"position"`
		ReferralCount int64   `json:"referral_count"`
		ReferredBy    *string `json:"referred_by"`
	}

	GetUserPayload struct {
		User      models.DiscordUser      `json:"user"`
		Signature GetUserPayloadSignature `json:"signature"`
	}
)

func getSelf(w http.ResponseWriter, r *http.Request) {
	session := r.Context().Value("session").(*auth.Session)
	userId := session.UserID

	data, err := models.GetUser(session.AccessToken)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	db := database.GetDatabase()

	hasSigned := true

	sig := database.Signature{UserID: userId}
	res := db.First(&sig)
	if res.Error != nil {
		if res.Error == gorm.ErrRecordNotFound {
			hasSigned = false
		} else {

			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	var count int64 = 0
	res = db.Model(&database.Signature{}).Where("referrer_id = ?", userId).Count(&count)
	if res.Error != nil {
		fmt.Printf("failed to count refs: %v\n", res.Error)
		count = 0
	}

	pl := GetUserPayload{
		User: *data,
		Signature: GetUserPayloadSignature{
			HasSigned:     hasSigned,
			ReferralCount: count,
			ReferredBy:    sig.ReferrerID,
		},
	}

	position, _ := database.GetUserPosition(db, userId)
	if position != 0 {
		pl.Signature.Position = &position
	}

	b, err := json.Marshal(pl)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(b)
}
