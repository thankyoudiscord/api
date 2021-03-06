package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"gorm.io/gorm"

	"github.com/thankyoudiscord/api/pkg/auth"
	"github.com/thankyoudiscord/api/pkg/database"
	"github.com/thankyoudiscord/api/pkg/models"
)

type UserRoutes struct{}

func (ur UserRoutes) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(auth.Authenticated)

	r.Get("/@me", ur.GetSelf)

	return r
}

type (
	GetUserPayloadSignature struct {
		HasSigned     bool    `json:"has_signed"`
		Position      int64   `json:"position"`
		ReferralCount int64   `json:"referral_count"`
		ReferredBy    *string `json:"referred_by"`
	}

	GetUserPayload struct {
		User      models.DiscordUser      `json:"user"`
		Signature GetUserPayloadSignature `json:"signature"`
	}
)

func (ur UserRoutes) GetSelf(w http.ResponseWriter, r *http.Request) {
	session := r.Context().Value("session").(*auth.Session)
	userId := session.UserID

	data, ok := r.Context().Value("user").(*models.DiscordUser)
	if !ok {
		fmt.Println("failed to read \"user\" from request context")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	db := database.GetDatabase()

	hasSigned := false

	sig := database.Signature{}
	res := db.Where("user_id = ?", userId).Find(&sig)
	if res.Error != nil {
		if res.Error == gorm.ErrRecordNotFound {
			hasSigned = false
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// TODO: is there a better way to check if the record exists?
	if !sig.CreatedAt.IsZero() {
		hasSigned = true
	}

	var count int64 = 0
	re := db.Model(&database.Signature{}).Where("referrer_id = ?", userId).Count(&count)
	if re.Error != nil {
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
	pl.Signature.Position = position

	b, err := json.Marshal(pl)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(b)
}
