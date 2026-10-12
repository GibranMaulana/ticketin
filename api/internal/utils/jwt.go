package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(id uuid.UUID, email string) (string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))

	claims := CustomClaims{
		ID:    id,
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func VerifyAccessToken(tokenString string) (*CustomClaims, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}

		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("Invalid access token")
}

func getCookieConfig() (bool, http.SameSite) {
	isProduction := os.Getenv("APP_ENV") == "production"
	if isProduction {
		// When frontend (e.g. Vercel) and backend (e.g. Cloud Run) are on different domains,
		// modern browsers require SameSite=None and Secure=true for cross-site credentials.
		return true, http.SameSiteNoneMode
	}
	return false, http.SameSiteLaxMode
}

func SetAuthCookie(w http.ResponseWriter, accessToken string, refreshToken string) {
	secure, sameSite := getCookieConfig()

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   15 * 60,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   7 * 24 * 60 * 60,
	})
}

func ClearAuthCookie(w http.ResponseWriter) {
	secure, sameSite := getCookieConfig()

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   -1,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   -1,
	})
}

