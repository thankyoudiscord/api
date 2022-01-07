package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/joho/godotenv"
	"github.com/thankyoudiscord/api/routes"
)

var (
	ADDR          string
	CLIENT_ID     string
	CLIENT_SECRET string
	REDIRECT_URI  string

	REQUIRED_ENV = []string{
		"ADDR",
		"CLIENT_ID",
		"CLIENT_SECRET",
		"REDIRECT_URI",
	}
)

func init() {
	if err := godotenv.Load("./.env"); err != nil {
		panic(err)
	}

	fmt.Println(ADDR, CLIENT_ID, CLIENT_SECRET, REDIRECT_URI)

	missing := checkenv(REQUIRED_ENV)

	ADDR = os.Getenv("ADDR")
	CLIENT_ID = os.Getenv("CLIENT_ID")
	CLIENT_SECRET = os.Getenv("CLIENT_SECRET")
	REDIRECT_URI = os.Getenv("REDIRECT_URI")

	if len(missing) == 0 {
		return
	}

	log.Fatalf(
		"missing %v in env",
		strings.Join(missing, ", "),
	)
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Mount("/auth", routes.AuthRoutes{}.Routes())

	fmt.Printf("Starting web server on ADDR=%v\n", ADDR)
	if err := http.ListenAndServe(ADDR, r); err != nil {
		log.Fatalf("failed to start server: %v\n", err)
	}
}

func checkenv(keys []string) []string {
	var missing []string
	for _, key := range keys {
		if val, ok := os.LookupEnv(key); len(val) == 0 || !ok {
			missing = append(missing, key)
		}
	}

	return missing
}
