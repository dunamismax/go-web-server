package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/dunamismax/go-web-server/internal/middleware"
	"github.com/dunamismax/go-web-server/internal/store"
	"github.com/dunamismax/go-web-server/internal/view"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

type userDataStore interface {
	ListUsers(ctx context.Context) ([]store.User, error)
	CountUsers(ctx context.Context) (int64, error)
	GetUser(ctx context.Context, id int64) (store.User, error)
	CreateUser(ctx context.Context, params store.CreateUserParams) (store.User, error)
	UpdateUser(ctx context.Context, params store.UpdateUserParams) (store.User, error)
	UpdateUserPassword(ctx context.Context, params store.UpdateUserPasswordParams) (store.User, error)
	DeactivateUser(ctx context.Context, id int64) error
	DeactivateUserChecked(ctx context.Context, id int64) (bool, error)
	DeleteUser(ctx context.Context, id int64) error
	DeleteUserChecked(ctx context.Context, id int64) (bool, error)
}

type userPasswordService interface {
	HashPasswordArgon2(password string) (string, error)
}

// UserHandler handles all user-related HTTP requests including CRUD operations.
type UserHandler struct {
	store       userDataStore
	authService userPasswordService
}

// NewUserHandler creates a new UserHandler with the given store.
func NewUserHandler(s *store.Store, authService *middleware.SessionAuthService) *UserHandler {
	return &UserHandler{
		store:       s,
		authService: authService,
	}
}

// ManagedUserUpdateRequest represents the editable user fields from the CRUD form.
type ManagedUserUpdateRequest struct {
	Email           string `json:"email" form:"email" validate:"required,email"`
	Name            string `json:"name" form:"name" validate:"required,min=2,max=100"`
	Password        string `json:"password,omitempty" form:"password" validate:"omitempty,password"`
	ConfirmPassword string `json:"confirm_password,omitempty" form:"confirm_password"`
	Bio             string `json:"bio,omitempty" form:"bio" validate:"max=500"`
	AvatarURL       string `json:"avatar_url,omitempty" form:"avatar_url" validate:"omitempty,url"`
}

// Validate implements custom validation for ManagedUserUpdateRequest.
func (r ManagedUserUpdateRequest) Validate() error {
	if r.Password == "" && r.ConfirmPassword == "" {
		return nil
	}

	if r.Password == "" || r.ConfirmPassword == "" {
		return middleware.ValidationErrors{
			{Field: "confirm_password", Message: "password and confirmation are both required to change the password"},
		}
	}

	if r.Password != r.ConfirmPassword {
		return middleware.ValidationErrors{
			{Field: "confirm_password", Message: "passwords do not match"},
		}
	}

	return nil
}

func (h *UserHandler) listUsers(ctx context.Context) ([]store.User, error) {
	return h.store.ListUsers(ctx)
}

func (h *UserHandler) countUsers(ctx context.Context) (int64, error) {
	return h.store.CountUsers(ctx)
}

func (h *UserHandler) createManagedUser(c echo.Context) (store.User, error) {
	ctx := c.Request().Context()

	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return store.User{}, validationError(c, err)
	}

	if validationErrors := middleware.ValidateStruct(req); len(validationErrors) > 0 {
		return store.User{}, validationErrorWithDetails(c, validationErrors)
	}

	if err := req.Validate(); err != nil {
		return store.User{}, validationErrorWithDetails(c, err)
	}

	hashedPassword, err := h.authService.HashPasswordArgon2(req.Password)
	if err != nil {
		return store.User{}, internalError(c, "Failed to process password", err)
	}

	user, err := h.store.CreateUser(ctx, store.CreateUserParams{
		Email:        req.Email,
		Name:         req.Name,
		Bio:          stringPtr(req.Bio),
		AvatarUrl:    stringPtr(req.AvatarURL),
		PasswordHash: hashedPassword,
	})
	if err != nil {
		slog.Error("Failed to create user",
			"email", req.Email,
			"name", req.Name,
			"error", err,
			"request_id", c.Response().Header().Get(echo.HeaderXRequestID))
		return store.User{}, databaseWriteError(c, err, "Failed to create user")
	}

	slog.Info("User created successfully",
		"user_id", user.ID,
		"name", user.Name,
		"email", user.Email,
		"request_id", c.Response().Header().Get(echo.HeaderXRequestID))

	return user, nil
}

func (h *UserHandler) updateManagedUser(c echo.Context, id int64) (store.User, error) {
	ctx := c.Request().Context()

	var req ManagedUserUpdateRequest
	if err := c.Bind(&req); err != nil {
		return store.User{}, validationError(c, err)
	}

	if validationErrors := middleware.ValidateStruct(req); len(validationErrors) > 0 {
		return store.User{}, validationErrorWithDetails(c, validationErrors)
	}

	if err := req.Validate(); err != nil {
		return store.User{}, validationErrorWithDetails(c, err)
	}

	if req.Password != "" {
		hashedPassword, err := h.authService.HashPasswordArgon2(req.Password)
		if err != nil {
			return store.User{}, internalError(c, "Failed to process password", err)
		}

		user, err := h.store.UpdateUserPassword(ctx, store.UpdateUserPasswordParams{
			Email:        req.Email,
			Name:         req.Name,
			Bio:          stringPtr(req.Bio),
			AvatarUrl:    stringPtr(req.AvatarURL),
			PasswordHash: hashedPassword,
			ID:           id,
		})
		if err != nil {
			slog.Error("Failed to update user with password",
				"id", id,
				"email", req.Email,
				"error", err,
				"request_id", c.Response().Header().Get(echo.HeaderXRequestID))
			return store.User{}, err
		}

		slog.Info("User updated successfully",
			"id", id,
			"name", req.Name,
			"email", req.Email,
			"request_id", c.Response().Header().Get(echo.HeaderXRequestID))

		return user, nil
	}

	user, err := h.store.UpdateUser(ctx, store.UpdateUserParams{
		Email:     req.Email,
		Name:      req.Name,
		Bio:       stringPtr(req.Bio),
		AvatarUrl: stringPtr(req.AvatarURL),
		ID:        id,
	})
	if err != nil {
		slog.Error("Failed to update user",
			"id", id,
			"email", req.Email,
			"error", err,
			"request_id", c.Response().Header().Get(echo.HeaderXRequestID))
		return store.User{}, err
	}

	slog.Info("User updated successfully",
		"id", id,
		"name", req.Name,
		"email", req.Email,
		"request_id", c.Response().Header().Get(echo.HeaderXRequestID))

	return user, nil
}

func (h *UserHandler) legacyUserListState(ctx context.Context) ([]store.User, int64, error) {
	users, err := h.listUsers(ctx)
	if err != nil {
		return nil, 0, err
	}

	return users, int64(len(users)), nil
}

func (h *UserHandler) renderLegacyUserList(c echo.Context, ctx context.Context) error {
	users, count, err := h.legacyUserListState(ctx)
	if err != nil {
		return logAndReturnError(c, "fetch updated users", err, http.StatusInternalServerError, "Failed to fetch updated users")
	}

	return view.UserListSwap(users, count).Render(ctx, c.Response().Writer)
}

// Users renders the main user management page.
func (h *UserHandler) Users(c echo.Context) error {
	ctx := c.Request().Context()
	token := setupCSRFHeaders(c)

	users, count, err := h.legacyUserListState(ctx)
	if err != nil {
		return logAndReturnError(c, "fetch users", err, http.StatusInternalServerError, "Failed to fetch users")
	}

	return renderWithCSRF(c,
		view.UsersContent(users, count),         // HTMX component
		view.UsersWithCSRF(users, count, token), // Full page component with CSRF
		view.Users(users, count),                // Basic component
	)
}

// UserForm renders the user creation/edit form.
func (h *UserHandler) UserForm(c echo.Context) error {
	token := setupCSRFHeaders(c)
	return view.UserForm(nil, token).Render(c.Request().Context(), c.Response().Writer)
}

// EditUserForm renders the user edit form with existing data.
func (h *UserHandler) EditUserForm(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parseIDParam(c)
	if err != nil {
		return err
	}

	user, err := h.store.GetUser(ctx, id)
	if err != nil {
		return logAndReturnError(c, "fetch user", err, http.StatusNotFound, "User not found")
	}

	token := setupCSRFHeaders(c)
	return view.UserForm(&user, token).Render(ctx, c.Response().Writer)
}

// CreateUser creates a new user for the legacy HTMX screen.
func (h *UserHandler) CreateUser(c echo.Context) error {
	ctx := c.Request().Context()

	if _, err := h.createManagedUser(c); err != nil {
		return err
	}

	c.Response().Header().Set(HtmxTrigger, "userCreated")

	return h.renderLegacyUserList(c, ctx)
}

// UpdateUser updates an existing user for the legacy HTMX screen.
func (h *UserHandler) UpdateUser(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parseIDParam(c)
	if err != nil {
		return err
	}

	if _, err := h.updateManagedUser(c, id); err != nil {
		return databaseWriteError(c, err, "Failed to update user")
	}

	c.Response().Header().Set(HtmxTrigger, "userUpdated")

	return h.renderLegacyUserList(c, ctx)
}

// DeactivateUser deactivates a user instead of deleting for the legacy HTMX screen.
func (h *UserHandler) DeactivateUser(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parseIDParam(c)
	if err != nil {
		return err
	}

	if err := h.store.DeactivateUser(ctx, id); err != nil {
		return logAndReturnError(c, "deactivate user", err, http.StatusInternalServerError, "Failed to deactivate user")
	}

	slog.Info("User deactivated successfully",
		"id", id,
		"request_id", c.Response().Header().Get(echo.HeaderXRequestID))

	c.Response().Header().Set(HtmxTrigger, "userDeactivated")

	return h.renderLegacyUserList(c, ctx)
}

// DeleteUser permanently deletes a user for the legacy HTMX screen.
func (h *UserHandler) DeleteUser(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parseIDParam(c)
	if err != nil {
		return err
	}

	if err := h.store.DeleteUser(ctx, id); err != nil {
		return logAndReturnError(c, "delete user", err, http.StatusInternalServerError, "Failed to delete user")
	}

	slog.Info("User deleted successfully",
		"id", id,
		"request_id", c.Response().Header().Get(echo.HeaderXRequestID))

	c.Response().Header().Set(HtmxTrigger, "userDeleted")

	return h.renderLegacyUserList(c, ctx)
}

// ListUsersAPI returns the active user list as JSON.
func (h *UserHandler) ListUsersAPI(c echo.Context) error {
	users, err := h.listUsers(c.Request().Context())
	if err != nil {
		return logAndReturnError(c, "fetch users", err, http.StatusInternalServerError, "Failed to fetch users")
	}

	return writeJSON(c, http.StatusOK, apiUserListResponse{
		Users: apiUsersFromStore(users),
		Count: len(users),
	})
}

// GetUserAPI returns a single user record as JSON for edit flows.
func (h *UserHandler) GetUserAPI(c echo.Context) error {
	id, err := parseIDParam(c)
	if err != nil {
		return err
	}

	user, err := h.store.GetUser(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return notFoundError(c, "User not found")
		}

		return logAndReturnError(c, "fetch user", err, http.StatusInternalServerError, "Failed to fetch user")
	}

	return writeJSON(c, http.StatusOK, apiUserResponse{
		User: apiUserFromStore(user),
	})
}

// CreateUserAPI creates a user and returns the created record as JSON.
func (h *UserHandler) CreateUserAPI(c echo.Context) error {
	user, err := h.createManagedUser(c)
	if err != nil {
		return err
	}

	return writeJSON(c, http.StatusCreated, apiUserMutationResponse{
		Message: MsgUserCreateSuccess,
		User:    apiUserFromStore(user),
	})
}

// UpdateUserAPI updates a user and returns the updated record as JSON.
func (h *UserHandler) UpdateUserAPI(c echo.Context) error {
	id, err := parseIDParam(c)
	if err != nil {
		return err
	}

	user, err := h.updateManagedUser(c, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return notFoundError(c, "User not found")
		}

		return databaseWriteError(c, err, "Failed to update user")
	}

	return writeJSON(c, http.StatusOK, apiUserMutationResponse{
		Message: MsgUserUpdateSuccess,
		User:    apiUserFromStore(user),
	})
}

// DeactivateUserAPI deactivates a user and returns the updated record as JSON.
func (h *UserHandler) DeactivateUserAPI(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parseIDParam(c)
	if err != nil {
		return err
	}

	updated, err := h.store.DeactivateUserChecked(ctx, id)
	if err != nil {
		return logAndReturnError(c, "deactivate user", err, http.StatusInternalServerError, "Failed to deactivate user")
	}
	if !updated {
		return notFoundError(c, "User not found")
	}

	user, err := h.store.GetUser(ctx, id)
	if err != nil {
		return logAndReturnError(c, "fetch deactivated user", err, http.StatusInternalServerError, "Failed to fetch user")
	}

	slog.Info("User deactivated successfully",
		"id", id,
		"request_id", c.Response().Header().Get(echo.HeaderXRequestID))

	return writeJSON(c, http.StatusOK, apiUserMutationResponse{
		Message: MsgUserDeactivateSuccess,
		User:    apiUserFromStore(user),
	})
}

// DeleteUserAPI deletes a user and returns a JSON acknowledgement.
func (h *UserHandler) DeleteUserAPI(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := parseIDParam(c)
	if err != nil {
		return err
	}

	deleted, err := h.store.DeleteUserChecked(ctx, id)
	if err != nil {
		return logAndReturnError(c, "delete user", err, http.StatusInternalServerError, "Failed to delete user")
	}
	if !deleted {
		return notFoundError(c, "User not found")
	}

	slog.Info("User deleted successfully",
		"id", id,
		"request_id", c.Response().Header().Get(echo.HeaderXRequestID))

	return writeJSON(c, http.StatusOK, apiDeleteUserResponse{
		ID:      id,
		Deleted: true,
		Message: MsgUserDeleteSuccess,
	})
}

// UserCountAPI returns the count of active users as JSON.
func (h *UserHandler) UserCountAPI(c echo.Context) error {
	count, err := h.countUsers(c.Request().Context())
	if err != nil {
		return logAndReturnError(c, "count users", err, http.StatusInternalServerError, "Failed to count users")
	}

	return writeJSON(c, http.StatusOK, apiUserCountResponse{
		Count: count,
	})
}
