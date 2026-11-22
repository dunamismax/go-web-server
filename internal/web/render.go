package web

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/a-h/templ"

	"github.com/dunamismax/go-web-server/internal/view"
)

// handler is an http.HandlerFunc that can fail. Returning the error keeps
// each handler linear and sends every failure through handleError.
type handler func(w http.ResponseWriter, r *http.Request) error

func (s *Server) handle(h handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			s.handleError(w, r, err)
		}
	})
}

var bufferPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

// render writes a component as an HTML response. It renders into a buffer
// first, so a failure halfway through a template produces a clean error page
// rather than half a document.
func render(w http.ResponseWriter, r *http.Request, status int, component templ.Component) error {
	buf, _ := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	if err := component.Render(r.Context(), buf); err != nil {
		return err
	}

	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	// Pages are per-user. Keep them out of shared caches and out of the
	// back/forward cache after sign-out.
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, err := buf.WriteTo(w)
	return err
}

// page builds the layout data for a normally rendered page and consumes any
// pending flash messages.
func (s *Server) page(r *http.Request, title string) view.Page {
	ctx := r.Context()
	return view.Page{
		Title:  title,
		Path:   r.URL.Path,
		User:   currentUser(ctx),
		Notice: s.sessions.PopString(ctx, flashNoticeKey),
		Alert:  s.sessions.PopString(ctx, flashAlertKey),
	}
}

// isHTMX reports whether htmx issued the request. History restores are
// excluded because they need the full page.
func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true" &&
		r.Header.Get("HX-History-Restore-Request") != "true"
}

// redirect sends the browser to target after a successful form post. htmx
// follows ordinary redirects invisibly and would swap the destination page
// into a fragment, so htmx requests get an HX-Redirect header instead, which
// makes it navigate the whole window.
//
// target must be a path the server chose. The one caller that starts from
// user input (the post-login "next") cleans it with safeNext first.
func redirect(w http.ResponseWriter, r *http.Request, target string) error {
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", target)
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
	http.Redirect(w, r, target, http.StatusSeeOther) //nolint:gosec // see the doc comment
	return nil
}

// safeNext returns next when it is a local path, and "/" otherwise. It keeps
// the post-login redirect from being abused to send users to another site.
func safeNext(next string) string {
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.ContainsAny(next, "\\\r\n") {
		return "/"
	}
	if u, err := url.Parse(next); err != nil || u.IsAbs() || u.Host != "" {
		return "/"
	}
	return next
}
