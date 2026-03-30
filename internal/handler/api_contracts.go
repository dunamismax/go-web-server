package handler

import (
	"time"

	"github.com/dunamismax/go-web-server/internal/middleware"
	"github.com/dunamismax/go-web-server/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

const (
	csrfHeaderName = "X-CSRF-Token"
	csrfFormField  = "csrf_token"
)

type apiCSRFContract struct {
	Header    string `json:"header"`
	FormField string `json:"form_field"`
	Token     string `json:"token"`
}

type apiSessionUser struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

type apiAuthStateResponse struct {
	Authenticated bool            `json:"authenticated"`
	User          *apiSessionUser `json:"user"`
	CSRF          apiCSRFContract `json:"csrf"`
}

type apiAuthMutationResponse struct {
	Message string          `json:"message"`
	User    *apiSessionUser `json:"user,omitempty"`
}

type apiUser struct {
	ID        int64   `json:"id"`
	Email     string  `json:"email"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url"`
	Bio       *string `json:"bio"`
	IsActive  bool    `json:"is_active"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type apiUserListResponse struct {
	Users []apiUser `json:"users"`
	Count int       `json:"count"`
}

type apiUserResponse struct {
	User apiUser `json:"user"`
}

type apiUserMutationResponse struct {
	Message string  `json:"message"`
	User    apiUser `json:"user"`
}

type apiUserCountResponse struct {
	Count int64 `json:"count"`
}

type apiDeleteUserResponse struct {
	ID      int64  `json:"id"`
	Deleted bool   `json:"deleted"`
	Message string `json:"message"`
}

func writeJSON(c echo.Context, status int, payload any) error {
	setupCSRFHeaders(c)
	return c.JSON(status, payload)
}

func currentCSRFContract(c echo.Context) apiCSRFContract {
	token := setupCSRFHeaders(c)

	return apiCSRFContract{
		Header:    csrfHeaderName,
		FormField: csrfFormField,
		Token:     token,
	}
}

func apiSessionUserFromMiddleware(user middleware.User) *apiSessionUser {
	return &apiSessionUser{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.Name,
		IsActive: user.IsActive,
	}
}

func apiUserFromStore(user store.User) apiUser {
	return apiUser{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		AvatarURL: user.AvatarUrl,
		Bio:       user.Bio,
		IsActive:  user.IsActive != nil && *user.IsActive,
		CreatedAt: formatAPITimestamp(user.CreatedAt),
		UpdatedAt: formatAPITimestamp(user.UpdatedAt),
	}
}

func apiUsersFromStore(users []store.User) []apiUser {
	items := make([]apiUser, 0, len(users))
	for _, user := range users {
		items = append(items, apiUserFromStore(user))
	}

	return items
}

func formatAPITimestamp(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}

	return ts.Time.UTC().Format(time.RFC3339)
}
