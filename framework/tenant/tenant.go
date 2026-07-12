package tenant

import (
    "context"
    "net/http"

    "github.com/mamba-framework/mamba/framework/database"
    "github.com/mamba-framework/mamba/framework/logger"
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
        db:      db,
        logger:  log,
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
            // Could load default tenant from config
            tenantID = "default"
        }

        // Get tenant from database
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

        // Add tenant to context
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

// Tenant-aware database queries
func (m *Manager) AddTenantFilter(query string) string {
    // Add tenant_id condition to WHERE clause
    if !strings.Contains(query, "WHERE") {
        return query + " WHERE tenant_id = $1"
    }
    return query + " AND tenant_id = $1"
}

// Helper for tenant-aware queries
func (m *Manager) BuildTenantQuery(baseQuery string, params ...interface{}) (string, []interface{}) {
    // Add tenant_id condition
    tenantID := "$" + fmt.Sprintf("%d", len(params)+1)
    if strings.Contains(baseQuery, "WHERE") {
        baseQuery += " AND tenant_id = " + tenantID
    } else {
        baseQuery += " WHERE tenant_id = " + tenantID
    }
    
    // Add current tenant ID if available
    if m.current != nil {
        params = append(params, m.current.ID)
    }
    
    return baseQuery, params
}