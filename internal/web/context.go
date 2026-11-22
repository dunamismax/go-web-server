package web

import (
	"context"

	"github.com/dunamismax/go-web-server/internal/store"
)

type contextKey int

const (
	requestIDKey contextKey = iota
	userKey
)

func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// currentUser returns the signed-in user, or nil.
func currentUser(ctx context.Context) *store.User {
	user, _ := ctx.Value(userKey).(*store.User)
	return user
}
