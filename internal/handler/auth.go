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

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	store       authDataStore
	authService authSessionService
}

type authDataStore interface {
	GetUserByEmail(ctx context.Context, email string) (store.User, error)
	CreateUser(ctx context.Context, params store.CreateUserParams) (store.User, error)
}

type authSessionService interface {
	GetCurrentUser(c echo.Context) (*middleware.User, bool)
	VerifyPasswordArgon2(password, encoded string) (bool, error)
	HashPasswordArgon2(password string) (string, error)
	LoginUser(c echo.Context, user middleware.User) error
	LogoutUser(c echo.Context) error
	RequireAuth() echo.MiddlewareFunc
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(s *store.Store, authService *middleware.SessionAuthService) *AuthHandler {
	return &AuthHandler{
		store:       s,
		authService: authService,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" form:"email" validate:"required,email"`
	Password string `json:"password" form:"password" validate:"required,min=1"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email           string `json:"email" form:"email" validate:"required,email"`
	Name            string `json:"name" form:"name" validate:"required,min=2,max=100"`
	Password        string `json:"password" form:"password" validate:"required,password"`
	ConfirmPassword string `json:"confirm_password" form:"confirm_password" validate:"required"`
	Bio             string `json:"bio,omitempty" form:"bio" validate:"max=500"`
	AvatarURL       string `json:"avatar_url,omitempty" form:"avatar_url" validate:"omitempty,url"`
}

// Validate implements custom validation for RegisterRequest
func (r RegisterRequest) Validate() error {
	if r.Password != r.ConfirmPassword {
		return middleware.ValidationErrors{
			{Field: "confirm_password", Message: "passwords do not match"},
		}
	}
	return nil
}

func (h *AuthHandler) currentSessionUser(c echo.Context) (*middleware.User, bool, error) {
	user, exists := h.authService.GetCurrentUser(c)
	if !exists {
		return nil, false, nil
	}

	if !user.IsActive {
		if err := h.authService.LogoutUser(c); err != nil {
			return nil, false, middleware.NewAppError(
				middleware.ErrorTypeInternal,
				http.StatusInternalServerError,
				"Failed to clear inactive user session",
			).WithContext(c).WithInternal(err)
		}

		return nil, false, nil
	}

	return user, true, nil
}

func (h *AuthHandler) authenticate(c echo.Context) (*middleware.User, error) {
	ctx := c.Request().Context()

	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return nil, validationError(c, err)
	}

	if validationErrors := middleware.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, validationErrorWithDetails(c, validationErrors)
	}

	user, err := h.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			slog.Error("Failed to load user for login",
				"email", req.Email,
				"error", err,
				"request_id", c.Response().Header().Get(echo.HeaderXRequestID))
			return nil, internalError(c, "Authentication error", err)
		}

		slog.Warn("Login attempt with invalid email",
			"email", req.Email,
			"error", err,
			"request_id", c.Response().Header().Get(echo.HeaderXRequestID))

		return nil, authenticationError(c, "Invalid email or password")
	}

	if user.PasswordHash == "" {
		slog.Warn("Login attempt for account without password hash",
			"email", req.Email,
			"request_id", c.Response().Header().Get(echo.HeaderXRequestID))
		return nil, authenticationError(c, "Invalid email or password")
	}

	valid, err := h.authService.VerifyPasswordArgon2(req.Password, user.PasswordHash)
	if err != nil {
		slog.Warn("Password verification failed due to invalid hash",
			"email", req.Email,
			"error", err,
			"request_id", c.Response().Header().Get(echo.HeaderXRequestID))
		return nil, authenticationError(c, "Invalid email or password")
	}
	if !valid {
		return nil, authenticationError(c, "Invalid email or password")
	}

	if user.IsActive == nil || !*user.IsActive {
		return nil, authenticationError(c, "Account is inactive")
	}

	authUser := middleware.User{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.Name,
		IsActive: *user.IsActive,
	}

	if err := h.authService.LoginUser(c, authUser); err != nil {
		slog.Error("Failed to create user session",
			"user_id", user.ID,
			"error", err,
			"request_id", c.Response().Header().Get(echo.HeaderXRequestID))

		return nil, middleware.NewAppError(
			middleware.ErrorTypeInternal,
			http.StatusInternalServerError,
			"Failed to create user session",
		).WithContext(c).WithInternal(err)
	}

	slog.Info("User logged in successfully",
		"user_id", user.ID,
		"email", user.Email,
		"request_id", c.Response().Header().Get(echo.HeaderXRequestID))

	return &authUser, nil
}

func (h *AuthHandler) registerAndLogin(c echo.Context) (*middleware.User, error) {
	ctx := c.Request().Context()

	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return nil, validationError(c, err)
	}

	if validationErrors := middleware.ValidateStruct(req); len(validationErrors) > 0 {
		return nil, validationErrorWithDetails(c, validationErrors)
	}

	if err := req.Validate(); err != nil {
		return nil, validationErrorWithDetails(c, err)
	}

	hashedPassword, err := h.authService.HashPasswordArgon2(req.Password)
	if err != nil {
		slog.Error("Failed to hash password",
			"error", err,
			"request_id", c.Response().Header().Get(echo.HeaderXRequestID))

		return nil, middleware.NewAppError(
			middleware.ErrorTypeInternal,
			http.StatusInternalServerError,
			"Failed to process password",
		).WithContext(c).WithInternal(err)
	}

	var bioPtr *string
	if req.Bio != "" {
		bioPtr = &req.Bio
	}

	var avatarURLPtr *string
	if req.AvatarURL != "" {
		avatarURLPtr = &req.AvatarURL
	}

	user, err := h.store.CreateUser(ctx, store.CreateUserParams{
		Email:        req.Email,
		Name:         req.Name,
		Bio:          bioPtr,
		AvatarUrl:    avatarURLPtr,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		slog.Error("Failed to create user",
			"email", req.Email,
			"name", req.Name,
			"error", err,
			"request_id", c.Response().Header().Get(echo.HeaderXRequestID))

		return nil, databaseWriteError(c, err, "Failed to create user account")
	}

	authUser := middleware.User{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.Name,
		IsActive: user.IsActive != nil && *user.IsActive,
	}

	if err := h.authService.LoginUser(c, authUser); err != nil {
		slog.Error("Failed to create user session after registration",
			"user_id", user.ID,
			"error", err,
			"request_id", c.Response().Header().Get(echo.HeaderXRequestID))

		return nil, middleware.NewAppError(
			middleware.ErrorTypeInternal,
			http.StatusInternalServerError,
			"Failed to create user session",
		).WithContext(c).WithInternal(err)
	}

	slog.Info("User registered and logged in successfully",
		"user_id", user.ID,
		"email", user.Email,
		"request_id", c.Response().Header().Get(echo.HeaderXRequestID))

	return &authUser, nil
}

// LoginPage renders the login page
func (h *AuthHandler) LoginPage(c echo.Context) error {
	// Check if user is already authenticated
	if user, exists, err := h.currentSessionUser(c); err != nil {
		return err
	} else if exists && user.IsActive {
		return c.Redirect(http.StatusFound, RouteHome)
	}

	token := middleware.GetCSRFToken(c)

	return renderWithCSRF(c,
		view.LoginContent(),       // HTMX component
		view.LoginWithCSRF(token), // Full page component with CSRF
		view.Login(),              // Basic component
	)
}

// RegisterPage renders the registration page
func (h *AuthHandler) RegisterPage(c echo.Context) error {
	// Check if user is already authenticated
	if user, exists, err := h.currentSessionUser(c); err != nil {
		return err
	} else if exists && user.IsActive {
		return c.Redirect(http.StatusFound, RouteHome)
	}

	token := middleware.GetCSRFToken(c)

	return renderWithCSRF(c,
		view.RegisterContent(),       // HTMX component
		view.RegisterWithCSRF(token), // Full page component with CSRF
		view.Register(),              // Basic component
	)
}

// Login handles user login
func (h *AuthHandler) Login(c echo.Context) error {
	if _, err := h.authenticate(c); err != nil {
		return err
	}

	return redirectOrHtmx(c, RouteHome, MsgLoginSuccess)
}

// Register handles user registration
func (h *AuthHandler) Register(c echo.Context) error {
	if _, err := h.registerAndLogin(c); err != nil {
		return err
	}

	return redirectOrHtmx(c, RouteHome, MsgRegisterSuccess)
}

// Logout handles user logout
func (h *AuthHandler) Logout(c echo.Context) error {
	// Log the logout
	if user, exists, err := h.currentSessionUser(c); err != nil {
		return err
	} else if exists {
		slog.Info("User logged out successfully",
			"user_id", user.ID,
			"email", user.Email,
			"request_id", c.Response().Header().Get(echo.HeaderXRequestID))
	}

	// Destroy user session
	err := h.authService.LogoutUser(c)
	if err != nil {
		slog.Error("Failed to destroy session",
			"error", err,
			"request_id", c.Response().Header().Get(echo.HeaderXRequestID))
	}

	// Return success response
	return redirectOrHtmx(c, RouteLogin, MsgLogoutSuccess)
}

// AuthState returns the current session state for frontend bootstrap.
func (h *AuthHandler) AuthState(c echo.Context) error {
	user, exists, err := h.currentSessionUser(c)
	if err != nil {
		return err
	}

	resp := apiAuthStateResponse{
		Authenticated: exists,
		User:          nil,
		CSRF:          currentCSRFContract(c),
	}
	if exists {
		resp.User = apiSessionUserFromMiddleware(*user)
	}

	return c.JSON(http.StatusOK, resp)
}

// LoginAPI authenticates a user and returns JSON instead of redirects.
func (h *AuthHandler) LoginAPI(c echo.Context) error {
	user, err := h.authenticate(c)
	if err != nil {
		return err
	}

	return writeJSON(c, http.StatusOK, apiAuthMutationResponse{
		Message: MsgLoginSuccess,
		User:    apiSessionUserFromMiddleware(*user),
	})
}

// RegisterAPI creates a user, starts a session, and returns JSON.
func (h *AuthHandler) RegisterAPI(c echo.Context) error {
	user, err := h.registerAndLogin(c)
	if err != nil {
		return err
	}

	return writeJSON(c, http.StatusCreated, apiAuthMutationResponse{
		Message: MsgRegisterSuccess,
		User:    apiSessionUserFromMiddleware(*user),
	})
}

// LogoutAPI destroys the current session and returns JSON.
func (h *AuthHandler) LogoutAPI(c echo.Context) error {
	if user, exists, err := h.currentSessionUser(c); err != nil {
		return err
	} else if exists {
		slog.Info("User logged out successfully",
			"user_id", user.ID,
			"email", user.Email,
			"request_id", c.Response().Header().Get(echo.HeaderXRequestID))
	}

	if err := h.authService.LogoutUser(c); err != nil {
		slog.Error("Failed to destroy session",
			"error", err,
			"request_id", c.Response().Header().Get(echo.HeaderXRequestID))
	}

	return writeJSON(c, http.StatusOK, apiAuthMutationResponse{
		Message: MsgLogoutSuccess,
	})
}

// Profile handles user profile page
func (h *AuthHandler) Profile(c echo.Context) error {
	user, exists, err := h.currentSessionUser(c)
	if err != nil {
		return err
	}
	if !exists {
		return c.Redirect(http.StatusFound, RouteLogin)
	}

	token := middleware.GetCSRFToken(c)

	return renderWithCSRF(c,
		view.ProfileContent(*user),         // HTMX component
		view.ProfileWithCSRF(*user, token), // Full page component with CSRF
		view.Profile(*user),                // Basic component
	)
}
