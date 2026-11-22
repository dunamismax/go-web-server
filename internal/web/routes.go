package web

import (
	"net/http"

	"github.com/dunamismax/go-web-server/internal/assets"
)

// Handler returns the complete HTTP handler: every route wrapped in the
// middleware stack.
func (s *Server) Handler() http.Handler {
	// Pages. Everything here runs with the session loaded and, when someone
	// is signed in, their user record in the request context.
	pages := http.NewServeMux()
	pages.Handle("GET /{$}", s.handle(s.home))

	pages.Handle("GET /login", s.handle(s.guestOnly(s.loginPage)))
	pages.Handle("POST /login", s.handle(s.rateLimited(s.guestOnly(s.login))))
	pages.Handle("GET /register", s.handle(s.guestOnly(s.registerPage)))
	pages.Handle("POST /register", s.handle(s.rateLimited(s.guestOnly(s.register))))
	pages.Handle("POST /logout", s.handle(s.logout))

	pages.Handle("GET /account", s.handle(s.requireUser(s.accountPage)))
	pages.Handle("POST /account/profile", s.handle(s.requireUser(s.updateProfile)))
	pages.Handle("POST /account/password", s.handle(s.rateLimited(s.requireUser(s.changePassword))))

	pages.Handle("GET /users", s.handle(s.requireAdmin(s.listUsers)))
	pages.Handle("GET /users/new", s.handle(s.requireAdmin(s.newUserPage)))
	pages.Handle("POST /users", s.handle(s.requireAdmin(s.createUser)))
	pages.Handle("GET /users/{id}/edit", s.handle(s.requireAdmin(s.editUserPage)))
	pages.Handle("POST /users/{id}", s.handle(s.requireAdmin(s.updateUser)))
	pages.Handle("POST /users/{id}/active", s.handle(s.requireAdmin(s.setUserActive)))
	pages.Handle("POST /users/{id}/delete", s.handle(s.requireAdmin(s.deleteUser)))

	pages.Handle("/", s.handle(s.notFound))

	// Static files and health probes skip the session, so they never touch
	// the database on behalf of a cookie.
	root := http.NewServeMux()
	root.Handle("GET "+assets.URLPrefix, assets.Handler())
	root.Handle("GET /favicon.ico", assets.Favicon())
	root.HandleFunc("GET /healthz", s.healthz)
	root.HandleFunc("GET /readyz", s.readyz)
	root.Handle("/", s.sessions.LoadAndSave(s.loadUser(pages)))

	// The first middleware listed is the outermost.
	return chain(root,
		s.requestID,
		s.accessLog,
		s.recoverPanic,
		s.secureHeaders,
		s.crossOriginProtection(),
		limitBody,
	)
}

func chain(h http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}
