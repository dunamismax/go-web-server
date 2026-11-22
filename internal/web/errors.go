package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/dunamismax/go-web-server/internal/view"
)

// httpError is an error with a status code and a message that is safe to
// show the user. Handlers return one to end a request with an error page.
type httpError struct {
	status  int
	message string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("%d %s: %s", e.status, http.StatusText(e.status), e.message)
}

func errBadRequest(message string) error {
	return &httpError{status: http.StatusBadRequest, message: message}
}

func errForbidden(message string) error {
	return &httpError{status: http.StatusForbidden, message: message}
}

func errNotFound() error {
	return &httpError{
		status:  http.StatusNotFound,
		message: "The page you are looking for does not exist or has moved.",
	}
}

func errTooManyRequests() error {
	return &httpError{
		status:  http.StatusTooManyRequests,
		message: "Too many attempts. Wait a minute and try again.",
	}
}

// statusClientClosedRequest is the non-standard code (from nginx) that marks
// a request abandoned by the client. It only ever appears in logs.
const statusClientClosedRequest = 499

var errorHeadings = map[int]string{
	http.StatusBadRequest:            "That request did not look right",
	http.StatusForbidden:             "You do not have access to this",
	http.StatusNotFound:              "Page not found",
	http.StatusRequestEntityTooLarge: "That upload is too large",
	http.StatusTooManyRequests:       "Slow down",
	http.StatusInternalServerError:   "Something went wrong",
}

// handleError is the single place a failed request is turned into a
// response. Expected failures become their status page; anything else is
// logged and shown as a generic 500 that leaks no detail.
func (s *Server) handleError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	message := "An unexpected error occurred. Try again in a moment."

	if httpErr, ok := errors.AsType[*httpError](err); ok {
		status, message = httpErr.status, httpErr.message
	} else if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		status, message = http.StatusRequestEntityTooLarge, "The submitted form was larger than this app accepts."
	} else if errors.Is(err, context.Canceled) && r.Context().Err() != nil {
		// The client went away mid-request. Nobody will read a response, so
		// just record the de facto "client closed request" status for the
		// access log instead of an error.
		w.WriteHeader(statusClientClosedRequest)
		return
	}

	if status >= http.StatusInternalServerError {
		s.log.ErrorContext(r.Context(), "request failed",
			slog.Any("error", err),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("request_id", requestIDFrom(r.Context())))
	}

	heading, ok := errorHeadings[status]
	if !ok {
		heading = http.StatusText(status)
	}

	if status == http.StatusTooManyRequests {
		w.Header().Set("Retry-After", "60")
	}

	// Error pages never consume flash messages: those wait for the next page
	// that renders normally.
	page := view.Page{Title: heading, Path: r.URL.Path, User: currentUser(r.Context())}
	component := view.ErrorPage(page, status, heading, message, requestIDFrom(r.Context()))

	if renderErr := render(w, r, status, component); renderErr != nil {
		s.log.ErrorContext(r.Context(), "render error page", slog.Any("error", renderErr))
	}
}

func (s *Server) notFound(_ http.ResponseWriter, _ *http.Request) error {
	return errNotFound()
}
