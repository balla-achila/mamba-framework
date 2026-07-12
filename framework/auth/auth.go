package auth

import (
    "context"
    "fmt"
    "time"

    "golang.org/x/crypto/bcrypt"
    "github.com/mamba-framework/mamba/framework/database"
    "github.com/mamba-framework/mamba/framework/logger"
    "github.com/mamba-framework/mamba/framework/session"
)

type Auth struct {
    db     database.DB
    logger logger.Logger
}

type User struct {
    ID        int64     `db:"id"`
    Username  string    `db:"username"`
    Email     string    `db:"email"`
    Password  string    `db:"password"`
    TenantID  string    `db:"tenant_id"`
    Role      string    `db:"role"`
    FirstName string    `db:"first_name"`
    LastName  string    `db:"last_name"`
    IsActive  bool      `db:"is_active"`
    CreatedAt time.Time `db:"created_at"`
    UpdatedAt time.Time `db:"updated_at"`
}

type LoginAttempt struct {
    Username    string    `db:"username"`
    IPAddress   string    `db:"ip_address"`
    Success     bool      `db:"success"`
    AttemptTime time.Time `db:"attempt_time"`
}

type PasswordReset struct {
    ID        int64     `db:"id"`
    UserID    int64     `db:"user_id"`
    Token     string    `db:"token"`
    ExpiresAt time.Time `db:"expires_at"`
    Used      bool      `db:"used"`
}

func New(db database.DB, log logger.Logger) *Auth {
    return &Auth{
        db:     db,
        logger: log,
    }
}

func (a *Auth) Login(ctx context.Context, username, password, ipAddress string) (*User, error) {
    var user User
    err := a.db.QueryRow(ctx, "SELECT * FROM users WHERE username = $1 AND is_active = true", username).
        Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.TenantID,
            &user.Role, &user.FirstName, &user.LastName, &user.IsActive,
            &user.CreatedAt, &user.UpdatedAt)
    
    if err != nil {
        a.logLoginAttempt(ctx, username, ipAddress, false)
        return nil, fmt.Errorf("invalid username or password")
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
        a.logLoginAttempt(ctx, username, ipAddress, false)
        return nil, fmt.Errorf("invalid username or password")
    }

    a.logLoginAttempt(ctx, username, ipAddress, true)
    return &user, nil
}

func (a *Auth) Register(ctx context.Context, user *User, password string) error {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return fmt.Errorf("failed to hash password: %w", err)
    }

    user.Password = string(hashedPassword)
    user.CreatedAt = time.Now()
    user.UpdatedAt = time.Now()

    id, err := a.db.Insert(ctx, "users", map[string]interface{}{
        "username":   user.Username,
        "email":      user.Email,
        "password":   user.Password,
        "tenant_id":  user.TenantID,
        "role":       user.Role,
        "first_name": user.FirstName,
        "last_name":  user.LastName,
        "is_active":  user.IsActive,
        "created_at": user.CreatedAt,
        "updated_at": user.UpdatedAt,
    })

    if err != nil {
        return fmt.Errorf("failed to create user: %w", err)
    }

    user.ID = id
    return nil
}

func (a *Auth) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
    var currentPassword string
    err := a.db.QueryRow(ctx, "SELECT password FROM users WHERE id = $1", userID).Scan(&currentPassword)
    if err != nil {
        return fmt.Errorf("user not found: %w", err)
    }

    if err := bcrypt.CompareHashAndPassword([]byte(currentPassword), []byte(oldPassword)); err != nil {
        return fmt.Errorf("invalid current password")
    }

    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
    if err != nil {
        return fmt.Errorf("failed to hash password: %w", err)
    }

    _, err = a.db.Update(ctx, "users", map[string]interface{}{
        "password": string(hashedPassword),
        "updated_at": time.Now(),
    }, "id = $1", userID)

    return err
}

func (a *Auth) ResetPassword(ctx context.Context, email string) (string, error) {
    var userID int64
    err := a.db.QueryRow(ctx, "SELECT id FROM users WHERE email = $1 AND is_active = true", email).Scan(&userID)
    if err != nil {
        return "", fmt.Errorf("email not found")
    }

    token := session.GenerateCSRF()
    expiresAt := time.Now().Add(24 * time.Hour)

    _, err = a.db.Insert(ctx, "password_resets", map[string]interface{}{
        "user_id":     userID,
        "token":       token,
        "expires_at":  expiresAt,
        "used":        false,
    })

    if err != nil {
        return "", fmt.Errorf("failed to create reset token: %w", err)
    }

    return token, nil
}

func (a *Auth) ConfirmReset(ctx context.Context, token, newPassword string) error {
    var reset PasswordReset
    err := a.db.QueryRow(ctx, "SELECT * FROM password_resets WHERE token = $1 AND used = false AND expires_at > NOW()", token).
        Scan(&reset.ID, &reset.UserID, &reset.Token, &reset.ExpiresAt, &reset.Used)
    
    if err != nil {
        return fmt.Errorf("invalid or expired token")
    }

    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
    if err != nil {
        return fmt.Errorf("failed to hash password: %w", err)
    }

    tx, err := a.db.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    _, err = tx.Exec(ctx, "UPDATE users SET password = $1, updated_at = NOW() WHERE id = $2", 
        string(hashedPassword), reset.UserID)
    if err != nil {
        return err
    }

    _, err = tx.Exec(ctx, "UPDATE password_resets SET used = true WHERE id = $1", reset.ID)
    if err != nil {
        return err
    }

    return tx.Commit(ctx)
}

func (a *Auth) Logout(ctx context.Context, userID int64) error {
    _, err := a.db.Update(ctx, "users", map[string]interface{}{
        "updated_at": time.Now(),
    }, "id = $1", userID)
    return err
}

func (a *Auth) logLoginAttempt(ctx context.Context, username, ipAddress string, success bool) {
    _, err := a.db.Insert(ctx, "login_attempts", map[string]interface{}{
        "username":    username,
        "ip_address":  ipAddress,
        "success":     success,
        "attempt_time": time.Now(),
    })
    if err != nil {
        a.logger.Error("Failed to log login attempt", "error", err)
    }
}

func (a *Auth) GetUserByID(ctx context.Context, userID int64) (*User, error) {
    var user User
    err := a.db.QueryRow(ctx, "SELECT * FROM users WHERE id = $1", userID).
        Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.TenantID,
            &user.Role, &user.FirstName, &user.LastName, &user.IsActive,
            &user.CreatedAt, &user.UpdatedAt)
    
    if err != nil {
        return nil, err
    }
    return &user, nil
}

func (a *Auth) GetUserByEmail(ctx context.Context, email string) (*User, error) {
    var user User
    err := a.db.QueryRow(ctx, "SELECT * FROM users WHERE email = $1", email).
        Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.TenantID,
            &user.Role, &user.FirstName, &user.LastName, &user.IsActive,
            &user.CreatedAt, &user.UpdatedAt)
    
    if err != nil {
        return nil, err
    }
    return &user, nil
}
