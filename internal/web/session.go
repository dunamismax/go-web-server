package web

import (
	"context"
	"net/http"
	"net/url"

	"github.com/dunamismax/go-web-server/internal/store"
)

// Session keys.
const (
	// sessionUserIDKey holds the signed-in user's ID.
	sessionUserIDKey = "user_id"
	// sessionAuthEpochKey holds the user's password_changed_at (in
	// microseconds) as of sign-in. A session whose value no longer matches
	// the database is stale: the password changed somewhere else.
	sessionAuthEpochKey = "auth_epoch"

	flashNoticeKey = "flash_notice"
	flashAlertKey  = "flash_alert"
)

// loadUser resolves the session to a user on every request and stores it in
// the context. Reading the row each time (one primary-key lookup) means a
// deactivated or deleted account, or a changed password, takes effect on the
// very next request instead of when the session expires.
func (s *Server) loadUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID := s.sessions.GetInt64(ctx, sessionUserIDKey)
		if userID == 0 {
			next.ServeHTTP(w, r)
			return
		}

		user, err := s.store.GetUser(ctx, userID)
		switch {
		case store.IsNotFound(err):
			// Account deleted since sign-in.
		case err != nil:
			s.handleError(w, r, err)
			return
		case user.IsActive && user.PasswordChangedAt.UnixMicro() == s.sessions.GetInt64(ctx, sessionAuthEpochKey):
			ctx = context.WithValue(ctx, userKey, &user)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// The session no longer maps to a usable account. End it.
		if err := s.sessions.Destroy(ctx); err != nil {
			s.handleError(w, r, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// signIn binds the session to user. The token is renewed first so a session
// ID planted before authentication cannot be reused afterwards.
func (s *Server) signIn(ctx context.Context, user store.User) error {
	if err := s.sessions.RenewToken(ctx); err != nil {
		return err
	}
	s.sessions.Put(ctx, sessionUserIDKey, user.ID)
	s.sessions.Put(ctx, sessionAuthEpochKey, user.PasswordChangedAt.UnixMicro())
	return nil
}

func (s *Server) flashNotice(ctx context.Context, message string) {
	s.sessions.Put(ctx, flashNoticeKey, message)
}

func (s *Server) flashAlert(ctx context.Context, message string) {
	s.sessions.Put(ctx, flashAlertKey, message)
}

// requireUser allows only signed-in users. Everyone else is sent to the
// sign-in page, and brought back afterwards when the request was a plain
// page view.
func (s *Server) requireUser(next handler) handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if currentUser(r.Context()) != nil {
			return next(w, r)
		}

		target := "/login"
		if r.Method == http.MethodGet && !isHTMX(r) {
			target += "?next=" + url.QueryEscape(r.URL.RequestURI())
		}
		return redirect(w, r, target)
	}
}

// requireAdmin allows only signed-in administrators.
func (s *Server) requireAdmin(next handler) handler {
	return s.requireUser(func(w http.ResponseWriter, r *http.Request) error {
		if !currentUser(r.Context()).IsAdmin() {
			return errForbidden("Only administrators can manage users.")
		}
		return next(w, r)
	})
}

// guestOnly keeps signed-in users away from the sign-in and sign-up pages.
func (s *Server) guestOnly(next handler) handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if currentUser(r.Context()) != nil {
			return redirect(w, r, "/")
		}
		return next(w, r)
	}
}

// rateLimited throttles a handler per client address.
func (s *Server) rateLimited(next handler) handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if !s.authLimiter.allow(s.clientIP.resolve(r)) {
			return errTooManyRequests()
		}
		return next(w, r)
	}
}
