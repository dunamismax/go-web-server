package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dunamismax/go-web-server/internal/middleware"
	"github.com/dunamismax/go-web-server/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

const testCSRFToken = "csrf-token"

type mockStore struct {
	getUserByEmailFn     func(context.Context, string) (store.User, error)
	createUserFn         func(context.Context, store.CreateUserParams) (store.User, error)
	listUsersFn          func(context.Context) ([]store.User, error)
	countUsersFn         func(context.Context) (int64, error)
	getUserFn            func(context.Context, int64) (store.User, error)
	updateUserFn         func(context.Context, store.UpdateUserParams) (store.User, error)
	updateUserPasswordFn func(context.Context, store.UpdateUserPasswordParams) (store.User, error)
	deactivateUserFn     func(context.Context, int64) error
	deactivateCheckedFn  func(context.Context, int64) (bool, error)
	deleteUserFn         func(context.Context, int64) error
	deleteCheckedFn      func(context.Context, int64) (bool, error)
}

func (m *mockStore) GetUserByEmail(ctx context.Context, email string) (store.User, error) {
	if m.getUserByEmailFn == nil {
		return store.User{}, errors.New("unexpected GetUserByEmail call")
	}
	return m.getUserByEmailFn(ctx, email)
}

func (m *mockStore) CreateUser(ctx context.Context, params store.CreateUserParams) (store.User, error) {
	if m.createUserFn == nil {
		return store.User{}, errors.New("unexpected CreateUser call")
	}
	return m.createUserFn(ctx, params)
}

func (m *mockStore) ListUsers(ctx context.Context) ([]store.User, error) {
	if m.listUsersFn == nil {
		return nil, errors.New("unexpected ListUsers call")
	}
	return m.listUsersFn(ctx)
}

func (m *mockStore) CountUsers(ctx context.Context) (int64, error) {
	if m.countUsersFn == nil {
		return 0, errors.New("unexpected CountUsers call")
	}
	return m.countUsersFn(ctx)
}

func (m *mockStore) GetUser(ctx context.Context, id int64) (store.User, error) {
	if m.getUserFn == nil {
		return store.User{}, errors.New("unexpected GetUser call")
	}
	return m.getUserFn(ctx, id)
}

func (m *mockStore) UpdateUser(ctx context.Context, params store.UpdateUserParams) (store.User, error) {
	if m.updateUserFn == nil {
		return store.User{}, errors.New("unexpected UpdateUser call")
	}
	return m.updateUserFn(ctx, params)
}

func (m *mockStore) UpdateUserPassword(ctx context.Context, params store.UpdateUserPasswordParams) (store.User, error) {
	if m.updateUserPasswordFn == nil {
		return store.User{}, errors.New("unexpected UpdateUserPassword call")
	}
	return m.updateUserPasswordFn(ctx, params)
}

func (m *mockStore) DeactivateUser(ctx context.Context, id int64) error {
	if m.deactivateUserFn == nil {
		return errors.New("unexpected DeactivateUser call")
	}
	return m.deactivateUserFn(ctx, id)
}

func (m *mockStore) DeactivateUserChecked(ctx context.Context, id int64) (bool, error) {
	if m.deactivateCheckedFn == nil {
		return false, errors.New("unexpected DeactivateUserChecked call")
	}
	return m.deactivateCheckedFn(ctx, id)
}

func (m *mockStore) DeleteUser(ctx context.Context, id int64) error {
	if m.deleteUserFn == nil {
		return errors.New("unexpected DeleteUser call")
	}
	return m.deleteUserFn(ctx, id)
}

func (m *mockStore) DeleteUserChecked(ctx context.Context, id int64) (bool, error) {
	if m.deleteCheckedFn == nil {
		return false, errors.New("unexpected DeleteUserChecked call")
	}
	return m.deleteCheckedFn(ctx, id)
}

type mockAuthService struct {
	currentUser   *middleware.User
	currentExists bool
	verifyResult  bool
	verifyErr     error
	hashPassword  string
	hashErr       error
	loginErr      error
	logoutErr     error
	loginCalls    int
	logoutCalls   int
}

func (m *mockAuthService) GetCurrentUser(echo.Context) (*middleware.User, bool) {
	return m.currentUser, m.currentExists
}

func (m *mockAuthService) VerifyPasswordArgon2(string, string) (bool, error) {
	return m.verifyResult, m.verifyErr
}

func (m *mockAuthService) HashPasswordArgon2(string) (string, error) {
	return m.hashPassword, m.hashErr
}

func (m *mockAuthService) LoginUser(echo.Context, middleware.User) error {
	m.loginCalls++
	return m.loginErr
}

func (m *mockAuthService) LogoutUser(echo.Context) error {
	m.logoutCalls++
	return m.logoutErr
}

func (m *mockAuthService) RequireAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	}
}

func newJSONContext(t *testing.T, method, target string, body string) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAccept, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("csrf", testCSRFToken)

	return c, rec
}

func mustDecodeJSON[T any](t *testing.T, body *bytes.Buffer, target *T) {
	t.Helper()

	if err := json.Unmarshal(body.Bytes(), target); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
}

func sampleStoreUser(id int64, active bool) store.User {
	ts := pgtype.Timestamptz{
		Time:  time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC),
		Valid: true,
	}

	return store.User{
		ID:           id,
		Email:        "user@example.com",
		Name:         "Example User",
		PasswordHash: "stored-hash",
		IsActive:     &active,
		CreatedAt:    ts,
		UpdatedAt:    ts,
	}
}

func TestAuthStateReturnsCurrentSessionAndCSRFContract(t *testing.T) {
	t.Parallel()

	auth := &mockAuthService{
		currentUser: &middleware.User{
			ID:       7,
			Email:    "user@example.com",
			Name:     "Example User",
			IsActive: true,
		},
		currentExists: true,
	}
	handler := &AuthHandler{store: &mockStore{}, authService: auth}

	c, rec := newJSONContext(t, http.MethodGet, RouteAPIAuthState, "")
	if err := handler.AuthState(c); err != nil {
		t.Fatalf("AuthState() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp apiAuthStateResponse
	mustDecodeJSON(t, rec.Body, &resp)

	if !resp.Authenticated || resp.User == nil || resp.User.ID != 7 {
		t.Fatalf("unexpected auth response: %+v", resp)
	}

	if resp.CSRF.Token != testCSRFToken || resp.CSRF.Header != csrfHeaderName || resp.CSRF.FormField != csrfFormField {
		t.Fatalf("unexpected csrf contract: %+v", resp.CSRF)
	}
}

func TestLoginAPIReturnsAuthenticatedUserJSON(t *testing.T) {
	t.Parallel()

	storeMock := &mockStore{
		getUserByEmailFn: func(context.Context, string) (store.User, error) {
			return sampleStoreUser(11, true), nil
		},
	}
	auth := &mockAuthService{verifyResult: true}
	handler := &AuthHandler{store: storeMock, authService: auth}

	c, rec := newJSONContext(t, http.MethodPost, RouteAPIAuthLogin, `{"email":"user@example.com","password":"Password1"}`)
	if err := handler.LoginAPI(c); err != nil {
		t.Fatalf("LoginAPI() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp apiAuthMutationResponse
	mustDecodeJSON(t, rec.Body, &resp)

	if resp.Message != MsgLoginSuccess || resp.User == nil || resp.User.ID != 11 {
		t.Fatalf("unexpected login response: %+v", resp)
	}

	if auth.loginCalls != 1 {
		t.Fatalf("loginCalls = %d, want 1", auth.loginCalls)
	}

	if got := rec.Header().Get(csrfHeaderName); got != testCSRFToken {
		t.Fatalf("%s = %q, want %q", csrfHeaderName, got, testCSRFToken)
	}
}

func TestCreateUserAPIReturnsCreatedUser(t *testing.T) {
	t.Parallel()

	storeMock := &mockStore{
		createUserFn: func(_ context.Context, params store.CreateUserParams) (store.User, error) {
			if params.PasswordHash != "hashed-password" {
				t.Fatalf("PasswordHash = %q, want hashed-password", params.PasswordHash)
			}

			user := sampleStoreUser(13, true)
			user.Email = params.Email
			user.Name = params.Name

			return user, nil
		},
	}
	auth := &mockAuthService{hashPassword: "hashed-password"}
	handler := &UserHandler{store: storeMock, authService: auth}

	body := `{"email":"created@example.com","name":"Created User","password":"Password1","confirm_password":"Password1"}`
	c, rec := newJSONContext(t, http.MethodPost, "/api/users", body)
	if err := handler.CreateUserAPI(c); err != nil {
		t.Fatalf("CreateUserAPI() error = %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var resp apiUserMutationResponse
	mustDecodeJSON(t, rec.Body, &resp)

	if resp.Message != MsgUserCreateSuccess || resp.User.Email != "created@example.com" {
		t.Fatalf("unexpected create response: %+v", resp)
	}
}

func TestGetUserAPIReturnsEditContract(t *testing.T) {
	t.Parallel()

	handler := &UserHandler{
		store: &mockStore{
			getUserFn: func(_ context.Context, id int64) (store.User, error) {
				if id != 42 {
					t.Fatalf("id = %d, want 42", id)
				}
				return sampleStoreUser(id, true), nil
			},
		},
		authService: &mockAuthService{},
	}

	c, rec := newJSONContext(t, http.MethodGet, "/api/users/42", "")
	c.SetParamNames("id")
	c.SetParamValues("42")

	if err := handler.GetUserAPI(c); err != nil {
		t.Fatalf("GetUserAPI() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp apiUserResponse
	mustDecodeJSON(t, rec.Body, &resp)

	if resp.User.ID != 42 || resp.User.CreatedAt == "" {
		t.Fatalf("unexpected user response: %+v", resp)
	}
}

func TestDeactivateUserAPIReturnsUpdatedUser(t *testing.T) {
	t.Parallel()

	handler := &UserHandler{
		store: &mockStore{
			deactivateCheckedFn: func(_ context.Context, id int64) (bool, error) {
				if id != 9 {
					t.Fatalf("id = %d, want 9", id)
				}
				return true, nil
			},
			getUserFn: func(_ context.Context, id int64) (store.User, error) {
				user := sampleStoreUser(id, false)
				return user, nil
			},
		},
		authService: &mockAuthService{},
	}

	c, rec := newJSONContext(t, http.MethodPatch, "/api/users/9/deactivate", "")
	c.SetParamNames("id")
	c.SetParamValues("9")

	if err := handler.DeactivateUserAPI(c); err != nil {
		t.Fatalf("DeactivateUserAPI() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp apiUserMutationResponse
	mustDecodeJSON(t, rec.Body, &resp)

	if resp.Message != MsgUserDeactivateSuccess || resp.User.IsActive {
		t.Fatalf("unexpected deactivate response: %+v", resp)
	}
}

func TestDeleteUserAPIReturnsNotFoundWhenMissing(t *testing.T) {
	t.Parallel()

	handler := &UserHandler{
		store: &mockStore{
			deleteCheckedFn: func(context.Context, int64) (bool, error) {
				return false, nil
			},
		},
		authService: &mockAuthService{},
	}

	c, rec := newJSONContext(t, http.MethodDelete, "/api/users/99", "")
	c.SetParamNames("id")
	c.SetParamValues("99")

	err := handler.DeleteUserAPI(c)
	if err == nil {
		t.Fatal("DeleteUserAPI() error = nil, want not found error")
	}

	middleware.ErrorHandler(err, c)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestUsersPageRendersInlineLegacyData(t *testing.T) {
	t.Parallel()

	handler := &UserHandler{
		store: &mockStore{
			listUsersFn: func(context.Context) ([]store.User, error) {
				return []store.User{sampleStoreUser(7, true)}, nil
			},
		},
		authService: &mockAuthService{},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("csrf", testCSRFToken)

	if err := handler.Users(c); err != nil {
		t.Fatalf("Users() error = %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "user@example.com") {
		t.Fatalf("users page did not include user email: %q", body)
	}
	if !strings.Contains(body, `id="user-count"`) || !strings.Contains(body, ">1</span>") {
		t.Fatalf("users page did not include inline count: %q", body)
	}
	if strings.Contains(body, "/users/list") {
		t.Fatalf("users page still referenced retired /users/list fragment: %q", body)
	}
	if strings.Contains(body, "/users/count") {
		t.Fatalf("users page still referenced retired /users/count fragment: %q", body)
	}
}

func TestUserCountAPIReturnsJSONContract(t *testing.T) {
	t.Parallel()

	handler := &UserHandler{
		store: &mockStore{
			countUsersFn: func(context.Context) (int64, error) {
				return 3, nil
			},
		},
		authService: &mockAuthService{},
	}

	c, rec := newJSONContext(t, http.MethodGet, "/api/users/count", "")
	if err := handler.UserCountAPI(c); err != nil {
		t.Fatalf("UserCountAPI() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp apiUserCountResponse
	mustDecodeJSON(t, rec.Body, &resp)

	if resp.Count != 3 {
		t.Fatalf("count = %d, want 3", resp.Count)
	}
}

func TestGetUserAPIReturnsNotFoundForMissingRecord(t *testing.T) {
	t.Parallel()

	handler := &UserHandler{
		store: &mockStore{
			getUserFn: func(context.Context, int64) (store.User, error) {
				return store.User{}, pgx.ErrNoRows
			},
		},
		authService: &mockAuthService{},
	}

	c, rec := newJSONContext(t, http.MethodGet, "/api/users/404", "")
	c.SetParamNames("id")
	c.SetParamValues("404")

	err := handler.GetUserAPI(c)
	if err == nil {
		t.Fatal("GetUserAPI() error = nil, want not found error")
	}

	middleware.ErrorHandler(err, c)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
