package server

import (
	"net/http"
	"ticketin/internal/handlers"
	ticketinmw "ticketin/internal/middleware"
	"ticketin/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func NewRouter(db *sqlx.DB, rdb *redis.Client, email *services.EmailService) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("API is running with live reload!"))
	})

	auth := &handlers.AuthHandler{DB: db, RDB: rdb, Email: email}
	user := &handlers.UserHandler{DB: db}

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

	return r
}
