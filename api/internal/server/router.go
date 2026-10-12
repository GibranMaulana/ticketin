package server

import (
	"net/http"
	"os"
	"strings"
	"ticketin/internal/handlers"
	ticketinmw "ticketin/internal/middleware"
	"ticketin/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func NewRouter(db *sqlx.DB, rdb *redis.Client, email *services.EmailService) *chi.Mux {
	r := chi.NewRouter()

	allowedOrigins := []string{"http://localhost:5173", "http://localhost:3000"}
	if envOrigins := os.Getenv("CORS_ALLOWED_ORIGINS"); envOrigins != "" {
		for _, o := range strings.Split(envOrigins, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				allowedOrigins = append(allowedOrigins, o)
			}
		}
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Not Found"}` + "\n"))
	})

	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("API is running with live reload!"))
	}
	r.Get("/", healthHandler)

	auth := &handlers.AuthHandler{DB: db, RDB: rdb, Email: email}
	user := &handlers.UserHandler{DB: db}

	routes := func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Route("/auth", func(r chi.Router) {
				r.Post("/register", auth.RegisterHandler)
				r.Post("/verify-otp", auth.VerifyOTP)
				r.Post("/login", auth.LoginHandler)
				r.Post("/login/verify-otp", auth.LoginVerifyOTPHandler)
				r.Post("/refresh", auth.RefreshHandler)
				r.Post("/logout", auth.LogoutHandler)
			})

			r.Group(func(r chi.Router) {
				r.Use(ticketinmw.AuthMiddleware)
				r.Get("/me", user.GetMe)
			})
		})
	}

	r.Group(routes)
	r.Route("/api", routes)

	return r
}
