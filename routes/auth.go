package routes

import (
	"net/http"

	"github.com/go-chi/chi"
)

type AuthRoutes struct{}

func (ar AuthRoutes) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/cb", ar.Callback)
	r.Post("/login", ar.Login)
	r.Post("/logout", ar.Logout)

	return r
}

func (ar AuthRoutes) Callback(w http.ResponseWriter, r *http.Request) {}
func (ar AuthRoutes) Login(w http.ResponseWriter, r *http.Request)    {}
func (ar AuthRoutes) Logout(w http.ResponseWriter, r *http.Request)   {}
