package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ticketin/internal/middleware"
	"ticketin/internal/models"

	"github.com/jmoiron/sqlx"
)

type UserHandler struct {
	DB *sqlx.DB
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID := claims.ID
	userEmail := claims.Email

	query := `SELECT * FROM users WHERE id = $1 AND email = $2 LIMIT 1`

	var user models.User
	err := h.DB.Get(&user, query, userID, userEmail)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Printf("Failed searching user from claims: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}
