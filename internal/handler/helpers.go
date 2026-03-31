package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/dunamismax/go-web-server/internal/middleware"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/labstack/echo/v4"
)

// setupCSRFHeaders sets CSRF token in response headers if available
func setupCSRFHeaders(c echo.Context) string {
	token := middleware.GetCSRFToken(c)
	if token != "" {
		c.Response().Header().Set("X-CSRF-Token", token)
	}
	return token
}

// Error helpers for common error patterns

// validationError creates a validation error with context
func validationError(c echo.Context, err error) error {
	return middleware.NewAppError(
		middleware.ErrorTypeValidation,
		http.StatusBadRequest,
		"Invalid request format",
	).WithContext(c).WithInternal(err)
}

// validationErrorWithDetails creates a validation error with validation details
func validationErrorWithDetails(c echo.Context, details interface{}) error {
	return middleware.NewAppErrorWithDetails(
		middleware.ErrorTypeValidation,
		http.StatusBadRequest,
		"Validation failed",
		details,
	).WithContext(c)
}

// authenticationError creates an authentication error
func authenticationError(c echo.Context, message string) error {
	return middleware.NewAppError(
		middleware.ErrorTypeAuthentication,
		http.StatusUnauthorized,
		message,
	).WithContext(c)
}

// notFoundError creates a not-found error with context.
func notFoundError(c echo.Context, message string) error {
	return middleware.NewAppError(
		middleware.ErrorTypeNotFound,
		http.StatusNotFound,
		message,
	).WithContext(c)
}

// conflictError creates a conflict error with optional details.
func conflictError(c echo.Context, message string, details interface{}) error {
	return middleware.NewAppErrorWithDetails(
		middleware.ErrorTypeConflict,
		http.StatusConflict,
		message,
		details,
	).WithContext(c)
}

// internalError creates an internal server error with context
func internalError(c echo.Context, message string, err error) error {
	return middleware.NewAppError(
		middleware.ErrorTypeInternal,
		http.StatusInternalServerError,
		message,
	).WithContext(c).WithInternal(err)
}

// databaseWriteError maps database constraint violations to useful client errors.
func databaseWriteError(c echo.Context, err error, fallbackMessage string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			field := databaseConflictField(pgErr)
			message := "Resource already exists"
			details := map[string]string{}
			if field != "" {
				message = fmt.Sprintf("%s already exists", humanizeFieldName(field))
				details[field] = message
			}
			return conflictError(c, message, details)
		case "23502":
			field := databaseFieldName(pgErr.ColumnName)
			if field == "" {
				field = "field"
			}
			return validationErrorWithDetails(c, map[string]string{
				field: fmt.Sprintf("%s is required", humanizeFieldName(field)),
			})
		}
	}

	return internalError(c, fallbackMessage, err)
}

func databaseConflictField(pgErr *pgconn.PgError) string {
	switch {
	case strings.Contains(pgErr.ConstraintName, "email"), pgErr.ColumnName == "email":
		return "email"
	default:
		return databaseFieldName(pgErr.ColumnName)
	}
}

func databaseFieldName(column string) string {
	return strings.TrimSpace(strings.ToLower(column))
}

func humanizeFieldName(field string) string {
	field = strings.ReplaceAll(field, "_", " ")
	if field == "" {
		return "Field"
	}
	return strings.ToUpper(field[:1]) + field[1:]
}

// parseIDParam parses and validates an ID parameter from the URL
func parseIDParam(c echo.Context) (int64, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, middleware.NewAppError(
			middleware.ErrorTypeValidation,
			http.StatusBadRequest,
			"Invalid ID format",
		).WithContext(c).WithInternal(err)
	}
	return id, nil
}

// logAndReturnError logs an error and returns an app error
func logAndReturnError(c echo.Context, operation string, err error, statusCode int, userMessage string) error {
	slog.Error("Operation failed",
		"operation", operation,
		"error", err,
		"request_id", c.Response().Header().Get(echo.HeaderXRequestID))

	return middleware.NewAppError(
		middleware.ErrorTypeInternal,
		statusCode,
		userMessage,
	).WithContext(c).WithInternal(err)
}

// stringPtr returns a pointer to string if not empty, nil otherwise
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
