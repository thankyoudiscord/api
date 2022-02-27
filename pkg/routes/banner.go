package routes

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/go-chi/chi"
	"github.com/jackc/pgconn"
	"github.com/joho/godotenv"
	"github.com/thankyoudiscord/api/pkg/auth"
	"github.com/thankyoudiscord/api/pkg/database"
	"github.com/thankyoudiscord/api/pkg/models"
)

func init() {
	// TODO: is it okay to call this in every file?
	godotenv.Load()
}

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

	var body struct {
		Referrer        *string `json:"referrer"`
		CaptchaSolution string  `json:"captchaSolution"`
	}
	err := json.NewDecoder(r.Body).Decode(&body)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(models.CreateError("Failed to parse JSON payload"))
		return
	}

	ref := body.Referrer
	if body.Referrer != nil {
		matches, err := regexp.Match(`^\d{16,20}$`, []byte(*ref))
		if err == nil && matches {
			sig.ReferrerID = ref
		}
	}

	solution := body.CaptchaSolution
	if body.CaptchaSolution != "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(models.CreateError("Failed to read capcha solution from payload"))
		return
	}

	captchaVerified := verifyCaptcha(solution)
	if !captchaVerified {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(models.CreateError("Captcha verification failed"))
		return
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

		log.Printf("Failed to create signature: %v\n", res.Error)

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

func verifyCaptcha(sol string) bool {
	secret := os.Getenv("FRIENDLY_CAPTCHA_SECRET")
	verifyUrl := os.Getenv("FRIENDLY_CAPTCHA_VERIFY_URL")

	payload := struct {
		Solution string `json:"solution"`
		Secret   string `json:"secret"`
	}{
		Solution: sol,
		Secret:   secret,
	}

	pl, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("failed to serialize json payload for captcha verification:", err)
		return false
	}

	req, _ := http.NewRequest("POST", verifyUrl, bytes.NewBuffer(pl))
	req.Header.Add("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("failed to verify captcha solution: %v\n", err)
		return false
	}

	bdy, _ := io.ReadAll(resp.Body)

	var body struct {
		Success bool     `json:"success"`
		Errors  []string `json:"errors"`
	}

	err = json.Unmarshal(bdy, &body)
	if err != nil {
		fmt.Printf("failed to parse JSON response: %v\n", err)
		return false
	}

	if body.Success {
		return true
	}

	fmt.Printf("friendlycaptcha responded with errors: %v\n", body.Errors)
	return false
}
