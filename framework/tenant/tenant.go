package tenant

import (
    "context"
    "net/http"

    "github.com/balla-achila/mamba-framework/framework/database"
    "github.com/balla-achila/mamba-framework/framework/logger"
)

type Tenant struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Code        string `json:"code"`
    Description string `json:"description"`
    IsActive    bool   `json:"is_active"`
}

type Manager struct {
    db      database.DB
    logger  logger.Logger
    current *Tenant
}

type contextKey string

const tenantContextKey contextKey = "tenant"

func New(db database.DB, log logger.Logger) *Manager {
    return &Manager{
        db:     db,
        logger: log,
    }
}

func (m *Manager) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get tenant from header or query parameter
        tenantID := r.Header.Get("X-Tenant-ID")
        if tenantID == "" {
            tenantID = r.URL.Query().Get("tenant")
        }

        // Use default tenant if not specified
        if tenantID == "" {
            tenantID = "default"
        }

        // For demo purposes, create a default tenant
        // In production, you would load from database
        tenant := &Tenant{
            ID:          tenantID,
            Name:        "Tenant " + tenantID,
            Code:        tenantID,
            Description: "Default tenant",
            IsActive:    true,
        }

        // Add tenant to context
        ctx := context.WithValue(r.Context(), tenantContextKey, tenant)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func (m *Manager) GetTenant(ctx context.Context, id string) (*Tenant, error) {
    // In production, load from database
    return &Tenant{
        ID:          id,
        Name:        "Tenant " + id,
        Code:        id,
        Description: "Default tenant",
        IsActive:    true,
    }, nil
}

func (m *Manager) SetCurrent(tenant *Tenant) {
    m.current = tenant
}

func (m *Manager) GetCurrent() *Tenant {
    return m.current
}

func GetTenantFromContext(ctx context.Context) *Tenant {
    if tenant, ok := ctx.Value(tenantContextKey).(*Tenant); ok {
        return tenant
    }
    return nil
}
