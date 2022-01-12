package routes

import (
	"io/ioutil"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/thankyoudiscord/api/auth"
)

type UserRoutes struct{}

func (ur UserRoutes) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(auth.RequireAuth)

	r.Get("/@me", getSelf)

	return r
}

func getSelf(w http.ResponseWriter, r *http.Request) {
	session := r.Context().Value("session").(*auth.Session)

	req, _ := http.NewRequest("GET", "https://discord.com/api/v9/users/@me", nil)
	req.Header.Add("Authorization", "Bearer "+session.AccessToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, _ := ioutil.ReadAll(res.Body)

	w.Write(body)
}
