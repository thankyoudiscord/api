package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"

	"github.com/thankyoudiscord/api/auth"
	"github.com/thankyoudiscord/api/routes"
)

const SESSION_ID_COOKIE = "session_id"

var (
	redisClient *redis.Client

	ADDR          string
	CLIENT_ID     string
	CLIENT_SECRET string
	REDIRECT_URI  string
	JWT_SECRET    string
	REDIS_HOST    string
	REDIS_PORT    string

	REQUIRED_ENV = []string{
		"ADDR",
		"CLIENT_ID",
		"CLIENT_SECRET",
		"REDIRECT_URI",
		"REDIS_HOST",
		"REDIS_PORT",
	}
)

func init() {
	if err := godotenv.Load("./.env"); err != nil {
		panic(err)
	}

	ADDR = os.Getenv("ADDR")
	CLIENT_ID = os.Getenv("CLIENT_ID")
	CLIENT_SECRET = os.Getenv("CLIENT_SECRET")
	REDIRECT_URI = os.Getenv("REDIRECT_URI")
	REDIS_HOST = os.Getenv("REDIS_HOST")
	REDIS_PORT = os.Getenv("REDIS_PORT")

	missing := checkenv(REQUIRED_ENV)

	if len(missing) != 0 {
		log.Fatalf(
			"missing %v in env",
			strings.Join(missing, ", "),
		)
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr: REDIS_HOST + ":" + REDIS_PORT,
	})

	auth.InitAuthManager(redisClient)
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Mount("/api", routes.AuthRoutes{}.Routes())
	r.Mount("/api/users", routes.UserRoutes{}.Routes())

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
