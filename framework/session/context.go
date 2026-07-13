package session

import (
    "context"
)

type contextKey string

const sessionContextKey contextKey = "session"

// ContextWithSession adds a session to the context
func ContextWithSession(ctx context.Context, session *Session) context.Context {
    return context.WithValue(ctx, sessionContextKey, session)
}

// FromContext retrieves a session from the context
func FromContext(ctx context.Context) *Session {
    if session, ok := ctx.Value(sessionContextKey).(*Session); ok {
        return session
    }
    return nil
}
