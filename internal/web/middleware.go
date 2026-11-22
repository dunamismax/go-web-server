package web

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// maxBodyBytes caps request bodies. The app only accepts small forms.
const maxBodyBytes = 1 << 20

// requestID tags each request with an ID that appears in the logs, in the
// X-Request-Id response header, and on 5xx error pages, so a user report can
// be matched to a log line.
func (s *Server) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := rand.Text()
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}

// accessLog writes one structured line per request.
func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r)

		level := slog.LevelInfo
		switch {
		case rec.status() >= 500:
			level = slog.LevelError
		case r.URL.Path == "/healthz" || r.URL.Path == "/readyz":
			// Probes arrive every few seconds; keep them out of normal logs.
			level = slog.LevelDebug
		}

		s.log.LogAttrs(r.Context(), level, "request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rec.status()),
			slog.Int64("bytes", rec.bytes),
			slog.Duration("duration", time.Since(start)),
			slog.String("ip", s.clientIP.resolve(r)),
			slog.String("request_id", requestIDFrom(r.Context())),
		)
	})
}

// recoverPanic turns a panic in a handler into a logged 500 instead of a
// dropped connection.
func (s *Server) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if v == http.ErrAbortHandler { //nolint:errorlint // sentinel panic value, compared by identity as net/http does
				panic(v)
			}

			s.log.ErrorContext(r.Context(), "panic recovered",
				slog.Any("panic", v),
				slog.String("stack", string(debug.Stack())),
				slog.String("request_id", requestIDFrom(r.Context())),
			)
			s.handleError(w, r, fmt.Errorf("panic: %v", v))
		}()

		next.ServeHTTP(w, r)
	})
}

// contentSecurityPolicy allows only same-origin resources. Nothing is
// inline: styles come from app.css and the only script is htmx.
const contentSecurityPolicy = "default-src 'self'; " +
	"base-uri 'none'; " +
	"form-action 'self'; " +
	"frame-ancestors 'none'; " +
	"object-src 'none'"

func (s *Server) secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		if s.cfg.IsProduction() {
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

// crossOriginProtection is the CSRF defense. The standard library's
// CrossOriginProtection rejects state-changing requests that a browser marks
// as coming from another origin (via Sec-Fetch-Site, or Origin on older
// browsers). No tokens, hidden fields or cookies are involved, so forms and
// htmx requests need nothing extra.
func (s *Server) crossOriginProtection() func(http.Handler) http.Handler {
	protection := http.NewCrossOriginProtection()
	for _, origin := range s.cfg.TrustedOrigins {
		if err := protection.AddTrustedOrigin(origin); err != nil {
			s.log.Warn("ignoring invalid TRUSTED_ORIGINS entry", "origin", origin, "error", err)
		}
	}
	protection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.handleError(w, r, errForbidden("This request came from another site and was blocked."))
	}))

	return protection.Handler
}

func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// statusRecorder captures the response status and size for the access log.
type statusRecorder struct {
	http.ResponseWriter

	code  int
	bytes int64
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.code == 0 {
		r.code = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.code == 0 {
		r.code = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += int64(n)
	return n, err
}

func (r *statusRecorder) status() int {
	if r.code == 0 {
		return http.StatusOK
	}
	return r.code
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
