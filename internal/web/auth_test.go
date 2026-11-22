package web_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/dunamismax/go-web-server/internal/auth"
	"github.com/dunamismax/go-web-server/internal/store"
)

func TestRegister(t *testing.T) {
	t.Parallel()
	a := newApp(t)

	first := a.newBrowser()
	resp := first.register("  Ada Lovelace ", " Ada@Example.COM ")
	wantRedirect(t, resp, "/")

	// Registration signs the new user in, and the welcome flash shows once.
	home := first.get("/")
	wantContains(t, home, "Welcome, Ada Lovelace.", "you are the administrator", "Welcome back, Ada Lovelace.")
	wantNotContains(t, first.get("/"), "you are the administrator")

	ada := a.userByEmail("ada@example.com") // trimmed and lowercased
	if ada.Name != "Ada Lovelace" || ada.Role != store.RoleAdmin {
		t.Errorf("first user = %q (%s); want Ada Lovelace (admin)", ada.Name, ada.Role)
	}
	if strings.Contains(ada.PasswordHash, testPassword) || !strings.HasPrefix(ada.PasswordHash, "$argon2id$") {
		t.Errorf("password is not stored as an Argon2id hash: %q", ada.PasswordHash)
	}

	second := a.newBrowser()
	second.mustRegister("Grace Hopper", "grace@example.com")
	if role := a.userByEmail("grace@example.com").Role; role != store.RoleMember {
		t.Errorf("second user role = %q; want member", role)
	}

	// Signed-in users have no business on the sign-up page.
	wantRedirect(t, first.get("/register"), "/")
}

func TestRegisterValidation(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	b := a.newBrowser()
	b.mustRegister("Taken", "taken@example.com")

	guest := a.newBrowser()
	tests := []struct {
		name string
		form url.Values
		want string
	}{
		{"missing name", url.Values{"name": {" "}, "email": {"a@example.com"}, "password": {testPassword}}, "Enter a name."},
		{"long name", url.Values{"name": {strings.Repeat("n", 101)}, "email": {"a@example.com"}, "password": {testPassword}}, "Use 100 characters or fewer."},
		{"missing email", url.Values{"name": {"A"}, "email": {""}, "password": {testPassword}}, "Enter an email address."},
		{"bad email", url.Values{"name": {"A"}, "email": {"not-an-email"}, "password": {testPassword}}, "Enter a valid email address"},
		{"display-name email", url.Values{"name": {"A"}, "email": {"A <a@example.com>"}, "password": {testPassword}}, "Enter a valid email address"},
		{"short password", url.Values{"name": {"A"}, "email": {"a@example.com"}, "password": {"too short"}}, "Use at least 12 characters."},
		{"long password", url.Values{"name": {"A"}, "email": {"a@example.com"}, "password": {strings.Repeat("p", auth.MaxPasswordLength+1)}}, "Use 128 characters or fewer."},
		{"taken email", url.Values{"name": {"A"}, "email": {"TAKEN@example.com"}, "password": {testPassword}}, "An account with this email already exists."},
	}
	for _, tt := range tests {
		resp := guest.post("/register", tt.form)
		if resp.status != http.StatusUnprocessableEntity {
			t.Errorf("%s: status = %d; want 422", tt.name, resp.status)
		}
		if !strings.Contains(resp.body, tt.want) {
			t.Errorf("%s: body does not contain %q", tt.name, tt.want)
		}
		if strings.Contains(resp.body, testPassword) {
			t.Errorf("%s: the password was echoed back into the page", tt.name)
		}
	}

	if guest.signedIn() {
		t.Error("a failed registration signed the visitor in")
	}
}

func TestLoginAndLogout(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	a.newBrowser().mustRegister("Ada", "ada@example.com")

	b := a.newBrowser()

	// Wrong password and unknown email produce the same response, so the
	// form cannot be used to discover which emails have accounts.
	const generic = "That email and password do not match an account."
	wrong := b.login("ada@example.com", "definitely not it")
	wantStatus(t, wrong, http.StatusUnprocessableEntity)
	wantContains(t, wrong, generic, `value="ada@example.com"`)

	unknown := b.login("nobody@example.com", testPassword)
	wantStatus(t, unknown, http.StatusUnprocessableEntity)
	wantContains(t, unknown, generic)

	overlong := b.login("ada@example.com", strings.Repeat("x", auth.MaxPasswordLength+1))
	wantStatus(t, overlong, http.StatusUnprocessableEntity)
	wantContains(t, overlong, generic)

	wantContains(t, b.login("", ""), "Enter your email address.", "Enter your password.")
	if b.signedIn() {
		t.Fatal("failed logins produced a session")
	}

	// Email matching ignores case and surrounding space.
	wantRedirect(t, b.login("  ADA@example.com ", testPassword), "/")
	if !b.signedIn() {
		t.Fatal("not signed in after a successful login")
	}
	wantRedirect(t, b.get("/login"), "/")

	wantRedirect(t, b.post("/logout", nil), "/")
	wantContains(t, b.get("/"), "You have been signed out.")
	if b.signedIn() {
		t.Fatal("still signed in after logout")
	}
}

func TestLoginRenewsSessionToken(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	a.newBrowser().mustRegister("Ada", "ada@example.com")

	b := a.newBrowser()
	// A failed login stores nothing, so trigger a flash to get an anonymous
	// session cookie first.
	b.mustRegister("Temp", "temp@example.com")
	wantRedirect(t, b.post("/logout", nil), "/")

	base, _ := url.Parse(a.server.URL)
	before := sessionCookie(b, base)
	if before == "" {
		t.Fatal("expected an anonymous session cookie holding the sign-out flash")
	}

	b.mustLogin("ada@example.com", testPassword)
	after := sessionCookie(b, base)
	if after == "" || after == before {
		t.Errorf("session token was not renewed at login (before %q, after %q); this allows session fixation", before, after)
	}
}

func sessionCookie(b *browser, base *url.URL) string {
	for _, c := range b.client.Jar.Cookies(base) {
		if c.Name == "session" {
			return c.Value
		}
	}
	return ""
}

func TestLoginReturnsToRequestedPage(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	a.newBrowser().mustRegister("Ada", "ada@example.com")

	b := a.newBrowser()

	// Visiting a protected page signed out remembers where you were going.
	resp := b.get("/users?q=grace")
	wantRedirect(t, resp, "/login?next=%2Fusers%3Fq%3Dgrace")
	wantContains(t, b.get(resp.location()), `name="next" value="/users?q=grace"`)

	form := url.Values{"email": {"ada@example.com"}, "password": {testPassword}, "next": {"/users?q=grace"}}
	wantRedirect(t, b.post("/login", form), "/users?q=grace")
	wantRedirect(t, b.post("/logout", nil), "/")

	// "next" can never send the user to another site.
	for _, evil := range []string{"https://evil.example/", "//evil.example/", `/\evil.example`, "javascript:alert(1)"} {
		form.Set("next", evil)
		wantRedirect(t, b.post("/login", form), "/")
		wantRedirect(t, b.post("/logout", nil), "/")
	}
}

func TestLoginRateLimit(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	b := a.newBrowser()

	for i := range 10 {
		resp := b.login("nobody@example.com", "wrong password here")
		if resp.status != http.StatusUnprocessableEntity {
			t.Fatalf("attempt %d: status = %d; want 422", i+1, resp.status)
		}
	}

	limited := b.login("nobody@example.com", "wrong password here")
	wantStatus(t, limited, http.StatusTooManyRequests)
	wantContains(t, limited, "Too many attempts")
	if limited.header.Get("Retry-After") == "" {
		t.Error("429 response has no Retry-After header")
	}

	// Viewing the form is not throttled, only submitting it.
	wantStatus(t, b.get("/login"), http.StatusOK)
}

func TestLoginUpgradesOutdatedPasswordHash(t *testing.T) {
	t.Parallel()
	old := newApp(t)
	old.newBrowser().mustRegister("Ada", "ada@example.com")
	before := old.userByEmail("ada@example.com")

	// The same database, now served by a build with stronger parameters.
	stronger := cheapParams
	stronger.Memory = 128
	upgraded := newAppWith(t, old.store, stronger, nil)

	b := upgraded.newBrowser()
	b.mustLogin("ada@example.com", testPassword)

	after := upgraded.userByEmail("ada@example.com")
	if !strings.Contains(after.PasswordHash, "m=128,") {
		t.Errorf("hash was not upgraded on login: %q", after.PasswordHash)
	}
	if !after.PasswordChangedAt.Equal(before.PasswordChangedAt) {
		t.Error("a hash upgrade moved password_changed_at, which would sign the user out everywhere")
	}
	if !b.signedIn() {
		t.Error("session did not survive the hash upgrade")
	}
}
