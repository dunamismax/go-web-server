package handler

import (
	"io/fs"
	"net/http"

	"log/slog"

	"github.com/dunamismax/go-web-server/internal/middleware"
	"github.com/dunamismax/go-web-server/internal/store"
	"github.com/dunamismax/go-web-server/internal/ui"
	webdist "github.com/dunamismax/go-web-server/web"
	"github.com/labstack/echo/v4"
)

// Handlers holds all the application handlers.
type Handlers struct {
	Home *HomeHandler
	User *UserHandler
	Auth *AuthHandler
}

// NewHandlers creates a new handlers instance with the given store.
func NewHandlers(s *store.Store, authService *middleware.SessionAuthService) *Handlers {
	return &Handlers{
		Home: NewHomeHandler(s),
		User: NewUserHandler(s, authService),
		Auth: NewAuthHandler(s, authService),
	}
}

// RegisterRoutes sets up all application routes.
func RegisterRoutes(e *echo.Echo, handlers *Handlers) error {
	// Serve static files
	staticFS, err := fs.Sub(ui.StaticFiles, "static")
	if err != nil {
		slog.Error("failed to create static file system", "error", err)

		return err
	}

	e.GET("/static/*", echo.WrapHandler(http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))))

	frontendFS, err := webdist.DistFS()
	if err != nil {
		slog.Error("failed to create frontend dist file system", "error", err)

		return err
	}

	frontend := NewFrontendHandler(frontendFS)
	e.GET("/_astro/*", frontend.Asset)

	// Home routes
	e.GET(RouteHome, frontend.Page("index.html"))
	e.GET("/demo", handlers.Home.Demo)
	e.GET("/health", handlers.Home.Health)

	// Authentication routes (no auth required)
	auth := e.Group("/auth")
	auth.GET("/login", frontend.Page("auth/login/index.html"))
	auth.GET("/register", frontend.Page("auth/register/index.html"))
	auth.GET("/logout", frontend.Page("auth/logout/index.html"))
	auth.POST("/login", handlers.Auth.Login)
	auth.POST("/register", handlers.Auth.Register)
	auth.POST("/logout", handlers.Auth.Logout)

	requireAuth := handlers.Auth.authService.RequireAuth()

	// Protected routes (authentication required)
	e.GET(RouteProfile, frontend.Page("profile/index.html"), requireAuth)

	// User management routes
	users := e.Group("/users", requireAuth)
	users.GET("", frontend.Page("users/index.html"))

	// API routes
	apiAuth := e.Group("/api/auth")
	apiAuth.GET("/state", handlers.Auth.AuthState)
	apiAuth.POST("/login", handlers.Auth.LoginAPI)
	apiAuth.POST("/register", handlers.Auth.RegisterAPI)
	apiAuth.POST("/logout", handlers.Auth.LogoutAPI)

	apiUsers := e.Group("/api/users", requireAuth)
	apiUsers.GET("", handlers.User.ListUsersAPI)
	apiUsers.GET("/count", handlers.User.UserCountAPI)
	apiUsers.GET("/:id", handlers.User.GetUserAPI)
	apiUsers.POST("", handlers.User.CreateUserAPI)
	apiUsers.PUT("/:id", handlers.User.UpdateUserAPI)
	apiUsers.PATCH("/:id/deactivate", handlers.User.DeactivateUserAPI)
	apiUsers.DELETE("/:id", handlers.User.DeleteUserAPI)

	return nil
}
