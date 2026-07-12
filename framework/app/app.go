package app

import (
    "context"
    "net/http"
    "path/filepath"

    "github.com/balla-achila/mamba-framework/framework/auth"
    "github.com/balla-achila/mamba-framework/framework/config"
    "github.com/balla-achila/mamba-framework/framework/database"
    "github.com/balla-achila/mamba-framework/framework/html"
    "github.com/balla-achila/mamba-framework/framework/layout"
    "github.com/balla-achila/mamba-framework/framework/logger"
    "github.com/balla-achila/mamba-framework/framework/router"
    "github.com/balla-achila/mamba-framework/framework/security"
    "github.com/balla-achila/mamba-framework/framework/session"
    "github.com/balla-achila/mamba-framework/framework/tenant"
)

type App struct {
    Config       *config.Config
    Logger       logger.Logger
    DB           database.DB
    Router       *router.Router
    Layout       *layout.LayoutEngine
    HTML         *html.HTMLHelper
    Auth         *auth.Auth
    Session      *session.Manager
    Tenant       *tenant.Manager
    Security     *security.Security
    pagesPath    string
    layoutsPath  string
    partialsPath string
}

type AppContext struct {
    App      *App
    Request  *http.Request
    Response http.ResponseWriter
    Session  *session.Session
    User     *auth.User
    Tenant   *tenant.Tenant
}

func New(cfg *config.Config, log logger.Logger, db database.DB) *App {
    // Initialize session manager
    sessionCfg := &session.Config{
        SecretKey: cfg.Session.SecretKey,
        Name:      cfg.Session.Name,
        MaxAge:    cfg.Session.MaxAge,
        Secure:    cfg.Session.Secure,
        HttpOnly:  cfg.Session.HttpOnly,
        SameSite:  cfg.Session.SameSite,
    }
    sessionMgr := session.New(sessionCfg)

    // Initialize HTML helper
    htmlHelper := html.NewHelper("")

    // Initialize layout engine
    layoutEngine := layout.New(
        cfg.Server.TemplatesPath,
        filepath.Join(cfg.Server.TemplatesPath, "layouts"),
        filepath.Join(cfg.Server.TemplatesPath, "partials"),
    )

    // Initialize tenant manager
    tenantMgr := tenant.New(db, log)

    // Initialize auth
    authMgr := auth.New(db, log)

    // Initialize security
    securityMgr := security.New(cfg, log)

    // Initialize router
    r := router.New()

    app := &App{
        Config:       cfg,
        Logger:       log,
        DB:           db,
        Router:       r,
        Layout:       layoutEngine,
        HTML:         htmlHelper,
        Auth:         authMgr,
        Session:      sessionMgr,
        Tenant:       tenantMgr,
        Security:     securityMgr,
        pagesPath:    filepath.Join(cfg.Server.TemplatesPath, "pages"),
        layoutsPath:  filepath.Join(cfg.Server.TemplatesPath, "layouts"),
        partialsPath: filepath.Join(cfg.Server.TemplatesPath, "partials"),
    }

    // Setup middlewares
    app.setupMiddlewares()

    return app
}

func (a *App) setupMiddlewares() {
    a.Router.Use(a.Session.Middleware)
    if a.Config.Tenant.Enabled {
        a.Router.Use(a.Tenant.Middleware)
    }
    a.Router.Use(a.Security.Middleware)
    a.Router.Use(a.Security.CSRFMiddleware)
    a.Router.Use(a.Security.RateLimitMiddleware)
}

func (a *App) HandlePage(page string, handler func(ctx *AppContext) error) {
    a.Router.Get("/"+page, a.pageHandler(page, handler))
    a.Router.Post("/"+page+"/save", a.pageHandler(page+"/save", handler))
    a.Router.Post("/"+page+"/delete", a.pageHandler(page+"/delete", handler))
    a.Router.Post("/"+page+"/update", a.pageHandler(page+"/update", handler))
}

func (a *App) Handle(path, method string, handler func(ctx *AppContext) error) {
    a.Router.AddRoute(method, path, a.handlerWrapper(handler))
}

func (a *App) pageHandler(page string, handler func(ctx *AppContext) error) http.HandlerFunc {
    return a.handlerWrapper(handler)
}

func (a *App) handlerWrapper(handler func(ctx *AppContext) error) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        sess := session.FromContext(r.Context())
        user := auth.GetUserFromContext(r.Context())
        tenant := tenant.GetTenantFromContext(r.Context())

        ctx := &AppContext{
            App:      a,
            Request:  r,
            Response: w,
            Session:  sess,
            User:     user,
            Tenant:   tenant,
        }

        if err := handler(ctx); err != nil {
            a.Logger.Error("Handler error", "error", err)
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }
    }
}

func (a *App) Render(ctx *AppContext, name string, data interface{}) error {
    return a.Layout.Render(ctx.Response, name, data)
}

func (a *App) RenderPartial(ctx *AppContext, name string, data interface{}) error {
    return a.Layout.RenderPartial(ctx.Response, name, data)
}

func (a *App) Redirect(ctx *AppContext, url string, code int) {
    http.Redirect(ctx.Response, ctx.Request, url, code)
}

func (a *App) JSON(ctx *AppContext, data interface{}) error {
    ctx.Response.Header().Set("Content-Type", "application/json")
    return nil
}

func (c *AppContext) Param(name string) string {
    return router.GetRouteParam(c.Request.Context(), name)
}

func (c *AppContext) Params() map[string]string {
    return router.GetRouteParams(c.Request.Context())
}

func (c *AppContext) FormValue(name string) string {
    return c.Request.FormValue(name)
}

func (c *AppContext) QueryValue(name string) string {
    return c.Request.URL.Query().Get(name)
}

func (c *AppContext) Flash(key string) string {
    if c.Session != nil {
        flashes := c.Session.GetFlashMessages()
        if msg, ok := flashes[key]; ok {
            return msg
        }
    }
    return ""
}

func (c *AppContext) AddFlash(key, message string) {
    if c.Session != nil {
        c.Session.AddFlashMessage(key, message)
        c.Session.Save()
    }
}

func (c *AppContext) IsAuthenticated() bool {
    return c.User != nil && c.User.ID > 0
}

func (c *AppContext) HasRole(role string) bool {
    if c.User == nil {
        return false
    }
    return c.User.Role == role
}
