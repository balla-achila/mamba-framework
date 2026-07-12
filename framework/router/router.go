package router

import (
    "context"
    "fmt"
    "net/http"
    "path/filepath"
    "regexp"
    "strings"
    "sync"
)

type Route struct {
    Method  string
    Path    string
    Handler http.HandlerFunc
    Regex   *regexp.Regexp
    Params  []string
}

type Router struct {
    routes     []Route
    notFound   http.HandlerFunc
    middlewares []Middleware
    mu         sync.RWMutex
}

type Middleware func(http.Handler) http.Handler

type RouterContext struct {
    Params map[string]string
}

type contextKey string

const routerContextKey contextKey = "router"

func New() *Router {
    return &Router{
        routes:     make([]Route, 0),
        middlewares: make([]Middleware, 0),
        notFound:   http.NotFound,
    }
}

func (r *Router) Use(middleware Middleware) {
    r.middlewares = append(r.middlewares, middleware)
}

func (r *Router) Get(path string, handler http.HandlerFunc) {
    r.addRoute("GET", path, handler)
}

func (r *Router) Post(path string, handler http.HandlerFunc) {
    r.addRoute("POST", path, handler)
}

func (r *Router) Put(path string, handler http.HandlerFunc) {
    r.addRoute("PUT", path, handler)
}

func (r *Router) Delete(path string, handler http.HandlerFunc) {
    r.addRoute("DELETE", path, handler)
}

func (r *Router) Patch(path string, handler http.HandlerFunc) {
    r.addRoute("PATCH", path, handler)
}

func (r *Router) Head(path string, handler http.HandlerFunc) {
    r.addRoute("HEAD", path, handler)
}

func (r *Router) Options(path string, handler http.HandlerFunc) {
    r.addRoute("OPTIONS", path, handler)
}

func (r *Router) Any(path string, handler http.HandlerFunc) {
    r.addRoute("GET", path, handler)
    r.addRoute("POST", path, handler)
    r.addRoute("PUT", path, handler)
    r.addRoute("DELETE", path, handler)
    r.addRoute("PATCH", path, handler)
}

func (r *Router) addRoute(method, path string, handler http.HandlerFunc) {
    r.mu.Lock()
    defer r.mu.Unlock()

    // Convert path parameters to regex
    regex, params := r.pathToRegex(path)
    
    route := Route{
        Method:  method,
        Path:    path,
        Handler: handler,
        Regex:   regex,
        Params:  params,
    }
    
    r.routes = append(r.routes, route)
}

func (r *Router) pathToRegex(path string) (*regexp.Regexp, []string) {
    // Replace :param with regex capture groups
    paramRegex := regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)
    params := paramRegex.FindAllStringSubmatch(path, -1)
    
    paramNames := make([]string, len(params))
    for i, p := range params {
        paramNames[i] = p[1]
    }

    // Convert path to regex pattern
    pattern := paramRegex.ReplaceAllString(path, `([^/]+)`)
    pattern = "^" + pattern + "$"
    
    return regexp.MustCompile(pattern), paramNames
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    // Apply middlewares
    handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
        r.routeHandler(w, req)
    })

    // Apply all middlewares in reverse order
    for i := len(r.middlewares) - 1; i >= 0; i-- {
        handler = r.middlewares[i](handler)
    }

    handler.ServeHTTP(w, req)
}

func (r *Router) routeHandler(w http.ResponseWriter, req *http.Request) {
    path := req.URL.Path
    method := req.Method

    // Remove trailing slash for matching
    if path != "/" && strings.HasSuffix(path, "/") {
        path = path[:len(path)-1]
    }

    r.mu.RLock()
    defer r.mu.RUnlock()

    for _, route := range r.routes {
        if route.Method != method && route.Method != "ANY" {
            continue
        }

        if route.Path == path {
            // Exact match
            ctx := context.WithValue(req.Context(), routerContextKey, &RouterContext{
                Params: make(map[string]string),
            })
            route.Handler(w, req.WithContext(ctx))
            return
        }

        // Regex match
        matches := route.Regex.FindStringSubmatch(path)
        if matches != nil {
            params := make(map[string]string)
            for i, name := range route.Params {
                if i+1 < len(matches) {
                    params[name] = matches[i+1]
                }
            }
            
            ctx := context.WithValue(req.Context(), routerContextKey, &RouterContext{
                Params: params,
            })
            route.Handler(w, req.WithContext(ctx))
            return
        }
    }

    // Handle static files
    if strings.HasPrefix(path, "/static/") {
        http.FileServer(http.Dir(".")).ServeHTTP(w, req)
        return
    }

    // 404 Not Found
    r.notFound(w, req)
}

func (r *Router) NotFound(handler http.HandlerFunc) {
    r.notFound = handler
}

// Helper functions for route parameters
func GetRouteParam(ctx context.Context, name string) string {
    if ctx == nil {
        return ""
    }
    val := ctx.Value(routerContextKey)
    if val == nil {
        return ""
    }
    rc, ok := val.(*RouterContext)
    if !ok {
        return ""
    }
    return rc.Params[name]
}

func GetRouteParams(ctx context.Context) map[string]string {
    if ctx == nil {
        return nil
    }
    val := ctx.Value(routerContextKey)
    if val == nil {
        return nil
    }
    rc, ok := val.(*RouterContext)
    if !ok {
        return nil
    }
    return rc.Params
}

// Group routes
type RouteGroup struct {
    router   *Router
    prefix   string
    middlewares []Middleware
}

func (r *Router) Group(prefix string) *RouteGroup {
    return &RouteGroup{
        router: r,
        prefix: prefix,
    }
}

func (g *RouteGroup) Use(middleware Middleware) {
    g.middlewares = append(g.middlewares, middleware)
}

func (g *RouteGroup) Get(path string, handler http.HandlerFunc) {
    g.router.Get(g.prefix+path, g.wrapHandler(handler))
}

func (g *RouteGroup) Post(path string, handler http.HandlerFunc) {
    g.router.Post(g.prefix+path, g.wrapHandler(handler))
}

func (g *RouteGroup) Put(path string, handler http.HandlerFunc) {
    g.router.Put(g.prefix+path, g.wrapHandler(handler))
}

func (g *RouteGroup) Delete(path string, handler http.HandlerFunc) {
    g.router.Delete(g.prefix+path, g.wrapHandler(handler))
}

func (g *RouteGroup) wrapHandler(handler http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Apply group middlewares
        h := http.HandlerFunc(handler)
        for i := len(g.middlewares) - 1; i >= 0; i-- {
            h = g.middlewares[i](h)
        }
        h.ServeHTTP(w, r)
    }
}