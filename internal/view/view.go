// Package view holds the templ components that render every page, together
// with the plain data types the handlers fill in to drive them.
//
// Files ending in _templ.go are generated from the .templ files by
// "make templ".
package view

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/dunamismax/go-web-server/internal/auth"
	"github.com/dunamismax/go-web-server/internal/store"
)

// AppName is the product name shown in the header, footer and page titles.
const AppName = "Go Web Server"

// htmxConfig locks htmx down so it works under the strict Content Security
// Policy the server sends: no inline styles, no eval, no script tags in
// swapped content, and no page snapshots kept in browser storage.
const htmxConfig = `{"includeIndicatorStyles":false,"allowEval":false,"allowScriptTags":false,"historyCacheSize":0,"refreshOnHistoryMiss":true}`

var passwordHint = fmt.Sprintf("At least %d characters. A few unrelated words work well.", auth.MinPasswordLength)

// Page carries what the shared layout needs on every request.
type Page struct {
	Title  string
	Path   string      // request path, used to highlight the current nav link
	User   *store.User // nil when signed out
	Notice string      // one-time success message
	Alert  string      // one-time error message
}

func (p Page) fullTitle() string {
	if p.Title == "" {
		return AppName
	}
	return p.Title + " · " + AppName
}

// Errors maps a form field name to its validation message. The empty key
// holds a message about the form as a whole.
type Errors map[string]string

// Add records msg for field unless the field already has a message.
func (e Errors) Add(field, msg string) {
	if _, exists := e[field]; !exists {
		e[field] = msg
	}
}

// Get returns the message for field, or "" when it is valid.
func (e Errors) Get(field string) string {
	return e[field]
}

// Any reports whether any field failed validation.
func (e Errors) Any() bool {
	return len(e) > 0
}

// focus reports whether field should take keyboard focus when its form
// renders. After a failed submit that is the first field (in order) with an
// error, so keyboard and screen-reader users land on what needs fixing. On a
// clean form it is the first field, and only when focusWhenClean is set.
func (e Errors) focus(field string, focusWhenClean bool, order ...string) bool {
	if len(order) == 0 {
		return false
	}
	if !e.Any() {
		return focusWhenClean && field == order[0]
	}
	for _, name := range order {
		if e.Get(name) != "" {
			return field == name
		}
	}
	// Only a form-level message, such as a failed sign-in.
	return field == order[0]
}

// LoginForm is the sign-in form's state.
type LoginForm struct {
	Email  string
	Next   string // local path to return to after signing in
	Errors Errors
}

// RegisterForm is the sign-up form's state.
type RegisterForm struct {
	Name   string
	Email  string
	Errors Errors
}

// ProfileForm is the account page's profile form.
type ProfileForm struct {
	Name   string
	Email  string
	Errors Errors
}

// PasswordForm is the account page's change-password form. Passwords are
// never echoed back, so it only carries errors.
type PasswordForm struct {
	Errors Errors
}

// UserForm is the admin form for creating or editing a user. ID is zero for
// a new user.
type UserForm struct {
	ID     int64
	Name   string
	Email  string
	Role   store.Role
	IsSelf bool // the admin is editing their own account
	Errors Errors
}

// UserList is one page of the admin user table.
type UserList struct {
	Users    []store.User
	Search   string
	Page     int
	PageSize int
	Total    int64
	SelfID   int64  // the signed-in admin, who cannot deactivate themselves
	Notice   string // result of an in-place action, shown above the table
	Alert    string
}

func (l UserList) lastPage() int {
	if l.Total == 0 || l.PageSize == 0 {
		return 1
	}
	return int((l.Total + int64(l.PageSize) - 1) / int64(l.PageSize))
}

func (l UserList) pageURL(page int) string {
	query := url.Values{}
	if l.Search != "" {
		query.Set("q", l.Search)
	}
	if page > 1 {
		query.Set("page", strconv.Itoa(page))
	}
	if len(query) == 0 {
		return "/users"
	}
	return "/users?" + query.Encode()
}

func (l UserList) summary() string {
	switch {
	case l.Total == 0 && l.Search != "":
		return fmt.Sprintf("No users match %q.", l.Search)
	case l.Total == 0:
		return "No users yet."
	case l.Total == 1:
		return "1 user"
	default:
		first := (l.Page-1)*l.PageSize + 1
		last := first + len(l.Users) - 1
		return fmt.Sprintf("%d–%d of %d users", first, last, l.Total)
	}
}

func userPath(id int64, suffix string) string {
	return "/users/" + strconv.FormatInt(id, 10) + suffix
}

func formatDate(t time.Time) string {
	return t.Format("Jan 2, 2006")
}

func roleLabel(r store.Role) string {
	if r == store.RoleAdmin {
		return "Admin"
	}
	return "Member"
}
