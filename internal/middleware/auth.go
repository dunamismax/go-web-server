package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/argon2"
)

// User represents authenticated user information
type User struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

// Argon2 parameters for password hashing
type Argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultArgon2Params provides secure defaults for Argon2id
var DefaultArgon2Params = Argon2Params{
	Memory:      64 * 1024, // 64 MB
	Iterations:  3,         // 3 iterations
	Parallelism: 2,         // 2 threads
	SaltLength:  16,        // 16 bytes salt
	KeyLength:   32,        // 32 bytes key
}

// SessionAuthService provides session-based authentication
type SessionAuthService struct {
	sessionManager *scs.SessionManager
	argon2Params   Argon2Params
}

// NewSessionAuthService creates a new session-based auth service
func NewSessionAuthService(sessionManager *scs.SessionManager) *SessionAuthService {
	return &SessionAuthService{
		sessionManager: sessionManager,
		argon2Params:   DefaultArgon2Params,
	}
}

// HashPasswordArgon2 hashes a password using Argon2id
func (s *SessionAuthService) HashPasswordArgon2(password string) (string, error) {
	// Generate random salt
	salt, err := generateRandomBytes(s.argon2Params.SaltLength)
	if err != nil {
		return "", err
	}

	// Hash the password using Argon2id
	hash := argon2.IDKey([]byte(password), salt, s.argon2Params.Iterations, s.argon2Params.Memory, s.argon2Params.Parallelism, s.argon2Params.KeyLength)

	// Encode the result as base64
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Format: $argon2id$v=19$m=65536,t=3,p=2$salt$hash
	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", s.argon2Params.Memory, s.argon2Params.Iterations, s.argon2Params.Parallelism, b64Salt, b64Hash)
	return encoded, nil
}

// VerifyPasswordArgon2 verifies a password against an Argon2id hash
func (s *SessionAuthService) VerifyPasswordArgon2(password, encoded string) (bool, error) {
	// Parse the encoded hash
	params, salt, hash, err := decodeArgon2Hash(encoded)
	if err != nil {
		return false, err
	}

	// Hash the password with the same parameters
	otherHash := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)

	// Compare the hashes using constant time comparison
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}
	return false, nil
}

// LoginUser creates a session for an authenticated user
func (s *SessionAuthService) LoginUser(c echo.Context, user User) error {
	ctx := c.Request().Context()

	if err := s.sessionManager.RenewToken(ctx); err != nil {
		return err
	}

	// Store user information in session
	s.sessionManager.Put(ctx, "user_id", user.ID)
	s.sessionManager.Put(ctx, "user_email", user.Email)
	s.sessionManager.Put(ctx, "user_name", user.Name)
	s.sessionManager.Put(ctx, "user_is_active", user.IsActive)
	s.sessionManager.Put(ctx, "authenticated", true)

	return nil
}

// LogoutUser destroys the user session
func (s *SessionAuthService) LogoutUser(c echo.Context) error {
	ctx := c.Request().Context()
	return s.sessionManager.Destroy(ctx)
}

// GetCurrentUser retrieves the current authenticated user from session
func (s *SessionAuthService) GetCurrentUser(c echo.Context) (*User, bool) {
	ctx := c.Request().Context()

	authenticated := s.sessionManager.GetBool(ctx, "authenticated")
	if !authenticated {
		return nil, false
	}

	userID := s.sessionManager.GetInt64(ctx, "user_id")
	if userID == 0 {
		return nil, false
	}

	user := User{
		ID:       userID,
		Email:    s.sessionManager.GetString(ctx, "user_email"),
		Name:     s.sessionManager.GetString(ctx, "user_name"),
		IsActive: s.sessionManager.GetBool(ctx, "user_is_active"),
	}

	return &user, true
}

// SessionMiddleware wraps the SCS session middleware for Echo
func (s *SessionAuthService) SessionMiddleware() echo.MiddlewareFunc {
	return echo.WrapMiddleware(s.sessionManager.LoadAndSave)
}

// RequireAuth middleware that requires session-based authentication
func (s *SessionAuthService) RequireAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, exists := s.GetCurrentUser(c)
			if !exists {
				// Redirect to login page for browser requests
				if !expectsJSONResponse(c.Request()) {
					return c.Redirect(http.StatusFound, "/auth/login")
				}

				return NewAppError(
					ErrorTypeAuthentication,
					http.StatusUnauthorized,
					"Authentication required",
				).WithContext(c)
			}

			if !user.IsActive {
				if err := s.LogoutUser(c); err != nil {
					return NewAppError(
						ErrorTypeInternal,
						http.StatusInternalServerError,
						"Failed to clear inactive user session",
					).WithContext(c).WithInternal(err)
				}
				return NewAppError(
					ErrorTypeAuthentication,
					http.StatusUnauthorized,
					"User account is inactive",
				).WithContext(c)
			}

			// Store user in context for backwards compatibility
			c.Set("user", *user)
			c.Set("user_id", user.ID)

			return next(c)
		}
	}
}

func expectsJSONResponse(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get(echo.HeaderAccept))
	return strings.Contains(accept, echo.MIMEApplicationJSON) || strings.HasPrefix(r.URL.Path, "/api/")
}

// OptionalAuth middleware that loads user if authenticated but doesn't require it
func (s *SessionAuthService) OptionalAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, exists := s.GetCurrentUser(c)
			if exists && user.IsActive {
				// Store user in context for backwards compatibility
				c.Set("user", *user)
				c.Set("user_id", user.ID)
			}

			return next(c)
		}
	}
}

// Helper functions

// generateRandomBytes generates cryptographically secure random bytes
func generateRandomBytes(n uint32) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

const maxUint32Value = ^uint32(0)

func checkedUint32Len(value []byte) (uint32, error) {
	if uint64(len(value)) > uint64(maxUint32Value) {
		return 0, errors.New("decoded hash value exceeds uint32 length")
	}

	//nolint:gosec // The length is explicitly bounded above by maxUint32Value.
	return uint32(len(value)), nil
}

// decodeArgon2Hash parses an encoded Argon2 hash
func decodeArgon2Hash(encoded string) (params Argon2Params, salt, hash []byte, err error) {
	var version int
	_, err = fmt.Sscanf(encoded, "$argon2id$v=%d$m=%d,t=%d,p=%d$", &version, &params.Memory, &params.Iterations, &params.Parallelism)
	if err != nil {
		return params, nil, nil, errors.New("invalid hash format")
	}

	if version != argon2.Version {
		return params, nil, nil, errors.New("incompatible version of argon2")
	}

	// Extract salt and hash parts
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return params, nil, nil, errors.New("invalid hash format")
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return params, nil, nil, err
	}
	params.SaltLength, err = checkedUint32Len(salt)
	if err != nil {
		return params, nil, nil, err
	}

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return params, nil, nil, err
	}
	params.KeyLength, err = checkedUint32Len(hash)
	if err != nil {
		return params, nil, nil, err
	}

	return params, salt, hash, nil
}

// Backwards compatibility functions
func GetCurrentUser(c echo.Context) (*User, bool) {
	user := c.Get("user")
	if user == nil {
		return nil, false
	}

	if u, ok := user.(User); ok {
		return &u, true
	}

	return nil, false
}

func GetCurrentUserID(c echo.Context) (int64, bool) {
	userID := c.Get("user_id")
	if userID == nil {
		return 0, false
	}

	if id, ok := userID.(int64); ok {
		return id, true
	}

	return 0, false
}
