package web

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/dunamismax/go-web-server/internal/store"
	"github.com/dunamismax/go-web-server/internal/view"
)

func (s *Server) accountPage(w http.ResponseWriter, r *http.Request) error {
	user := currentUser(r.Context())
	return s.renderAccount(w, r, http.StatusOK, profileFormFor(user), view.PasswordForm{Errors: view.Errors{}})
}

func (s *Server) renderAccount(w http.ResponseWriter, r *http.Request, status int, profile view.ProfileForm, password view.PasswordForm) error {
	return render(w, r, status, view.Account(s.page(r, "Your account"), profile, password))
}

func profileFormFor(user *store.User) view.ProfileForm {
	return view.ProfileForm{Name: user.Name, Email: user.Email, Errors: view.Errors{}}
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) error {
	if err := parseForm(r); err != nil {
		return err
	}
	ctx := r.Context()
	user := currentUser(ctx)

	form := view.ProfileForm{
		Name:   strings.TrimSpace(r.PostFormValue("name")),
		Email:  normalizeEmail(r.PostFormValue("email")),
		Errors: view.Errors{},
	}
	validateName(form.Errors, form.Name)
	validateEmail(form.Errors, form.Email)

	if !form.Errors.Any() {
		_, err := s.store.UpdateUserProfile(ctx, store.UpdateUserProfileParams{
			ID:    user.ID,
			Name:  form.Name,
			Email: form.Email,
		})
		switch {
		case store.IsUniqueViolation(err, store.ConstraintUsersEmail):
			form.Errors.Add("email", emailTakenMessage)
		case err != nil:
			return err
		}
	}
	if form.Errors.Any() {
		return s.renderAccount(w, r, http.StatusUnprocessableEntity, form, view.PasswordForm{Errors: view.Errors{}})
	}

	s.flashNotice(ctx, "Your profile was updated.")
	return redirect(w, r, "/account")
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) error {
	if err := parseForm(r); err != nil {
		return err
	}
	ctx := r.Context()
	user := currentUser(ctx)

	current := r.PostFormValue("current_password")
	replacement := r.PostFormValue("new_password")
	form := view.PasswordForm{Errors: view.Errors{}}

	// Asking for the current password stops someone at an unattended,
	// signed-in browser from taking over the account.
	if ok, err := s.hasher.Verify(current, user.PasswordHash); err != nil || !ok {
		form.Errors.Add("current_password", "That is not your current password.")
	}
	validateNewPassword(form.Errors, "new_password", replacement)
	if !form.Errors.Any() && replacement == current {
		form.Errors.Add("new_password", "Choose a password you are not already using.")
	}
	if form.Errors.Any() {
		return s.renderAccount(w, r, http.StatusUnprocessableEntity, profileFormFor(user), form)
	}

	hash, err := s.hasher.Hash(replacement)
	if err != nil {
		return err
	}
	updated, err := s.store.UpdateUserPassword(ctx, store.UpdateUserPasswordParams{ID: user.ID, PasswordHash: hash})
	if err != nil {
		return err
	}

	// The update moved password_changed_at, which invalidates every session
	// for this user. Re-bind this one so only the other devices are signed
	// out.
	if err := s.signIn(ctx, updated); err != nil {
		return err
	}
	s.log.InfoContext(ctx, "user changed password", slog.Int64("user_id", user.ID))

	s.flashNotice(ctx, "Your password was changed. Other devices have been signed out.")
	return redirect(w, r, "/account")
}
