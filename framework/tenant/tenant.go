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
        tenantID := r.Header.Get("X-Tenant-ID")
        if tenantID == "" {
            tenantID = r.URL.Query().Get("tenant")
        }

        if tenantID == "" {
            tenantID = "default"
        }

        tenant, err := m.GetTenant(r.Context(), tenantID)
        if err != nil {
            m.logger.Error("Failed to load tenant", "tenant_id", tenantID, "error", err)
            http.Error(w, "Tenant not found", http.StatusNotFound)
            return
        }

        if !tenant.IsActive {
            http.Error(w, "Tenant is inactive", http.StatusForbidden)
            return
        }

        ctx := context.WithValue(r.Context(), tenantContextKey, tenant)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func (m *Manager) GetTenant(ctx context.Context, id string) (*Tenant, error) {
    var tenant Tenant
    err := m.db.QueryRow(ctx, "SELECT id, name, code, description, is_active FROM tenants WHERE id = $1", id).
        Scan(&tenant.ID, &tenant.Name, &tenant.Code, &tenant.Description, &tenant.IsActive)

    if err != nil {
        return nil, err
    }

    return &tenant, nil
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