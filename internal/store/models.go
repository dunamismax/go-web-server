// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0

package store

import (
	"database/sql"
)

type User struct {
	ID        int64          `db:"id" json:"id"`
	Email     string         `db:"email" json:"email"`
	Name      string         `db:"name" json:"name"`
	AvatarUrl sql.NullString `db:"avatar_url" json:"avatar_url"`
	Bio       sql.NullString `db:"bio" json:"bio"`
	IsActive  sql.NullBool   `db:"is_active" json:"is_active"`
	CreatedAt sql.NullTime   `db:"created_at" json:"created_at"`
	UpdatedAt sql.NullTime   `db:"updated_at" json:"updated_at"`
}
