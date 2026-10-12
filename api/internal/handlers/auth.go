package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ticketin/internal/models"
	"ticketin/internal/services"
	"ticketin/internal/utils"
	"time"
	"uuid"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	DB    *sqlx.DB
	RDB   *redis.Client
	Email *services.EmailService
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type PendingUserRequest struct {
	Email        string `json:"email"`
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	OtpCode      string `json:"otp_code"`
}

type OtpRequest struct {
	Email   string `json:"email" validate:"required,email"`
	OtpCode string `json:"otp_code" validate:"required,len=6,numeric"`
}

type ExistingUser struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

func (h *AuthHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	var existing ExistingUser

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	if err := utils.Validate.Struct(req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": utils.FormatValidationError(err),
		})
		return
	}

	err := h.DB.Get(&existing, `SELECT email, username FROM users WHERE email = $1 OR username = $2 LIMIT 1`, req.Email, req.Username)
	if err == nil {
		if existing.Email == req.Email {
			http.Error(w, "Email already registered", http.StatusConflict)
			return
		}
		http.Error(w, "Username already been taken", http.StatusConflict)
		return
	}

	if err != sql.ErrNoRows {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Something wrong with generate password", http.StatusInternalServerError)
		return
	}
	hashPassword := string(hashBytes)

	otpCode, err := utils.GenerateOTP()
	if err != nil {
		log.Printf("error generting otp: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	pendingUserJSON, err := json.Marshal(&PendingUserRequest{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: hashPassword,
		OtpCode:      otpCode,
	})
	if err != nil {
		log.Printf("Failed to marshal pending user: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// sent redis
	key := "pending_user:" + req.Email
	err = h.RDB.Set(r.Context(), key, pendingUserJSON, 5*time.Minute).Err()
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		log.Printf("failed to store pending user to redis: %v", err)
		return
	}

	// send mail
	if err := h.Email.SendOTP(req.Email, otpCode); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		log.Printf("failed to send mail: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Verification code sent to your email",
	})
}

func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req OtpRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	if err := utils.Validate.Struct(req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": utils.FormatValidationError(err),
		})
		return
	}

	key := "pending_user:" + req.Email
	rawJSONredis, err := h.RDB.Get(r.Context(), key).Result()
	if err == redis.Nil {
		http.Error(w, "OTP Code already expired or registration session not found", http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("Failed to get key value from redis: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var pendingUser PendingUserRequest

	if err = json.Unmarshal([]byte(rawJSONredis), &pendingUser); err != nil {
		log.Printf("Failed to unmarshal raw json: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if pendingUser.OtpCode != req.OtpCode {
		http.Error(w, "Invalid OTP code", http.StatusUnauthorized)
		return
	}

	var newUserID uuid.UUID
	insertQuery := `INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3) RETURNING id`
	if err = h.DB.Get(&newUserID, insertQuery, pendingUser.Email, pendingUser.Username, pendingUser.PasswordHash); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Printf("Failed insert DB: %v", err)
		return
	}

	if err = h.RDB.Del(r.Context(), key).Err(); err != nil {
		log.Printf("failed to remove pending user from redis: %v", err)
	}

	accessToken, err := utils.GenerateAccessToken(newUserID, pendingUser.Email)
	if err != nil {
		log.Printf("Failed generate access token: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		log.Printf("Failed generate refresh token: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = h.RDB.Set(r.Context(), "refresh_token:"+refreshToken, newUserID.String(), 7*24*time.Hour).Err()
	if err != nil {
		log.Printf("Failed put refresh token on redis: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	utils.SetAuthCookie(w, accessToken, refreshToken)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "success",
	})
}

type LoginRequest struct {
	Identifier string `json:"identifier" validate:"required"`
	Password   string `json:"password" validate:"required,min=8,max=72"`
}

type LoginSession struct {
	ID      uuid.UUID `json:"id"`
	Email   string    `json:"email"`
	OtpCode string    `json:"otp_code"`
}

func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	if err := utils.Validate.Struct(req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": utils.FormatValidationError(err),
		})
		return
	}

	var user models.User

	searchQuery := `SELECT id, email, username, password_hash FROM users WHERE email = $1 OR username = $1`
	if err := h.DB.Get(&user, searchQuery, req.Identifier); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid Email/Username or Password", http.StatusUnauthorized)
			return
		}

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if user.PasswordHash == nil {
		http.Error(w, "This account registered with google, please login with google", http.StatusBadRequest)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid Email/Username or Password", http.StatusUnauthorized)
		return
	}

	otpCode, err := utils.GenerateOTP()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Failed generate OTP: %v", err)
		return
	}

	userJSON, err := json.Marshal(&LoginSession{
		ID:      user.ID,
		Email:   user.Email,
		OtpCode: otpCode,
	})
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Printf("Failed parse object to json: %v", err)
		return
	}

	err = h.Email.SendOTP(user.Email, otpCode)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Failed sending otp to email: %v", err)
		return
	}

	err = h.RDB.Set(r.Context(), "login_otp:"+user.Email, userJSON, 5*time.Minute).Err()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "OTP verification required",
		"email":   user.Email,
	})
}

type LoginVerifyOTPRequest struct {
	Email   string `json:"email" validate:"required,email"`
	OtpCode string `json:"otp_code" validate:"required,len=6,numeric"`
}

func (h *AuthHandler) LoginVerifyOTPHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginVerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	if err := utils.Validate.Struct(req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": utils.FormatValidationError(err),
		})
		return
	}

	userRawRedis, err := h.RDB.Get(r.Context(), "login_otp:"+req.Email).Result()
	if err != nil {
		if err == redis.Nil {
			http.Error(w, "Session already expired, please login again", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var userSession LoginSession
	if err := json.Unmarshal([]byte(userRawRedis), &userSession); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Printf("Failed parse json to object: %v", err)
		return
	}

	if userSession.OtpCode != req.OtpCode {
		http.Error(w, "Invalid OTP code", http.StatusUnauthorized)
		return
	}

	err = h.RDB.Del(r.Context(), "login_otp:"+req.Email).Err()
	if err != nil {
		log.Printf("Failed to delete redis entries: %v", err)
	}

	accessToken, err := utils.GenerateAccessToken(userSession.ID, userSession.Email)
	if err != nil {
		log.Printf("Failed to generate access token: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		log.Printf("Failed to generate refresh token: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = h.RDB.Set(r.Context(), "refresh_token:"+refreshToken, userSession.ID.String(), 7*24*time.Hour).Err()
	if err != nil {
		log.Printf("Failed put refresh token on redis: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	utils.SetAuthCookie(w, accessToken, refreshToken)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "verified",
	})
}

func (h *AuthHandler) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		http.Error(w, "Missing refresh token", http.StatusUnauthorized)
		return
	}
	oldRefreshToken := cookie.Value

	key := "refresh_token:" + oldRefreshToken
	userIDStr, err := h.RDB.Get(r.Context(), key).Result()
	if err == redis.Nil {
		utils.ClearAuthCookie(w)
		http.Error(w, "Invalid or expired refresh token", http.StatusUnauthorized)
		return
	}
	if err != nil {
		log.Printf("Failed to get refresh token from Redis: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Printf("Failed to parse user ID from Redis: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var user models.User
	if err := h.DB.Get(&user, `SELECT id, email FROM users WHERE id = $1 LIMIT 1`, userID); err != nil {
		utils.ClearAuthCookie(w)
		http.Error(w, "User not found or account deactivated", http.StatusUnauthorized)
		return
	}

	// 1. Invalidate old refresh token (Token Rotation)
	_ = h.RDB.Del(r.Context(), key).Err()

	// 2. Generate brand new pair of tokens
	newAccessToken, err := utils.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		log.Printf("Failed to generate access token on refresh: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	newRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		log.Printf("Failed to generate refresh token on refresh: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 3. Store new refresh token in Redis (7 days TTL)
	err = h.RDB.Set(r.Context(), "refresh_token:"+newRefreshToken, user.ID.String(), 7*24*time.Hour).Err()
	if err != nil {
		log.Printf("Failed to save new refresh token in Redis: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 4. Update HttpOnly cookies
	utils.SetAuthCookie(w, newAccessToken, newRefreshToken)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Tokens refreshed successfully",
	})
}

func (h *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		if err := h.RDB.Del(r.Context(), "refresh_token:"+cookie.Value).Err(); err != nil {
			log.Printf("Warning: Failed to delete refresh token from Redis on logout: %v", err)
		}
	}

	utils.ClearAuthCookie(w)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out successfully",
	})
}

