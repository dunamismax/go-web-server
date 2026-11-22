package web

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/dunamismax/go-web-server/internal/store"
	"github.com/dunamismax/go-web-server/internal/view"
)

const (
	usersPageSize   = 20
	maxSearchLength = 100
)

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) error {
	list, err := s.loadUserList(r.Context(), r.URL.Query())
	if err != nil {
		return err
	}

	// The search box asks for just the table. Anything else, including a
	// direct visit to the same URL, gets the whole page.
	if isHTMX(r) && r.Header.Get("HX-Target") == "user-results" {
		return render(w, r, http.StatusOK, view.UserResults(list))
	}
	return render(w, r, http.StatusOK, view.Users(s.page(r, "Users"), list))
}

// loadUserList reads one page of users for the "q" and "page" parameters in
// query.
func (s *Server) loadUserList(ctx context.Context, query url.Values) (view.UserList, error) {
	search := strings.TrimSpace(query.Get("q"))
	if len(search) > maxSearchLength {
		search = search[:maxSearchLength]
	}

	total, err := s.store.CountUsers(ctx, search)
	if err != nil {
		return view.UserList{}, err
	}

	list := view.UserList{
		Search:   search,
		Page:     1,
		PageSize: usersPageSize,
		Total:    total,
		SelfID:   currentUser(ctx).ID,
	}

	// Clamp the page so a stale link past the end shows the last page
	// rather than an empty table.
	lastPage := max(1, int((total+usersPageSize-1)/usersPageSize))
	if page, err := strconv.Atoi(query.Get("page")); err == nil && page > 1 {
		list.Page = min(page, lastPage)
	}

	list.Users, err = s.store.ListUsers(ctx, store.ListUsersParams{
		Search:     search,
		PageSize:   usersPageSize,
		PageOffset: int32((list.Page - 1) * usersPageSize), //nolint:gosec // bounded by lastPage
	})
	return list, err
}

func (s *Server) newUserPage(w http.ResponseWriter, r *http.Request) error {
	form := view.UserForm{Role: store.RoleMember, Errors: view.Errors{}}
	return render(w, r, http.StatusOK, view.UserNew(s.page(r, "New user"), form))
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) error {
	if err := parseForm(r); err != nil {
		return err
	}
	ctx := r.Context()

	form := view.UserForm{
		Name:   strings.TrimSpace(r.PostFormValue("name")),
		Email:  normalizeEmail(r.PostFormValue("email")),
		Role:   store.Role(r.PostFormValue("role")),
		Errors: view.Errors{},
	}
	password := r.PostFormValue("password")

	validateName(form.Errors, form.Name)
	validateEmail(form.Errors, form.Email)
	validateRole(form.Errors, form.Role)
	validateNewPassword(form.Errors, "password", password)

	if !form.Errors.Any() {
		hash, err := s.hasher.Hash(password)
		if err != nil {
			return err
		}
		created, err := s.store.CreateUser(ctx, store.CreateUserParams{
			Email:        form.Email,
			Name:         form.Name,
			PasswordHash: hash,
			Role:         form.Role,
		})
		switch {
		case store.IsUniqueViolation(err, store.ConstraintUsersEmail):
			form.Errors.Add("email", emailTakenMessage)
		case err != nil:
			return err
		default:
			s.logAdminAction(ctx, "user created", created.ID)
		}
	}
	if form.Errors.Any() {
		return render(w, r, http.StatusUnprocessableEntity, view.UserNew(s.page(r, "New user"), form))
	}

	s.flashNotice(ctx, form.Name+" was added.")
	return redirect(w, r, "/users")
}

func (s *Server) editUserPage(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	target, err := s.userFromPath(r)
	if err != nil {
		return err
	}

	form := view.UserForm{
		ID:     target.ID,
		Name:   target.Name,
		Email:  target.Email,
		Role:   target.Role,
		IsSelf: target.ID == currentUser(ctx).ID,
		Errors: view.Errors{},
	}
	return render(w, r, http.StatusOK, view.UserEdit(s.page(r, "Edit user"), form))
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) error {
	if err := parseForm(r); err != nil {
		return err
	}
	ctx := r.Context()

	target, err := s.userFromPath(r)
	if err != nil {
		return err
	}
	isSelf := target.ID == currentUser(ctx).ID

	form := view.UserForm{
		ID:     target.ID,
		Name:   strings.TrimSpace(r.PostFormValue("name")),
		Email:  normalizeEmail(r.PostFormValue("email")),
		Role:   store.Role(r.PostFormValue("role")),
		IsSelf: isSelf,
		Errors: view.Errors{},
	}
	if isSelf {
		// Admins cannot demote themselves, so there is always at least one
		// admin left. The role field is disabled in the form; ignore it here
		// too in case it was posted anyway.
		form.Role = target.Role
	}
	password := r.PostFormValue("password")

	validateName(form.Errors, form.Name)
	validateEmail(form.Errors, form.Email)
	validateRole(form.Errors, form.Role)
	if password != "" {
		validateNewPassword(form.Errors, "password", password)
	}

	var updated store.User
	if !form.Errors.Any() {
		var hash string
		if password != "" {
			if hash, err = s.hasher.Hash(password); err != nil {
				return err
			}
		}

		// Both statements commit together or not at all.
		err = s.store.InTx(ctx, func(q *store.Queries) error {
			var err error
			updated, err = q.UpdateUser(ctx, store.UpdateUserParams{
				ID:    target.ID,
				Name:  form.Name,
				Email: form.Email,
				Role:  form.Role,
			})
			if err != nil || hash == "" {
				return err
			}
			updated, err = q.UpdateUserPassword(ctx, store.UpdateUserPasswordParams{ID: target.ID, PasswordHash: hash})
			return err
		})
		switch {
		case store.IsUniqueViolation(err, store.ConstraintUsersEmail):
			form.Errors.Add("email", emailTakenMessage)
		case store.IsNotFound(err):
			return errNotFound()
		case err != nil:
			return err
		}
	}
	if form.Errors.Any() {
		return render(w, r, http.StatusUnprocessableEntity, view.UserEdit(s.page(r, "Edit user"), form))
	}

	if isSelf && password != "" {
		// Changing a password ends that user's sessions. Keep the admin who
		// just changed their own signed in on this device.
		if err := s.signIn(ctx, updated); err != nil {
			return err
		}
	}
	s.logAdminAction(ctx, "user updated", target.ID)

	s.flashNotice(ctx, updated.Name+" was updated.")
	return redirect(w, r, "/users")
}

func (s *Server) setUserActive(w http.ResponseWriter, r *http.Request) error {
	if err := parseForm(r); err != nil {
		return err
	}
	ctx := r.Context()

	target, err := s.userFromPath(r)
	if err != nil {
		return err
	}
	active, err := strconv.ParseBool(r.PostFormValue("active"))
	if err != nil {
		return errBadRequest("The form did not say whether to activate or deactivate the user.")
	}

	var notice, alert string
	if target.ID == currentUser(ctx).ID {
		alert = "You cannot deactivate your own account."
	} else {
		updated, err := s.store.SetUserActive(ctx, store.SetUserActiveParams{ID: target.ID, IsActive: active})
		if store.IsNotFound(err) {
			return errNotFound()
		}
		if err != nil {
			return err
		}
		s.logAdminAction(ctx, "user active flag changed", target.ID, slog.Bool("active", active))

		notice = updated.Name + " was deactivated and signed out."
		if active {
			notice = updated.Name + " was reactivated."
		}
	}

	// From the table, htmx swaps in a fresh copy of the list the admin was
	// looking at, taken from the page URL it reports.
	if isHTMX(r) {
		var query url.Values
		if current, err := url.Parse(r.Header.Get("HX-Current-URL")); err == nil {
			query = current.Query()
		}
		list, err := s.loadUserList(ctx, query)
		if err != nil {
			return err
		}
		list.Notice, list.Alert = notice, alert
		return render(w, r, http.StatusOK, view.UserResults(list))
	}

	if alert != "" {
		s.flashAlert(ctx, alert)
	} else {
		s.flashNotice(ctx, notice)
	}
	return redirect(w, r, "/users")
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	target, err := s.userFromPath(r)
	if err != nil {
		return err
	}
	if target.ID == currentUser(ctx).ID {
		s.flashAlert(ctx, "You cannot delete your own account.")
		return redirect(w, r, "/users")
	}

	deleted, err := s.store.DeleteUser(ctx, target.ID)
	if err != nil {
		return err
	}
	if deleted == 0 {
		return errNotFound()
	}
	s.logAdminAction(ctx, "user deleted", target.ID)

	s.flashNotice(ctx, target.Name+" was deleted.")
	return redirect(w, r, "/users")
}

// userFromPath loads the user named by the {id} path segment, or returns a
// 404 error.
func (s *Server) userFromPath(r *http.Request) (store.User, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		return store.User{}, errNotFound()
	}

	user, err := s.store.GetUser(r.Context(), id)
	if store.IsNotFound(err) {
		return store.User{}, errNotFound()
	}
	return user, err
}

// logAdminAction records who did what to whom. It is a minimal audit trail;
// a real one belongs in the database.
func (s *Server) logAdminAction(ctx context.Context, action string, targetID int64, attrs ...slog.Attr) {
	attrs = append(attrs,
		slog.Int64("admin_id", currentUser(ctx).ID),
		slog.Int64("target_user_id", targetID),
		slog.String("request_id", requestIDFrom(ctx)),
	)
	s.log.LogAttrs(ctx, slog.LevelInfo, action, attrs...)
}
