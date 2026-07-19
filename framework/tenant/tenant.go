package tenant

import (
	"context"
	"net/http"

	"github.com/balla-achila/mamba-framework/framework/database"
	"github.com/balla-achila/mamba-framework/framework/logger"
)

// Tenant defines the data structure for a tenant in the system.
type Tenant struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

// Manager handles tenant-related operations and HTTP middleware.
// It is intentionally stateless to ensure safe concurrent usage across HTTP goroutines.
type Manager struct {
	db     database.DB
	logger logger.Logger
}

// contextKey is an unexported empty struct to prevent context key collisions 
// with other packages. Empty structs consume 0 bytes of allocation.
type contextKey struct{}

var tenantContextKey contextKey

// New initializes and returns a new tenant Manager.
func New(db database.DB, log logger.Logger) *Manager {
	return &Manager{
		db:     db,
		logger: log,
	}
}

// Middleware extracts the tenant ID from the headers or query parameters,
// validates it (or falls back to a default), and injects it into the request context.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract tenant ID from header
		tenantID := r.Header.Get("X-Tenant-ID")
		
		// 2. Fallback to query parameter if header is missing
		if tenantID == "" {
			tenantID = r.URL.Query().Get("tenant")
		}

		// 3. Fallback to default tenant if still empty
		if tenantID == "" {
			tenantID = "default"
		}

		// 4. Fetch tenant information. 
		// In production, swap this out to query your database: 
		// tenant, err := m.GetTenant(r.Context(), tenantID)
		tenant := &Tenant{
			ID:          tenantID,
			Name:        "Tenant " + tenantID,
			Code:        tenantID,
			Description: "Default tenant",
			IsActive:    true,
		}

		// 5. Inject the tenant into the request context and pass it down the chain
		ctx := context.WithValue(r.Context(), tenantContextKey, tenant)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetTenant fetches a specific tenant's details. 
func (m *Manager) GetTenant(ctx context.Context, id string) (*Tenant, error) {
	// TODO: Implement actual database fetching logic using m.db
	return &Tenant{
		ID:          id,
		Name:        "Tenant " + id,
		Code:        id,
		Description: "Default tenant",
		IsActive:    true,
	}, nil
}

// GetTenantFromContext safely retrieves the tenant pointer from a given context.
// Use this inside your downstream HTTP handlers to know which tenant is making the request.
func GetTenantFromContext(ctx context.Context) (*Tenant, bool) {
	tenant, ok := ctx.Value(tenantContextKey).(*Tenant)
	return tenant, ok
}