package web_test

import (
	"net/http"
	"net/url"
	"testing"
)

func TestAccountRequiresSignIn(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	b := a.newBrowser()

	wantRedirect(t, b.get("/account"), "/login?next=%2Faccount")
	// A POST has nowhere sensible to return to.
	wantRedirect(t, b.post("/account/profile", url.Values{"name": {"X"}, "email": {"x@example.com"}}), "/login")
}

func TestUpdateProfile(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	a.newBrowser().mustRegister("Grace", "grace@example.com")

	b := a.newBrowser()
	b.mustRegister("Ada", "ada@example.com")

	page := b.get("/account")
	wantStatus(t, page, http.StatusOK)
	wantContains(t, page, `value="Ada"`, `value="ada@example.com"`, "Member since")

	wantRedirect(t, b.post("/account/profile", url.Values{"name": {"Ada Lovelace"}, "email": {"Countess@Example.com"}}), "/account")
	wantContains(t, b.get("/account"), "Your profile was updated.", `value="Ada Lovelace"`, `value="countess@example.com"`)

	// Someone else's email is refused, and the form keeps what was typed.
	taken := b.post("/account/profile", url.Values{"name": {"Ada Lovelace"}, "email": {"grace@example.com"}})
	wantStatus(t, taken, http.StatusUnprocessableEntity)
	wantContains(t, taken, "An account with this email already exists.", `value="grace@example.com"`)

	invalid := b.post("/account/profile", url.Values{"name": {""}, "email": {"nope"}})
	wantStatus(t, invalid, http.StatusUnprocessableEntity)
	wantContains(t, invalid, "Enter a name.", "Enter a valid email address")

	if got := a.userByEmail("countess@example.com").Name; got != "Ada Lovelace" {
		t.Errorf("name = %q after rejected updates; want Ada Lovelace", got)
	}
}

func TestChangePassword(t *testing.T) {
	t.Parallel()
	a := newApp(t)

	laptop := a.newBrowser()
	laptop.mustRegister("Ada", "ada@example.com")
	phone := a.newBrowser()
	phone.mustLogin("ada@example.com", testPassword)

	const newPassword = "an entirely different passphrase"

	wrongCurrent := laptop.post("/account/password", url.Values{"current_password": {"not my password"}, "new_password": {newPassword}})
	wantStatus(t, wrongCurrent, http.StatusUnprocessableEntity)
	wantContains(t, wrongCurrent, "That is not your current password.")

	tooShort := laptop.post("/account/password", url.Values{"current_password": {testPassword}, "new_password": {"short"}})
	wantStatus(t, tooShort, http.StatusUnprocessableEntity)
	wantContains(t, tooShort, "Use at least 12 characters.")

	same := laptop.post("/account/password", url.Values{"current_password": {testPassword}, "new_password": {testPassword}})
	wantStatus(t, same, http.StatusUnprocessableEntity)
	wantContains(t, same, "Choose a password you are not already using.")

	if !phone.signedIn() {
		t.Fatal("rejected password changes signed the other device out")
	}

	wantRedirect(t, laptop.post("/account/password", url.Values{"current_password": {testPassword}, "new_password": {newPassword}}), "/account")
	wantContains(t, laptop.get("/account"), "Your password was changed.")

	// The device that made the change stays signed in; every other session
	// for the account is dead on its next request.
	if !laptop.signedIn() {
		t.Error("the device that changed the password was signed out")
	}
	if phone.signedIn() {
		t.Error("another device is still signed in after the password changed")
	}

	fresh := a.newBrowser()
	wantStatus(t, fresh.login("ada@example.com", testPassword), http.StatusUnprocessableEntity)
	wantRedirect(t, fresh.login("ada@example.com", newPassword), "/")
}
