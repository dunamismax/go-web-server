package web_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestPublicPagesAndProbes(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	b := a.newBrowser()

	home := b.get("/")
	wantStatus(t, home, http.StatusOK)
	wantContains(t, home, "<!doctype html>", "A Go web app you can start from.", `href="/register"`)

	for path, heading := range map[string]string{"/login": "Sign in", "/register": "Create an account"} {
		resp := b.get(path)
		wantStatus(t, resp, http.StatusOK)
		wantContains(t, resp, heading)
	}

	health := b.get("/healthz")
	wantStatus(t, health, http.StatusOK)
	wantContains(t, health, `"status":"ok"`)

	ready := b.get("/readyz")
	wantStatus(t, ready, http.StatusOK)
	wantContains(t, ready, `"database":"ok"`)

	missing := b.get("/no-such-page")
	wantStatus(t, missing, http.StatusNotFound)
	wantContains(t, missing, "Page not found")
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()
	a := newApp(t)

	resp := a.newBrowser().get("/")

	want := map[string]string{
		"Content-Security-Policy":      "default-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'; object-src 'none'",
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Referrer-Policy":              "strict-origin-when-cross-origin",
		"Cross-Origin-Opener-Policy":   "same-origin",
		"Cross-Origin-Resource-Policy": "same-origin",
		"Cache-Control":                "no-store",
	}
	for name, value := range want {
		if got := resp.header.Get(name); got != value {
			t.Errorf("%s = %q; want %q", name, got, value)
		}
	}
	if resp.header.Get("X-Request-Id") == "" {
		t.Error("X-Request-Id header is missing")
	}
	// HSTS is only sent in production, where TLS is expected.
	if got := resp.header.Get("Strict-Transport-Security"); got != "" {
		t.Errorf("Strict-Transport-Security = %q in development; want none", got)
	}

	// Nothing inline: the CSP above would block it anyway.
	for _, banned := range []string{"<style", " style=", "onclick=", "<script>"} {
		if strings.Contains(resp.body, banned) {
			t.Errorf("page contains inline %q, which the CSP forbids", banned)
		}
	}
}

func TestProductionHardening(t *testing.T) {
	t.Parallel()
	db := newApp(t).store
	a := newAppWith(t, db, cheapParams, map[string]string{"APP_ENV": "production"})
	b := a.newBrowser()

	resp := b.get("/")
	if got := resp.header.Get("Strict-Transport-Security"); !strings.HasPrefix(got, "max-age=") {
		t.Errorf("Strict-Transport-Security = %q in production; want a max-age", got)
	}

	// The session cookie is issued Secure, HttpOnly and with the __Host-
	// prefix once there is something to store.
	resp = b.register("Prod User", "prod@example.com")
	wantStatus(t, resp, http.StatusSeeOther)
	cookie := resp.header.Get("Set-Cookie")
	for _, attr := range []string{"__Host-session=", "Secure", "HttpOnly", "SameSite=Lax", "Path=/"} {
		if !strings.Contains(cookie, attr) {
			t.Errorf("Set-Cookie %q is missing %q", cookie, attr)
		}
	}
}

func TestStaticAssets(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	b := a.newBrowser()

	// The layout links fingerprinted asset URLs; fetch exactly what it links.
	home := b.get("/")
	for _, name := range []string{"app.css", "htmx.min.js"} {
		start := strings.Index(home.body, "/static/"+name)
		if start < 0 {
			t.Fatalf("home page does not link %s", name)
		}
		end := start + strings.IndexAny(home.body[start:], `"'`)
		assetURL := home.body[start:end]
		if !strings.Contains(assetURL, "?v=") {
			t.Errorf("%s is linked as %q; want a ?v= content hash", name, assetURL)
		}

		resp := b.get(assetURL)
		wantStatus(t, resp, http.StatusOK)
		if got := resp.header.Get("Cache-Control"); !strings.Contains(got, "immutable") {
			t.Errorf("%s Cache-Control = %q; want immutable", name, got)
		}
		if len(resp.body) < 1000 {
			t.Errorf("%s is only %d bytes; was it built?", name, len(resp.body))
		}
	}

	wantStatus(t, b.get("/favicon.ico"), http.StatusOK)
	wantStatus(t, b.get("/static/"), http.StatusNotFound)
	wantStatus(t, b.get("/static/missing.css"), http.StatusNotFound)
}

func TestCrossSitePostsAreRejected(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	b := a.newBrowser()
	b.mustRegister("Ada", "ada@example.com")

	form := url.Values{"name": {"Mallory"}, "email": {"ada@example.com"}}

	// A form on another site, submitted by the victim's browser.
	crossSite := b.do(http.MethodPost, "/account/profile", form, map[string]string{"Sec-Fetch-Site": "cross-site"})
	wantStatus(t, crossSite, http.StatusForbidden)
	wantContains(t, crossSite, "came from another site")

	// An older browser without Sec-Fetch-Site still sends Origin.
	foreignOrigin := b.do(http.MethodPost, "/account/profile", form, map[string]string{"Origin": "https://evil.example"})
	wantStatus(t, foreignOrigin, http.StatusForbidden)

	if name := a.userByEmail("ada@example.com").Name; name != "Ada" {
		t.Fatalf("name = %q after blocked cross-site posts; want it unchanged", name)
	}

	// The same post from the app's own pages goes through.
	sameOrigin := b.do(http.MethodPost, "/account/profile", form, map[string]string{"Sec-Fetch-Site": "same-origin"})
	wantRedirect(t, sameOrigin, "/account")
	if name := a.userByEmail("ada@example.com").Name; name != "Mallory" {
		t.Fatalf("name = %q after same-origin post; want Mallory", name)
	}
}

func TestOversizedBodyIsRejected(t *testing.T) {
	t.Parallel()
	a := newApp(t)

	huge := url.Values{"email": {"a@example.com"}, "password": {strings.Repeat("x", 2<<20)}}
	resp := a.newBrowser().post("/login", huge)
	wantStatus(t, resp, http.StatusRequestEntityTooLarge)
}
