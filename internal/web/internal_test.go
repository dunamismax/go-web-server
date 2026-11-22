package web

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/dunamismax/go-web-server/internal/store"
	"github.com/dunamismax/go-web-server/internal/view"
)

func TestClientIPIgnoresForwardedHeaderFromUntrustedPeers(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.RemoteAddr = "203.0.113.9:51234"
	req.Header.Set("X-Forwarded-For", "198.51.100.1")

	// No trusted proxies: the header is attacker-controlled and ignored.
	if got := (ipResolver{}).resolve(req); got != "203.0.113.9" {
		t.Errorf("resolve = %q; want the TCP peer 203.0.113.9", got)
	}

	// The peer is not one of the trusted proxies either.
	res := ipResolver{trusted: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}}
	if got := res.resolve(req); got != "203.0.113.9" {
		t.Errorf("resolve = %q; want the TCP peer 203.0.113.9", got)
	}
}

func TestClientIPWalksTrustedProxyChain(t *testing.T) {
	t.Parallel()

	res := ipResolver{trusted: []netip.Prefix{
		netip.MustParsePrefix("127.0.0.1/32"),
		netip.MustParsePrefix("10.0.0.0/8"),
	}}

	tests := []struct {
		name      string
		remote    string
		forwarded []string
		want      string
	}{
		{"single proxy", "127.0.0.1:4000", []string{"198.51.100.7"}, "198.51.100.7"},
		{"two proxies", "127.0.0.1:4000", []string{"198.51.100.7, 10.1.2.3"}, "198.51.100.7"},
		{"spoofed prefix", "127.0.0.1:4000", []string{"1.2.3.4, 198.51.100.7"}, "198.51.100.7"},
		{"split headers", "127.0.0.1:4000", []string{"1.2.3.4", "198.51.100.7, 10.1.2.3"}, "198.51.100.7"},
		{"no header", "127.0.0.1:4000", nil, "127.0.0.1"},
		{"garbage entry", "127.0.0.1:4000", []string{"198.51.100.7, not-an-ip"}, "127.0.0.1"},
		{"all trusted", "127.0.0.1:4000", []string{"10.9.9.9, 10.1.2.3"}, "10.9.9.9"},
		{"ipv4-mapped peer", "[::ffff:127.0.0.1]:4000", []string{"198.51.100.7"}, "198.51.100.7"},
		{"ipv6 client", "127.0.0.1:4000", []string{"2001:db8::1"}, "2001:db8::1"},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.RemoteAddr = tt.remote
		for _, v := range tt.forwarded {
			req.Header.Add("X-Forwarded-For", v)
		}
		if got := res.resolve(req); got != tt.want {
			t.Errorf("%s: resolve = %q; want %q", tt.name, got, tt.want)
		}
	}
}

func TestRateLimiterIsPerKeyAndBounded(t *testing.T) {
	t.Parallel()

	limiter := newRateLimiter(3, time.Hour)

	for i := range 3 {
		if !limiter.allow("a") {
			t.Fatalf("request %d for key a was refused inside the burst", i+1)
		}
	}
	if limiter.allow("a") {
		t.Error("a fourth request for key a was allowed; want it refused")
	}
	if !limiter.allow("b") {
		t.Error("key b was refused because key a is exhausted; limits must be per key")
	}
}

func TestRateLimiterForgetsIdleKeys(t *testing.T) {
	t.Parallel()

	limiter := newRateLimiter(1, time.Millisecond) // TTL of 2ms
	for _, key := range []string{"a", "b", "c"} {
		limiter.allow(key)
	}
	time.Sleep(10 * time.Millisecond)
	limiter.allow("d") // triggers the sweep

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if len(limiter.visitors) != 1 {
		t.Errorf("limiter tracks %d keys after the idle sweep; want 1", len(limiter.visitors))
	}
}

func TestSafeNext(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"/account":               "/account",
		"/users?q=ada&page=2":    "/users?q=ada&page=2",
		"":                       "/",
		"account":                "/",
		"https://evil.example":   "/",
		"//evil.example/path":    "/",
		`/\evil.example`:         "/",
		"/ok\r\nSet-Cookie: x=1": "/",
		"javascript:alert(1)":    "/",
		"/%2F%2Fevil.example":    "/%2F%2Fevil.example", // stays a local path
	}
	for input, want := range tests {
		if got := safeNext(input); got != want {
			t.Errorf("safeNext(%q) = %q; want %q", input, got, want)
		}
	}
}

func TestValidators(t *testing.T) {
	t.Parallel()

	emails := map[string]bool{
		"ada@example.com":          true,
		"ada+tag@sub.example.co":   true,
		"":                         false,
		"ada":                      false,
		"ada@":                     false,
		"Ada <ada@example.com>":    false,
		"ada@example.com, b@c.com": false,
		strings.Repeat("a", 250) + "@example.com": false,
	}
	for email, valid := range emails {
		errs := view.Errors{}
		validateEmail(errs, email)
		if got := !errs.Any(); got != valid {
			t.Errorf("validateEmail(%q) valid = %v; want %v", email, got, valid)
		}
	}

	names := map[string]bool{
		"A":                      true,
		"Zoë Åström":             true,
		strings.Repeat("é", 100): true, // counted in characters, not bytes
		strings.Repeat("x", 101): false,
		"":                       false,
	}
	for name, valid := range names {
		errs := view.Errors{}
		validateName(errs, name)
		if got := !errs.Any(); got != valid {
			t.Errorf("validateName(%q) valid = %v; want %v", name, got, valid)
		}
	}

	passwords := map[string]bool{
		"twelve chars":           true,
		"eleven char":            false,
		strings.Repeat("p", 128): true,
		strings.Repeat("p", 129): false,
	}
	for password, valid := range passwords {
		errs := view.Errors{}
		validateNewPassword(errs, "password", password)
		if got := !errs.Any(); got != valid {
			t.Errorf("validateNewPassword(len %d) valid = %v; want %v", len(password), got, valid)
		}
	}

	for role, valid := range map[store.Role]bool{store.RoleAdmin: true, store.RoleMember: true, "": false, "root": false} {
		errs := view.Errors{}
		validateRole(errs, role)
		if got := !errs.Any(); got != valid {
			t.Errorf("validateRole(%q) valid = %v; want %v", role, got, valid)
		}
	}

	if got := normalizeEmail("  Ada@Example.COM "); got != "ada@example.com" {
		t.Errorf("normalizeEmail = %q; want ada@example.com", got)
	}
}

func TestIsHTMX(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	if isHTMX(req) {
		t.Error("a plain request was treated as htmx")
	}
	req.Header.Set("HX-Request", "true")
	if !isHTMX(req) {
		t.Error("an htmx request was not recognized")
	}
	req.Header.Set("HX-History-Restore-Request", "true")
	if isHTMX(req) {
		t.Error("a history restore must get the full page, not a fragment")
	}
}
