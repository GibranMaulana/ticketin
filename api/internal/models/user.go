package models

import (
	"time"
	"uuid"
)

type User struct {
	ID        uuid.UUID `db:"id" json:"id"`
	Email     string    `db:"email" json:"email"`
	Username  string    `db:"username" json:"username"`
	Fullname  *string   `db:"full_name" json:"full_name"`
	AvatarUrl *string   `db:"avatar_url" json:"avatar_url"`

	PasswordHash *string `db:"password_hash" json:"-"`
	GoogleID     *string `db:"google_id" json:"google_id"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
