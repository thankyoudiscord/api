package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/thankyoudiscord/api/pkg/auth"
	"github.com/thankyoudiscord/api/pkg/cache"
	"github.com/thankyoudiscord/api/pkg/database"
	"github.com/thankyoudiscord/api/pkg/protos"
	"github.com/thankyoudiscord/api/pkg/routes"
)

const DEFAULT_ADDR = ":3000"

var (
	redisClient *redis.Client

	ADDR,
	CLIENT_ID,
	CLIENT_SECRET,
	REDIRECT_URI,
	REDIS_HOST,
	REDIS_PORT,
	POSTGRES_HOST,
	POSTGRES_PORT,
	POSTGRES_USER,
	POSTGRES_PASSWORD,
	POSTGRES_DB,
	BANNER_GRPC_ADDR string

	REQUIRED_ENV = []string{
		"CLIENT_ID",
		"CLIENT_SECRET",
		"REDIRECT_URI",
		"REDIS_HOST",
		"REDIS_PORT",
		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_DB",
		"BANNER_GRPC_ADDR",
	}
)

func init() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	CLIENT_ID = os.Getenv("CLIENT_ID")
	CLIENT_SECRET = os.Getenv("CLIENT_SECRET")
	REDIRECT_URI = os.Getenv("REDIRECT_URI")
	REDIS_HOST = os.Getenv("REDIS_HOST")
	REDIS_PORT = os.Getenv("REDIS_PORT")
	POSTGRES_HOST = os.Getenv("POSTGRES_HOST")
	POSTGRES_PORT = os.Getenv("POSTGRES_PORT")
	POSTGRES_USER = os.Getenv("POSTGRES_USER")
	POSTGRES_PASSWORD = os.Getenv("POSTGRES_PASSWORD")
	POSTGRES_DB = os.Getenv("POSTGRES_DB")
	BANNER_GRPC_ADDR = os.Getenv("BANNER_GRPC_ADDR")

	var ok bool
	ADDR, ok = os.LookupEnv("ADDR")
	if !ok {
		ADDR = DEFAULT_ADDR
	}

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
	cache.InitBannerCache(redisClient)

	pgConnUrl := url.URL{
		User:   url.UserPassword(POSTGRES_USER, POSTGRES_PASSWORD),
		Scheme: "postgres",
		Host:   POSTGRES_HOST + ":" + POSTGRES_PORT,
		Path:   POSTGRES_DB,
		RawQuery: url.Values{
			"sslmode":  {"disable"},
			"TimeZone": {"America/New_York"},
		}.Encode(),
	}

	d, err := gorm.Open(postgres.Open(pgConnUrl.String()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v\n", err)
	}

	database.InitDatabase(d)
}

func main() {
	bannerGRPCConn, err := grpc.Dial(
		BANNER_GRPC_ADDR,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to dial banner grpc server: %v\n", err)
		os.Exit(1)
	}

	bannerGenClient := protos.NewBannerClient(bannerGRPCConn)

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Use(httprate.Limit(
		15,
		10*time.Second,
		httprate.WithKeyFuncs(httprate.KeyByEndpoint, httprate.KeyByIP),
	))

	ALLOWED_ORIGINS := []string{"*"}

	if os.Getenv("APP_ENV") == "production" {
		ALLOWED_ORIGINS = []string{
			"localhost:*",
			"*.wah.wtf",
			"*.thankyoudiscord.com",
		}
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   ALLOWED_ORIGINS,
		AllowCredentials: true,
	}))

	r.Mount("/", routes.AuthRoutes{}.Routes())

	r.Mount("/banner", routes.NewBannerRoutes(bannerGenClient).Routes())
	r.Mount("/users", routes.UserRoutes{}.Routes())

	r.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
		db := database.GetDatabase()

		var count int64

		res := db.Model(&database.Signature{}).Count(&count)
		if res.Error != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp := map[string]int64{
			"signatures": count,
		}

		bytes, err := json.Marshal(resp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(bytes)
	})

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
