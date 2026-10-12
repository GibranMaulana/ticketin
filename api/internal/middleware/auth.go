package middleware

import (
	"context"
	"net/http"
	"strings"
	"ticketin/internal/utils"
)

type contextKey string

const ClaimsKey contextKey = "user_claims"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenString string
		cookie, err := r.Cookie("access_token")
		if err == nil {
			tokenString = cookie.Value
		}

		if tokenString == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenString == "" {
			http.Error(w, "Missing access token", http.StatusUnauthorized)
			return
		}

		claims, err := utils.VerifyAccessToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid access token or already expired", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsKey, claims)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetClaims(ctx context.Context) (*utils.CustomClaims, bool) {
	claims, ok := ctx.Value(ClaimsKey).(*utils.CustomClaims)
	return claims, ok
}
