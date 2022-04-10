package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgconn"
	"github.com/joho/godotenv"
	"github.com/thankyoudiscord/api/pkg/auth"
	"github.com/thankyoudiscord/api/pkg/cache"
	"github.com/thankyoudiscord/api/pkg/database"
	"github.com/thankyoudiscord/api/pkg/models"
	"github.com/thankyoudiscord/api/pkg/protos"
)

func init() {
	// TODO: is it okay to call this in every file?
	godotenv.Load()
}

type BannerRoutes struct {
	bannerGenClient protos.BannerClient
}

func NewBannerRoutes(bannerGenClient protos.BannerClient) *BannerRoutes {
	return &BannerRoutes{
		bannerGenClient: bannerGenClient,
	}
}

func (br BannerRoutes) Routes() chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(auth.Authenticated)

		r.Group(func(r chi.Router) {
			r.Use(httprate.Limit(2, 5*time.Minute, httprate.WithKeyFuncs(
				httprate.KeyByEndpoint,
				func(r *http.Request) (string, error) {
					var sessionID string
					sessionID = r.Context().Value("session_id").(string)
					return sessionID, nil
				})))

			r.Post("/sign", br.SignBanner)
		})

		r.Delete("/sign", br.UnsignBanner)
	})

	r.Get("/image.png", br.GenerateBanner)

	return r
}

func (br BannerRoutes) SignBanner(w http.ResponseWriter, r *http.Request) {
	var session *auth.Session
	var user *models.DiscordUser
	session = r.Context().Value("session").(*auth.Session)
	user = r.Context().Value("user").(*models.DiscordUser)
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

	if os.Getenv("APP_ENV") == "production" {
		solution := body.CaptchaSolution
		if body.CaptchaSolution == "" {
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
	}

	ref := body.Referrer
	if ref != nil && *ref != userId {
		matches, err := regexp.Match(`^\d{16,20}$`, []byte(*ref))
		if err == nil && matches {
			sig.ReferrerID = ref
		}
	}

	if ref != nil {
		var count int64
		db.Raw(`
			SELECT 1
			FROM signatures
			WHERE user_id = ?
			LIMIT 1
		`, ref).
			Count(&count)

		if count == 0 {
			ref = nil
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

		log.Printf("Failed to create signature: %v\n", res.Error)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bytes, err := json.Marshal(sig)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	pos, err := database.GetUserPosition(db, userId)
	if err != nil {
		log.Printf("failed to get user position for feed: %v\n", err)
	}

	sendSignatureFeedMessage(user, pos)
	addSignatureRoleToUser(user)

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

func (br BannerRoutes) GenerateBanner(w http.ResponseWriter, r *http.Request) {
	bannerCache := cache.GetBannerCache()

	b, shouldRegen, err := bannerCache.Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read banner image from cache: %v\n", err)
	}

	if b != nil {
		img := b.GetImage()
		if img != nil {
			w.Header().Add("Content-Type", "image/png")
			w.Write(img)
		}
	}

	var regend *protos.CreateBannerResponse
	var genError error

	if shouldRegen {
		regend, genError = br.bannerGenClient.GenerateBanner(
			context.Background(),
			&protos.CreateBannerRequest{},
		)

		if regend != nil && genError == nil {
			bannerCache.Set(regend)
		}
	}

	if genError != nil {
		fmt.Fprintf(os.Stderr, "failed to generate banner: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if b != nil {
		return
	}

	w.Header().Add("Content-Type", "image/png")
	w.Write(regend.GetImage())
	return
}

func sendSignatureFeedMessage(user *models.DiscordUser, position int64) {
	webhook, exists := os.LookupEnv("SIGNATURE_FEED_WEBHOOK")
	if !exists {
		return
	}

	postBody := map[string]interface{}{
		"content": fmt.Sprintf(
			":pencil: **%s#%s** signed the banner! (**#%v**)",
			user.Username,
			user.Discriminator,
			position,
		),
		"allowed_mentions": map[string]interface{}{
			"parse": []string{},
		},
	}

	j, _ := json.Marshal(&postBody)
	_, err := http.Post(webhook, "application/json", bytes.NewBuffer(j))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to post feed message: %v\n", err)
		return
	}
}

func addSignatureRoleToUser(user *models.DiscordUser) {
	discordToken, discordTokenExists := os.LookupEnv("DISCORD_TOKEN")
	signatureRole, signatureRoleExists := os.LookupEnv("SIGNATURE_ROLE")
	guildID, guildIDExists := os.LookupEnv("SIGNATURE_ROLE_GUILD_ID")

	if !discordTokenExists || !signatureRoleExists || !guildIDExists {
		return
	}

	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf(
			"https://discord.com/api/v10/guilds/%s/members/%s/roles/%s",
			guildID,
			user.ID,
			signatureRole,
		),
		nil,
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create PUT /guilds/:guild/members/:member/roles/:role request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+discordToken)

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to add role to user: %v\n", err)
		return
	}
}
