package web_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/dunamismax/go-web-server/internal/auth"
	"github.com/dunamismax/go-web-server/internal/config"
	"github.com/dunamismax/go-web-server/internal/store"
	"github.com/dunamismax/go-web-server/internal/store/storetest"
	"github.com/dunamismax/go-web-server/internal/web"
)

// A password that satisfies the length rule, for accounts the tests create.
const testPassword = "correct horse battery staple"

// cheapParams keep Argon2 fast under test.
var cheapParams = auth.Params{Memory: 64, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}

// app is a running instance of the application on its own database schema.
type app struct {
	t      *testing.T
	store  *store.Store
	server *httptest.Server
}

// newApp starts the full handler stack against an isolated database. The
// test is skipped when no test database is configured.
func newApp(t *testing.T) *app {
	t.Helper()
	return newAppWith(t, storetest.New(t), cheapParams, nil)
}

func newAppWith(t *testing.T, db *store.Store, params auth.Params, env map[string]string) *app {
	t.Helper()

	environ := map[string]string{"DATABASE_URL": os.Getenv(storetest.EnvVar)}
	for k, v := range env {
		environ[k] = v
	}
	cfg, err := config.LoadFrom(environ)
	if err != nil {
		t.Fatalf("config: %v", err)
	}

	hasher, err := auth.NewHasher(params)
	if err != nil {
		t.Fatalf("NewHasher: %v", err)
	}

	srv, err := web.New(web.Options{
		Config: cfg,
		Logger: slog.New(slog.DiscardHandler),
		Store:  db,
		Hasher: hasher,
	})
	if err != nil {
		t.Fatalf("web.New: %v", err)
	}

	server := httptest.NewServer(srv.Handler())
	t.Cleanup(func() {
		server.Close()
		srv.Close()
	})

	return &app{t: t, store: db, server: server}
}

// browser is an HTTP client with its own cookie jar: one signed-in (or
// signed-out) visitor. It never follows redirects, so tests can assert on
// them.
type browser struct {
	t      *testing.T
	base   string
	client *http.Client
}

func (a *app) newBrowser() *browser {
	a.t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		a.t.Fatalf("cookiejar: %v", err)
	}
	return &browser{
		t:    a.t,
		base: a.server.URL,
		client: &http.Client{
			Jar: jar,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

type response struct {
	status int
	header http.Header
	body   string
}

// location returns the redirect target with the test server's origin
// stripped.
func (r response) location() string {
	return r.header.Get("Location")
}

func (b *browser) do(method, path string, form url.Values, headers map[string]string) response {
	b.t.Helper()

	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(b.t.Context(), method, b.base+path, body)
	if err != nil {
		b.t.Fatalf("new request: %v", err)
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		b.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		b.t.Fatalf("read body: %v", err)
	}
	return response{status: resp.StatusCode, header: resp.Header, body: string(data)}
}

func (b *browser) get(path string) response {
	b.t.Helper()
	return b.do(http.MethodGet, path, nil, nil)
}

func (b *browser) post(path string, form url.Values) response {
	b.t.Helper()
	if form == nil {
		form = url.Values{}
	}
	return b.do(http.MethodPost, path, form, nil)
}

// htmx sends the request the way htmx does for an element targeting
// #user-results on the given page.
func (b *browser) htmx(method, path string, form url.Values, currentURL string) response {
	b.t.Helper()
	if form == nil && method == http.MethodPost {
		form = url.Values{}
	}
	return b.do(method, path, form, map[string]string{
		"HX-Request":     "true",
		"HX-Target":      "user-results",
		"HX-Current-URL": b.base + currentURL,
	})
}

func (b *browser) register(name, email string) response {
	b.t.Helper()
	return b.post("/register", url.Values{"name": {name}, "email": {email}, "password": {testPassword}})
}

func (b *browser) login(email, password string) response {
	b.t.Helper()
	return b.post("/login", url.Values{"email": {email}, "password": {password}})
}

// mustRegister signs up a new account and fails the test unless it worked.
func (b *browser) mustRegister(name, email string) {
	b.t.Helper()
	if resp := b.register(name, email); resp.status != http.StatusSeeOther {
		b.t.Fatalf("register %s: status %d, body: %s", email, resp.status, resp.body)
	}
}

func (b *browser) mustLogin(email, password string) {
	b.t.Helper()
	if resp := b.login(email, password); resp.status != http.StatusSeeOther {
		b.t.Fatalf("login %s: status %d, body: %s", email, resp.status, resp.body)
	}
}

// signedIn reports whether the browser's session is authenticated, by
// asking for a page only signed-in users can see.
func (b *browser) signedIn() bool {
	b.t.Helper()
	return b.get("/account").status == http.StatusOK
}

func (a *app) userByEmail(email string) store.User {
	a.t.Helper()
	user, err := a.store.GetUserByEmail(a.t.Context(), email)
	if err != nil {
		a.t.Fatalf("GetUserByEmail(%s): %v", email, err)
	}
	return user
}

func wantStatus(t *testing.T, resp response, want int) {
	t.Helper()
	if resp.status != want {
		t.Fatalf("status = %d; want %d\nbody: %s", resp.status, want, resp.body)
	}
}

func wantRedirect(t *testing.T, resp response, want string) {
	t.Helper()
	if resp.status != http.StatusSeeOther || resp.location() != want {
		t.Fatalf("got status %d to %q; want 303 to %q\nbody: %s", resp.status, resp.location(), want, resp.body)
	}
}

func wantContains(t *testing.T, resp response, substrings ...string) {
	t.Helper()
	for _, s := range substrings {
		if !strings.Contains(resp.body, s) {
			t.Errorf("body does not contain %q\nbody: %s", s, resp.body)
		}
	}
}

func wantNotContains(t *testing.T, resp response, substrings ...string) {
	t.Helper()
	for _, s := range substrings {
		if strings.Contains(resp.body, s) {
			t.Errorf("body unexpectedly contains %q", s)
		}
	}
}
