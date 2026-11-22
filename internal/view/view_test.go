package view

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/dunamismax/go-web-server/internal/store"
)

func TestUserListPaging(t *testing.T) {
	t.Parallel()

	users := make([]store.User, 20)
	tests := []struct {
		name         string
		list         UserList
		wantSummary  string
		wantLastPage int
	}{
		{"empty", UserList{PageSize: 20, Page: 1}, "No users yet.", 1},
		{"empty search", UserList{PageSize: 20, Page: 1, Search: "zed"}, `No users match "zed".`, 1},
		{"one", UserList{PageSize: 20, Page: 1, Total: 1, Users: users[:1]}, "1 user", 1},
		{"exactly one page", UserList{PageSize: 20, Page: 1, Total: 20, Users: users}, "1–20 of 20 users", 1},
		{"first of three", UserList{PageSize: 20, Page: 1, Total: 41, Users: users}, "1–20 of 41 users", 3},
		{"last of three", UserList{PageSize: 20, Page: 3, Total: 41, Users: users[:1]}, "41–41 of 41 users", 3},
	}
	for _, tt := range tests {
		if got := tt.list.summary(); got != tt.wantSummary {
			t.Errorf("%s: summary = %q; want %q", tt.name, got, tt.wantSummary)
		}
		if got := tt.list.lastPage(); got != tt.wantLastPage {
			t.Errorf("%s: lastPage = %d; want %d", tt.name, got, tt.wantLastPage)
		}
	}
}

func TestUserListPageURL(t *testing.T) {
	t.Parallel()

	plain := UserList{}
	if got := plain.pageURL(1); got != "/users" {
		t.Errorf("pageURL(1) = %q; want /users", got)
	}
	if got := plain.pageURL(3); got != "/users?page=3" {
		t.Errorf("pageURL(3) = %q; want /users?page=3", got)
	}

	searching := UserList{Search: "ada & co"}
	if got := searching.pageURL(2); got != "/users?page=2&q=ada+%26+co" {
		t.Errorf("pageURL with search = %q; want the term query-escaped", got)
	}
}

func TestIsCurrent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path, href string
		want       bool
	}{
		{"/", "/", true},
		{"/users", "/", false},
		{"/users", "/users", true},
		{"/users/7/edit", "/users", true},
		{"/usersettings", "/users", false},
		{"/account", "/users", false},
	}
	for _, tt := range tests {
		if got := isCurrent(tt.path, tt.href); got != tt.want {
			t.Errorf("isCurrent(%q, %q) = %v; want %v", tt.path, tt.href, got, tt.want)
		}
	}
}

func TestTemplatesEscapeUserContent(t *testing.T) {
	t.Parallel()

	hostile := `<script>alert("x")</script>`
	admin := &store.User{ID: 1, Name: hostile, Email: "a@example.com", Role: store.RoleAdmin, IsActive: true}
	list := UserList{
		Users: []store.User{
			{ID: 2, Name: hostile, Email: "b@example.com", Role: store.RoleMember, IsActive: true, CreatedAt: time.Now()},
		},
		Search: hostile, Page: 1, PageSize: 20, Total: 1, SelfID: 1, Notice: hostile,
	}

	var buf bytes.Buffer
	page := Page{Title: hostile, Path: "/users", User: admin, Notice: hostile, Alert: hostile}
	if err := Users(page, list).Render(t.Context(), &buf); err != nil {
		t.Fatalf("render Users: %v", err)
	}
	form := UserForm{ID: 2, Name: hostile, Email: hostile, Role: store.RoleMember, Errors: Errors{"name": hostile, "": hostile}}
	if err := UserEdit(page, form).Render(t.Context(), &buf); err != nil {
		t.Fatalf("render UserEdit: %v", err)
	}

	html := buf.String()
	if strings.Contains(html, "<script>alert") {
		t.Error("user-supplied text reached the page unescaped")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("expected the hostile text to appear HTML-escaped")
	}
}

func TestFieldAccessibilityWiring(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	form := RegisterForm{Name: "Ada", Errors: Errors{"email": "Enter an email address."}}
	if err := Register(Page{Path: "/register"}, form).Render(t.Context(), &buf); err != nil {
		t.Fatalf("render Register: %v", err)
	}
	html := buf.String()

	for _, want := range []string{
		`<label for="email"`,
		`aria-invalid="true"`,
		`aria-describedby="email-error"`,
		`<p id="email-error"`,
		`aria-describedby="password-hint"`, // valid fields point at their hint
		`autocomplete="new-password"`,
		`minlength="12"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("register form is missing %s", want)
		}
	}
}

func TestErrorsFocus(t *testing.T) {
	t.Parallel()

	order := []string{"name", "email", "password"}
	tests := []struct {
		name           string
		errs           Errors
		focusWhenClean bool
		want           string // the one field that takes focus, or "" for none
	}{
		{"clean form focuses the first field", Errors{}, true, "name"},
		{"clean form without autofocus", Errors{}, false, ""},
		{"first invalid field wins", Errors{"password": "x", "email": "x"}, true, "email"},
		{"invalid field wins even without autofocus", Errors{"password": "x"}, false, "password"},
		{"form-level error focuses the first field", Errors{"": "x"}, false, "name"},
	}
	for _, tt := range tests {
		var got []string
		for _, field := range order {
			if tt.errs.focus(field, tt.focusWhenClean, order...) {
				got = append(got, field)
			}
		}
		switch {
		case tt.want == "" && len(got) != 0:
			t.Errorf("%s: focused %v; want none", tt.name, got)
		case tt.want != "" && (len(got) != 1 || got[0] != tt.want):
			t.Errorf("%s: focused %v; want exactly [%s]", tt.name, got, tt.want)
		}
	}
}

func TestFailedSubmitFocusesTheFieldToFix(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	form := RegisterForm{Name: "Ada", Email: "nope", Errors: Errors{"email": "Enter a valid email address."}}
	if err := Register(Page{Path: "/register"}, form).Render(t.Context(), &buf); err != nil {
		t.Fatalf("render Register: %v", err)
	}
	html := buf.String()

	if got := strings.Count(html, " autofocus"); got != 1 {
		t.Fatalf("page has %d autofocus attributes; want exactly 1", got)
	}
	_, afterEmail, found := strings.Cut(html, `<input id="email"`)
	if !found {
		t.Fatal("email input not found")
	}
	emailAttrs, _, _ := strings.Cut(afterEmail, ">")
	if !strings.Contains(emailAttrs, " autofocus") {
		t.Errorf("the invalid email field does not take focus: %s", emailAttrs)
	}
}
