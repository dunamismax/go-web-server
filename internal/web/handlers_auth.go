package web

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/dunamismax/go-web-server/internal/auth"
	"github.com/dunamismax/go-web-server/internal/store"
	"github.com/dunamismax/go-web-server/internal/view"
)

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) error {
	form := view.LoginForm{Errors: view.Errors{}}
	if next := safeNext(r.URL.Query().Get("next")); next != "/" {
		form.Next = next
	}
	return render(w, r, http.StatusOK, view.Login(s.page(r, "Sign in"), form))
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) error {
	if err := parseForm(r); err != nil {
		return err
	}
	ctx := r.Context()

	form := view.LoginForm{
		Email:  normalizeEmail(r.PostFormValue("email")),
		Errors: view.Errors{},
	}
	if next := safeNext(r.PostFormValue("next")); next != "/" {
		form.Next = next
	}
	password := r.PostFormValue("password")

	if form.Email == "" {
		form.Errors.Add("email", "Enter your email address.")
	}
	if password == "" {
		form.Errors.Add("password", "Enter your password.")
	}
	if form.Errors.Any() {
		return s.renderLogin(w, r, form)
	}

	user, err := s.authenticate(r, form.Email, password)
	if err != nil {
		return err
	}
	if user == nil {
		// One message for every failure, so the form never confirms whether
		// an email is registered.
		form.Errors.Add("", "That email and password do not match an account.")
		return s.renderLogin(w, r, form)
	}
	if !user.IsActive {
		form.Errors.Add("", "This account has been deactivated. Ask an administrator to restore it.")
		return s.renderLogin(w, r, form)
	}

	if err := s.signIn(ctx, *user); err != nil {
		return err
	}
	s.log.InfoContext(ctx, "user signed in", slog.Int64("user_id", user.ID))

	return redirect(w, r, safeNext(form.Next))
}

func (s *Server) renderLogin(w http.ResponseWriter, r *http.Request, form view.LoginForm) error {
	return render(w, r, http.StatusUnprocessableEntity, view.Login(s.page(r, "Sign in"), form))
}

// authenticate returns the user when the credentials are right, and nil when
// they are not. Every failing path costs one password hash, so response time
// does not reveal which emails have accounts.
func (s *Server) authenticate(r *http.Request, email, password string) (*store.User, error) {
	ctx := r.Context()

	if len(password) > auth.MaxPasswordLength {
		s.hasher.VerifyDecoy("")
		return nil, nil //nolint:nilnil // nil user means "wrong credentials", which is not an error
	}

	user, err := s.store.GetUserByEmail(ctx, email)
	if store.IsNotFound(err) {
		s.hasher.VerifyDecoy(password)
		return nil, nil //nolint:nilnil // see above
	}
	if err != nil {
		return nil, err
	}

	ok, err := s.hasher.Verify(password, user.PasswordHash)
	if err != nil {
		s.log.WarnContext(ctx, "stored password hash is unreadable", slog.Int64("user_id", user.ID), slog.Any("error", err))
		return nil, nil //nolint:nilnil // see above
	}
	if !ok {
		return nil, nil //nolint:nilnil // see above
	}

	// The hash parameters were raised since this password was set. Upgrade
	// it now, while the plaintext is in hand. Failing to is not fatal.
	if s.hasher.NeedsRehash(user.PasswordHash) {
		if hash, err := s.hasher.Hash(password); err == nil {
			err = s.store.UpgradeUserPasswordHash(ctx, store.UpgradeUserPasswordHashParams{ID: user.ID, PasswordHash: hash})
			if err != nil {
				s.log.WarnContext(ctx, "could not upgrade password hash", slog.Int64("user_id", user.ID), slog.Any("error", err))
			}
		}
	}

	return &user, nil
}

func (s *Server) registerPage(w http.ResponseWriter, r *http.Request) error {
	form := view.RegisterForm{Errors: view.Errors{}}
	return render(w, r, http.StatusOK, view.Register(s.page(r, "Create an account"), form))
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) error {
	if err := parseForm(r); err != nil {
		return err
	}
	ctx := r.Context()

	form := view.RegisterForm{
		Name:   strings.TrimSpace(r.PostFormValue("name")),
		Email:  normalizeEmail(r.PostFormValue("email")),
		Errors: view.Errors{},
	}
	password := r.PostFormValue("password")

	validateName(form.Errors, form.Name)
	validateEmail(form.Errors, form.Email)
	validateNewPassword(form.Errors, "password", password)
	if form.Errors.Any() {
		return s.renderRegister(w, r, form)
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return err
	}

	user, err := s.store.RegisterUser(ctx, store.RegisterUserParams{
		Email:        form.Email,
		Name:         form.Name,
		PasswordHash: hash,
	})
	if store.IsUniqueViolation(err, store.ConstraintUsersEmail) {
		form.Errors.Add("email", emailTakenMessage)
		return s.renderRegister(w, r, form)
	}
	if err != nil {
		return err
	}

	if err := s.signIn(ctx, user); err != nil {
		return err
	}
	s.log.InfoContext(ctx, "user registered", slog.Int64("user_id", user.ID), slog.String("role", string(user.Role)))

	welcome := "Welcome, " + user.Name + ". Your account is ready."
	if user.IsAdmin() {
		welcome = "Welcome, " + user.Name + ". You are the first user, so you are the administrator."
	}
	s.flashNotice(ctx, welcome)

	return redirect(w, r, "/")
}

func (s *Server) renderRegister(w http.ResponseWriter, r *http.Request, form view.RegisterForm) error {
	return render(w, r, http.StatusUnprocessableEntity, view.Register(s.page(r, "Create an account"), form))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	if user := currentUser(ctx); user != nil {
		if err := s.sessions.Destroy(ctx); err != nil {
			return err
		}
		s.log.InfoContext(ctx, "user signed out", slog.Int64("user_id", user.ID))
		s.flashNotice(ctx, "You have been signed out.")
	}

	return redirect(w, r, "/")
}
