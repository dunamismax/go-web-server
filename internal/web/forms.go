package web

import (
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/dunamismax/go-web-server/internal/auth"
	"github.com/dunamismax/go-web-server/internal/store"
	"github.com/dunamismax/go-web-server/internal/view"
)

const (
	maxNameLength  = 100
	maxEmailLength = 254
)

// parseForm parses the request body, turning an oversized or malformed body
// into the matching error page.
func parseForm(r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
			return err
		}
		return errBadRequest("The submitted form could not be read.")
	}
	return nil
}

// normalizeEmail trims and lowercases an address. Emails are stored and
// compared in this form, so "Ada@Example.com" and "ada@example.com" are the
// same account.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateName(errs view.Errors, name string) {
	switch n := utf8.RuneCountInString(name); {
	case n == 0:
		errs.Add("name", "Enter a name.")
	case n > maxNameLength:
		errs.Add("name", fmt.Sprintf("Use %d characters or fewer.", maxNameLength))
	}
}

func validateEmail(errs view.Errors, email string) {
	if email == "" {
		errs.Add("email", "Enter an email address.")
		return
	}
	// ParseAddress also accepts forms like "Ada <ada@example.com>"; requiring
	// the parsed address to equal the input rejects those.
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || len(email) > maxEmailLength {
		errs.Add("email", "Enter a valid email address, like name@example.com.")
	}
}

func validateNewPassword(errs view.Errors, field, password string) {
	switch n := len(password); {
	case n < auth.MinPasswordLength:
		errs.Add(field, fmt.Sprintf("Use at least %d characters.", auth.MinPasswordLength))
	case n > auth.MaxPasswordLength:
		errs.Add(field, fmt.Sprintf("Use %d characters or fewer.", auth.MaxPasswordLength))
	}
}

func validateRole(errs view.Errors, role store.Role) {
	if !role.Valid() {
		errs.Add("role", "Choose a role.")
	}
}

const emailTakenMessage = "An account with this email already exists."
