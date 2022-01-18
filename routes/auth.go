package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/joho/godotenv"
	"github.com/thankyoudiscord/api/auth"
	"github.com/thankyoudiscord/api/database"
	"github.com/thankyoudiscord/api/models"
	"golang.org/x/oauth2"
)

type AuthRoutes struct{}

var oauthConf *oauth2.Config

func init() {
	godotenv.Load()
	oauthConf = &oauth2.Config{
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		RedirectURL:  os.Getenv("REDIRECT_URI"),
		Scopes:       []string{"identify"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://discord.com/api/oauth2/authorize",
			TokenURL: "https://discord.com/api/oauth2/token",
		},
	}

}

func (ar AuthRoutes) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/login", ar.Login)
	r.Group(func(r chi.Router) {
		r.Use(auth.Authenticated)
		r.Post("/logout", ar.Logout)
	})

	return r
}

type LoginPayload struct {
	Code string `json:"code"`
}

func (ar AuthRoutes) Login(w http.ResponseWriter, r *http.Request) {
	var pl LoginPayload
	err := json.NewDecoder(r.Body).Decode(&pl)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	code := pl.Code

	if len(code) == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
		return
	}

	ctx := context.Background()

	tok, err := oauthConf.Exchange(ctx, code)
	if err != nil {
		log.Printf("failed to exchange code: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userData, err := models.GetUser(tok.AccessToken)
	if err != nil {
		fmt.Printf("failed to get user from discord: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	mgr := auth.GetManager()

	sID, err := mgr.CreateSession(auth.Session{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		UserID:       userData.ID,
	})

	if err != nil {
		fmt.Println("failed to save session in redis:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dbUser := database.User{
		UserID:        userData.ID,
		Username:      userData.Username,
		Discriminator: userData.Discriminator,
		AvatarHash:    userData.Avatar,
	}

	db := database.GetDatabase()
	res := db.Model(&dbUser).Where("user_id = ?", dbUser.UserID).Updates(&dbUser)
	if res.RowsAffected == 0 {
		res = db.Create(&dbUser)
	}

	if res.Error != nil {
		fmt.Printf("failed to update user data in database: %v\n", res.Error)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.SESSION_ID_COOKIE,
		Value:    sID,
		Path:     "/",
		Expires:  time.Now().Add(auth.SESSION_TTL),
		HttpOnly: true,
	})
}

func (ar AuthRoutes) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   auth.SESSION_ID_COOKIE,
		MaxAge: -1,
	})

	sIdCookie, err := r.Cookie(auth.SESSION_ID_COOKIE)
	if err != nil {
		return
	}

	sId := sIdCookie.Value

	if len(sId) == 0 {
		return
	}

	mgr := auth.GetManager()
	mgr.DeleteSession(sId)
}
