package router

import (
    "fmt"
    "net/http"
    "path/filepath"
    "strings"
)

// PageRouter automaps URLs to page handlers
type PageRouter struct {
    router     *Router
    pagesPath  string
    extension  string
}

func NewPageRouter(router *Router, pagesPath string) *PageRouter {
    return &PageRouter{
        router:    router,
        pagesPath: pagesPath,
        extension: ".gohtml",
    }
}

// AutoMap automatically maps routes to pages
func (pr *PageRouter) AutoMap() {
    // Map common CRUD patterns
    pr.router.Get("/", pr.pageHandler("index"))
    pr.router.Get("/:page", pr.pageHandler(":page"))
    pr.router.Get("/:page/:action", pr.pageHandler(":page/:action"))
    
    // Map POST actions
    pr.router.Post("/:page/save", pr.postHandler(":page", "save"))
    pr.router.Post("/:page/delete", pr.postHandler(":page", "delete"))
    pr.router.Post("/:page/update", pr.postHandler(":page", "update"))
    pr.router.Post("/:page/:action", pr.postHandler(":page", ":action"))
}

func (pr *PageRouter) pageHandler(pattern string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Extract page name from URL
        path := r.URL.Path
        if path == "/" {
            path = "/index"
        }
        
        page := strings.TrimPrefix(path, "/")
        if page == "" {
            page = "index"
        }
        
        // Find page file
        pagePath := filepath.Join(pr.pagesPath, page+pr.extension)
        
        // If page doesn't exist, try to find a matching file
        // This allows /employees to map to /pages/employees/list.gohtml
        
        // TODO: Implement page detection logic
        // For now, pass through to the router
        pr.router.ServeHTTP(w, r)
    }
}

func (pr *PageRouter) postHandler(pagePattern, actionPattern string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Handle POST actions
        pr.router.ServeHTTP(w, r)
    }
}

// GetPagePath returns the full path to a page template
func (pr *PageRouter) GetPagePath(page string) string {
    return filepath.Join(pr.pagesPath, page+pr.extension)
}

// PageExists checks if a page template exists
func (pr *PageRouter) PageExists(page string) bool {
    return true // Simplified for now
}