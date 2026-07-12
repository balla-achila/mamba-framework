package auth

import (
    "context"
    "net/http"

    "github.com/balla-achila/mamba-framework/framework/session"
)

type contextKey string

const userContextKey contextKey = "user"

func (a *Auth) RequireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        sess := session.FromContext(r.Context())
        if sess == nil {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }

        userID := sess.GetUserID()
        if userID == 0 {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }

        user, err := a.GetUserByID(r.Context(), userID)
        if err != nil {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }

        ctx := context.WithValue(r.Context(), userContextKey, user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func (a *Auth) RequireRole(roles ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user, ok := r.Context().Value(userContextKey).(*User)
            if !ok {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            for _, role := range roles {
                if user.Role == role {
                    next.ServeHTTP(w, r)
                    return
                }
            }

            http.Error(w, "Forbidden", http.StatusForbidden)
        })
    }
}

func GetUserFromContext(ctx context.Context) *User {
    if user, ok := ctx.Value(userContextKey).(*User); ok {
        return user
    }
    return nil
}
