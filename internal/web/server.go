// Package web is the HTTP layer: routing, middleware, sessions and the
// handlers behind every page.
package web

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"

	"github.com/dunamismax/go-web-server/internal/auth"
	"github.com/dunamismax/go-web-server/internal/config"
	"github.com/dunamismax/go-web-server/internal/store"
)

// Options are the collaborators a Server needs.
type Options struct {
	Config config.Config
	Logger *slog.Logger
	Store  *store.Store
	// Hasher is optional. Tests pass one with cheap parameters; production
	// leaves it nil to get auth.DefaultParams.
	Hasher *auth.Hasher
}

// Server holds the dependencies shared by all handlers.
type Server struct {
	cfg    config.Config
	log    *slog.Logger
	store  *store.Store
	hasher *auth.Hasher

	sessions     *scs.SessionManager
	sessionStore *pgxstore.PostgresStore

	clientIP    ipResolver
	authLimiter *rateLimiter
}

// New wires up a Server. Call Close when it is no longer serving.
func New(opts Options) (*Server, error) {
	if opts.Logger == nil || opts.Store == nil {
		return nil, errors.New("web: Logger and Store are required")
	}

	hasher := opts.Hasher
	if hasher == nil {
		var err error
		if hasher, err = auth.NewHasher(auth.DefaultParams); err != nil {
			return nil, err
		}
	}

	s := &Server{
		cfg:      opts.Config,
		log:      opts.Logger,
		store:    opts.Store,
		hasher:   hasher,
		clientIP: ipResolver{trusted: opts.Config.TrustedProxies},
		// Ten sign-in or sign-up attempts per minute per client address.
		authLimiter: newRateLimiter(10, time.Minute),
	}
	s.sessionStore = pgxstore.New(opts.Store.Pool())
	s.sessions = s.newSessionManager()

	return s, nil
}

// Close stops the Server's background work. It does not close the Store.
func (s *Server) Close() {
	s.sessionStore.StopCleanup()
}

func (s *Server) newSessionManager() *scs.SessionManager {
	sm := scs.New()
	sm.Store = s.sessionStore
	sm.Lifetime = s.cfg.SessionLifetime
	sm.Cookie.Name = "session"
	sm.Cookie.HttpOnly = true
	sm.Cookie.Path = "/"
	sm.Cookie.Persist = true
	// Lax keeps the user signed in when they follow a link from another site,
	// while withholding the cookie from cross-site form posts.
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Secure = s.cfg.CookieSecure()
	if sm.Cookie.Secure {
		// The __Host- prefix makes browsers refuse the cookie unless it is
		// Secure, host-only and scoped to "/", which blocks subdomain and
		// plain-HTTP cookie injection.
		sm.Cookie.Name = "__Host-session"
	}
	sm.ErrorFunc = func(w http.ResponseWriter, r *http.Request, err error) {
		s.handleError(w, r, err)
	}
	return sm
}
