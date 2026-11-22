package web_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/dunamismax/go-web-server/internal/store"
)

// adminAndMember registers the first (admin) account and a second (member)
// account, returning a signed-in browser for each.
func adminAndMember(t *testing.T, a *app) (admin, member *browser) {
	t.Helper()

	admin = a.newBrowser()
	admin.mustRegister("Ada Admin", "ada@example.com")
	member = a.newBrowser()
	member.mustRegister("Grace Member", "grace@example.com")
	return admin, member
}

func userPath(u store.User, suffix string) string {
	return "/users/" + strconv.FormatInt(u.ID, 10) + suffix
}

func TestUserManagementIsAdminOnly(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	admin, member := adminAndMember(t, a)
	ada := a.userByEmail("ada@example.com")

	guest := a.newBrowser()
	wantRedirect(t, guest.get("/users"), "/login?next=%2Fusers")

	// A member is refused on every admin route, reads and writes alike.
	requests := []struct {
		method, path string
	}{
		{http.MethodGet, "/users"},
		{http.MethodGet, "/users/new"},
		{http.MethodPost, "/users"},
		{http.MethodGet, userPath(ada, "/edit")},
		{http.MethodPost, userPath(ada, "")},
		{http.MethodPost, userPath(ada, "/active")},
		{http.MethodPost, userPath(ada, "/delete")},
	}
	for _, req := range requests {
		var form url.Values
		if req.method == http.MethodPost {
			form = url.Values{"name": {"Hacked"}, "email": {"hacked@example.com"}, "role": {"admin"}, "active": {"false"}}
		}
		resp := member.do(req.method, req.path, form, nil)
		if resp.status != http.StatusForbidden {
			t.Errorf("member %s %s: status = %d; want 403", req.method, req.path, resp.status)
		}
	}
	if after := a.userByEmail("ada@example.com"); after.Name != "Ada Admin" || !after.IsActive {
		t.Errorf("a member changed the admin account: %+v", after)
	}

	// The nav only offers the Users link to admins.
	wantNotContains(t, member.get("/"), `href="/users"`)
	wantContains(t, admin.get("/"), `href="/users"`)

	wantStatus(t, admin.get("/users"), http.StatusOK)
}

func TestUserListSearchAndPagination(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	admin, _ := adminAndMember(t, a)

	for i := range 21 {
		_, err := a.store.CreateUser(t.Context(), store.CreateUserParams{
			Email: fmt.Sprintf("bulk%02d@example.com", i), Name: fmt.Sprintf("Bulk %02d", i),
			PasswordHash: "x", Role: store.RoleMember,
		})
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
	}

	// 23 users at 20 per page.
	first := admin.get("/users")
	wantStatus(t, first, http.StatusOK)
	wantContains(t, first, "1–20 of 23 users", "Page 1 of 2", `href="/users?page=2"`, "Bulk 20")
	wantNotContains(t, first, "ada@example.com") // oldest, so on page two

	second := admin.get("/users?page=2")
	wantContains(t, second, "21–23 of 23 users", "ada@example.com", "(you)", `href="/users"`)

	// A page past the end shows the last page rather than nothing.
	wantContains(t, admin.get("/users?page=99"), "21–23 of 23 users")
	wantContains(t, admin.get("/users?page=banana"), "1–20 of 23 users")

	search := admin.get("/users?q=GRACE")
	wantContains(t, search, "Grace Member", "1 user", `value="GRACE"`)
	wantNotContains(t, search, "Bulk 00", "Page 1 of")

	wantContains(t, admin.get("/users?q=zzz"), `No users match &#34;zzz&#34;.`)

	// Search terms are text, never markup or SQL.
	hostile := admin.get("/users?q=" + url.QueryEscape(`<script>alert(1)</script>' OR 1=1 --`))
	wantStatus(t, hostile, http.StatusOK)
	wantNotContains(t, hostile, "<script>alert(1)</script>")
	wantContains(t, hostile, "No users match")
}

func TestUserListHTMXFragment(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	admin, _ := adminAndMember(t, a)

	// The search box asks for the results region only.
	fragment := admin.htmx(http.MethodGet, "/users?q=grace", nil, "/users")
	wantStatus(t, fragment, http.StatusOK)
	wantContains(t, fragment, `<div id="user-results"`, "Grace Member")
	wantNotContains(t, fragment, "<!doctype html>", "<nav", "Ada Admin")

	// Restoring history, or an htmx request aimed elsewhere, needs the page.
	restore := admin.do(http.MethodGet, "/users", nil, map[string]string{
		"HX-Request": "true", "HX-Target": "user-results", "HX-History-Restore-Request": "true",
	})
	wantContains(t, restore, "<!doctype html>")

	// An expired session during an htmx request navigates the whole window
	// to the sign-in page instead of swapping it into the table.
	guest := a.newBrowser()
	expired := guest.htmx(http.MethodGet, "/users", nil, "/users")
	if expired.header.Get("HX-Redirect") != "/login" {
		t.Errorf("signed-out htmx request: HX-Redirect = %q (status %d); want /login", expired.header.Get("HX-Redirect"), expired.status)
	}
}

func TestCreateUser(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	admin, _ := adminAndMember(t, a)

	wantContains(t, admin.get("/users/new"), "New user", `name="role"`)

	invalid := admin.post("/users", url.Values{"name": {""}, "email": {"bad"}, "role": {"emperor"}, "password": {"short"}})
	wantStatus(t, invalid, http.StatusUnprocessableEntity)
	wantContains(t, invalid, "Enter a name.", "Enter a valid email address", "Choose a role.", "Use at least 12 characters.")

	taken := admin.post("/users", url.Values{"name": {"Dup"}, "email": {"Grace@example.com"}, "role": {"member"}, "password": {testPassword}})
	wantStatus(t, taken, http.StatusUnprocessableEntity)
	wantContains(t, taken, "An account with this email already exists.", `value="Dup"`)

	created := admin.post("/users", url.Values{"name": {"Linus"}, "email": {"Linus@Example.com"}, "role": {"admin"}, "password": {testPassword}})
	wantRedirect(t, created, "/users")
	wantContains(t, admin.get("/users"), "Linus was added.", "linus@example.com")

	linus := a.userByEmail("linus@example.com")
	if linus.Role != store.RoleAdmin || !linus.IsActive {
		t.Errorf("created user = %+v; want an active admin", linus)
	}
	// The password the admin set works.
	a.newBrowser().mustLogin("linus@example.com", testPassword)
}

func TestEditUser(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	admin, member := adminAndMember(t, a)
	grace := a.userByEmail("grace@example.com")

	page := admin.get(userPath(grace, "/edit"))
	wantStatus(t, page, http.StatusOK)
	wantContains(t, page, `value="Grace Member"`, `value="grace@example.com"`, "Delete this user")

	wantStatus(t, admin.get("/users/999999/edit"), http.StatusNotFound)
	wantStatus(t, admin.get("/users/abc/edit"), http.StatusNotFound)
	wantStatus(t, admin.post("/users/999999", url.Values{"name": {"X"}, "email": {"x@example.com"}, "role": {"member"}}), http.StatusNotFound)

	invalid := admin.post(userPath(grace, ""), url.Values{"name": {"Grace"}, "email": {"ada@example.com"}, "role": {"member"}})
	wantStatus(t, invalid, http.StatusUnprocessableEntity)
	wantContains(t, invalid, "An account with this email already exists.")

	// Promote without touching the password: Grace's session survives and
	// gains admin access on its next request.
	promoted := admin.post(userPath(grace, ""), url.Values{"name": {"Grace Hopper"}, "email": {"grace@example.com"}, "role": {"admin"}, "password": {""}})
	wantRedirect(t, promoted, "/users")
	wantContains(t, admin.get("/users"), "Grace Hopper was updated.")
	wantStatus(t, member.get("/users"), http.StatusOK)

	// Resetting her password signs her out everywhere.
	const resetTo = "freshly reset passphrase"
	reset := admin.post(userPath(grace, ""), url.Values{"name": {"Grace Hopper"}, "email": {"grace@example.com"}, "role": {"member"}, "password": {resetTo}})
	wantRedirect(t, reset, "/users")
	if member.signedIn() {
		t.Error("user is still signed in after an admin reset their password")
	}
	fresh := a.newBrowser()
	wantStatus(t, fresh.login("grace@example.com", testPassword), http.StatusUnprocessableEntity)
	fresh.mustLogin("grace@example.com", resetTo)
	wantStatus(t, fresh.get("/users"), http.StatusForbidden) // demoted in the same edit

	short := admin.post(userPath(grace, ""), url.Values{"name": {"Grace Hopper"}, "email": {"grace@example.com"}, "role": {"member"}, "password": {"short"}})
	wantStatus(t, short, http.StatusUnprocessableEntity)
}

func TestAdminCannotLockThemselvesOut(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	admin, _ := adminAndMember(t, a)
	ada := a.userByEmail("ada@example.com")

	// Their own edit page offers neither a role choice nor deletion.
	page := admin.get(userPath(ada, "/edit"))
	wantContains(t, page, "You cannot change your own role.")
	wantNotContains(t, page, "Delete this user")
	wantNotContains(t, admin.get("/users?q=ada"), userPath(ada, "/active"))

	// Posting the forbidden changes directly is refused as well.
	wantRedirect(t, admin.post(userPath(ada, ""), url.Values{"name": {"Ada Admin"}, "email": {"ada@example.com"}, "role": {"member"}}), "/users")
	wantRedirect(t, admin.post(userPath(ada, "/active"), url.Values{"active": {"false"}}), "/users")
	wantContains(t, admin.get("/users"), "You cannot deactivate your own account.")
	wantRedirect(t, admin.post(userPath(ada, "/delete"), nil), "/users")
	wantContains(t, admin.get("/users"), "You cannot delete your own account.")

	after := a.userByEmail("ada@example.com")
	if after.Role != store.RoleAdmin || !after.IsActive {
		t.Errorf("admin after self-directed changes = %+v; want an active admin", after)
	}

	// Changing their own password from the admin form keeps them signed in.
	wantRedirect(t, admin.post(userPath(ada, ""), url.Values{"name": {"Ada Admin"}, "email": {"ada@example.com"}, "role": {"admin"}, "password": {"my brand new passphrase"}}), "/users")
	if !admin.signedIn() {
		t.Error("admin was signed out by changing their own password in the admin form")
	}
}

func TestDeactivateAndReactivateUser(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	admin, member := adminAndMember(t, a)
	grace := a.userByEmail("grace@example.com")

	// From the table, htmx gets the refreshed results for the list the admin
	// was looking at (here a search), with the outcome shown inline.
	fragment := admin.htmx(http.MethodPost, userPath(grace, "/active"), url.Values{"active": {"false"}}, "/users?q=grace")
	wantStatus(t, fragment, http.StatusOK)
	wantContains(t, fragment, `<div id="user-results"`, "Grace Member was deactivated and signed out.", "Inactive", "Activate")
	wantNotContains(t, fragment, "<!doctype html>", "Ada Admin")

	// Deactivation takes effect on the user's very next request.
	if member.signedIn() {
		t.Error("deactivated user is still signed in")
	}
	blocked := a.newBrowser().login("grace@example.com", testPassword)
	wantStatus(t, blocked, http.StatusUnprocessableEntity)
	wantContains(t, blocked, "This account has been deactivated.")
	// ...but only someone who knows the password learns that.
	wantNotContains(t, a.newBrowser().login("grace@example.com", "wrong password here"), "deactivated")

	// Without JavaScript the same form posts normally and redirects.
	wantRedirect(t, admin.post(userPath(grace, "/active"), url.Values{"active": {"true"}}), "/users")
	wantContains(t, admin.get("/users"), "Grace Member was reactivated.")
	a.newBrowser().mustLogin("grace@example.com", testPassword)

	wantStatus(t, admin.post("/users/999999/active", url.Values{"active": {"false"}}), http.StatusNotFound)

	// A post that does not say which way to flip the account changes nothing.
	wantStatus(t, admin.post(userPath(grace, "/active"), nil), http.StatusBadRequest)
	wantStatus(t, admin.post(userPath(grace, "/active"), url.Values{"active": {"maybe"}}), http.StatusBadRequest)
	if !a.userByEmail("grace@example.com").IsActive {
		t.Error("a malformed activate request deactivated the user")
	}
}

func TestDeleteUser(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	admin, member := adminAndMember(t, a)
	grace := a.userByEmail("grace@example.com")

	// htmx (after its confirm dialog) is told to navigate, not to swap.
	resp := admin.htmx(http.MethodPost, userPath(grace, "/delete"), nil, userPath(grace, "/edit"))
	if resp.status != http.StatusNoContent || resp.header.Get("HX-Redirect") != "/users" {
		t.Fatalf("htmx delete: status %d, HX-Redirect %q; want 204 and /users", resp.status, resp.header.Get("HX-Redirect"))
	}
	list := admin.get("/users")
	wantContains(t, list, "Grace Member was deleted.")
	if strings.Contains(list.body, "grace@example.com") {
		t.Error("deleted user is still listed")
	}

	if member.signedIn() {
		t.Error("deleted user is still signed in")
	}
	wantStatus(t, a.newBrowser().login("grace@example.com", testPassword), http.StatusUnprocessableEntity)
	wantStatus(t, admin.post(userPath(grace, "/delete"), nil), http.StatusNotFound)
}
