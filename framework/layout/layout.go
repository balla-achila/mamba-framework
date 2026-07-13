package layout

import (
    "fmt"
    "html/template"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "sync"
)

type LayoutEngine struct {
    templates   map[string]*template.Template
    basePath    string
    layoutPath  string
    partialPath string
    mu          sync.RWMutex
    funcMap     template.FuncMap
    dataFuncs   map[string]interface{}
}

type LayoutData struct {
    Title   string
    Content template.HTML
    Data    map[string]interface{}
    Section map[string]template.HTML
    Layout  string
    Code    map[string]interface{}
    Request *http.Request
}

// WebMatrix-style PageData
type PageData struct {
    Request *http.Request
    Data    map[string]interface{}
    User    interface{}
    Session interface{}
    DB      interface{}
    IsPost  bool
    IsGet   bool
    Form    map[string]string
    Query   map[string]string
    UrlData []string
}

func New(basePath, layoutPath, partialPath string) *LayoutEngine {
    engine := &LayoutEngine{
        templates:   make(map[string]*template.Template),
        basePath:    basePath,
        layoutPath:  layoutPath,
        partialPath: partialPath,
        funcMap:     make(template.FuncMap),
        dataFuncs:   make(map[string]interface{}),
    }

    // WebMatrix-style @ syntax helpers
    engine.funcMap["Layout"] = func(layout string) string { return layout }
    engine.funcMap["Partial"] = engine.partialFunc
    engine.funcMap["Include"] = engine.includeFunc
    
    // WebMatrix @IsPost, @IsGet, etc.
    engine.funcMap["IsPost"] = engine.isPostFunc
    engine.funcMap["IsGet"] = engine.isGetFunc
    engine.funcMap["IsPut"] = engine.isPutFunc
    engine.funcMap["IsDelete"] = engine.isDeleteFunc
    
    // WebMatrix @Request
    engine.funcMap["Request"] = engine.requestFunc
    engine.funcMap["Form"] = engine.formFunc
    engine.funcMap["QueryString"] = engine.queryStringFunc
    engine.funcMap["UrlData"] = engine.urlDataFunc
    engine.funcMap["Href"] = engine.hrefFunc
    
    // WebMatrix @DB
    engine.funcMap["DB"] = engine.dbFunc
    engine.funcMap["Query"] = engine.queryFunc
    engine.funcMap["QueryValue"] = engine.queryValueFunc
    engine.funcMap["Execute"] = engine.executeFunc
    
    // WebMatrix @Session
    engine.funcMap["Session"] = engine.sessionFunc
    engine.funcMap["SessionValue"] = engine.sessionValueFunc
    
    // WebMatrix @User
    engine.funcMap["User"] = engine.userFunc
    engine.funcMap["UserId"] = engine.userIdFunc
    engine.funcMap["UserRole"] = engine.userRoleFunc
    
    // WebMatrix @Html helpers
    engine.funcMap["Html"] = engine.htmlFunc
    engine.funcMap["Raw"] = engine.rawFunc
    
    // WebMatrix @Validation
    engine.funcMap["Validation"] = engine.validationFunc
    engine.funcMap["ValidationSummary"] = engine.validationSummaryFunc
    
    // WebMatrix @AntiForgery
    engine.funcMap["AntiForgery"] = engine.antiForgeryFunc
    
    // WebMatrix @Url
    engine.funcMap["Url"] = engine.urlFunc
    
    // WebMatrix @Redirect
    engine.funcMap["Redirect"] = engine.redirectFunc
    
    return engine
}

// WebMatrix @IsPost
func (le *LayoutEngine) isPostFunc(data interface{}) bool {
    if layoutData, ok := data.(*LayoutData); ok && layoutData.Request != nil {
        return layoutData.Request.Method == "POST"
    }
    return false
}

// WebMatrix @IsGet
func (le *LayoutEngine) isGetFunc(data interface{}) bool {
    if layoutData, ok := data.(*LayoutData); ok && layoutData.Request != nil {
        return layoutData.Request.Method == "GET"
    }
    return false
}

// WebMatrix @IsPut
func (le *LayoutEngine) isPutFunc(data interface{}) bool {
    if layoutData, ok := data.(*LayoutData); ok && layoutData.Request != nil {
        return layoutData.Request.Method == "PUT"
    }
    return false
}

// WebMatrix @IsDelete
func (le *LayoutEngine) isDeleteFunc(data interface{}) bool {
    if layoutData, ok := data.(*LayoutData); ok && layoutData.Request != nil {
        return layoutData.Request.Method == "DELETE"
    }
    return false
}

// WebMatrix @Request
func (le *LayoutEngine) requestFunc(data interface{}) *http.Request {
    if layoutData, ok := data.(*LayoutData); ok && layoutData.Request != nil {
        return layoutData.Request
    }
    return nil
}

// WebMatrix @Form["key"]
func (le *LayoutEngine) formFunc(key string, data interface{}) string {
    if layoutData, ok := data.(*LayoutData); ok && layoutData.Request != nil {
        return layoutData.Request.FormValue(key)
    }
    return ""
}

// WebMatrix @QueryString["key"]
func (le *LayoutEngine) queryStringFunc(key string, data interface{}) string {
    if layoutData, ok := data.(*LayoutData); ok && layoutData.Request != nil {
        return layoutData.Request.URL.Query().Get(key)
    }
    return ""
}

// WebMatrix @UrlData[index]
func (le *LayoutEngine) urlDataFunc(index int, data interface{}) string {
    if layoutData, ok := data.(*LayoutData); ok && layoutData.Request != nil {
        parts := strings.Split(strings.Trim(layoutData.Request.URL.Path, "/"), "/")
        if index < len(parts) {
            return parts[index]
        }
    }
    return ""
}

// WebMatrix @Href("path")
func (le *LayoutEngine) hrefFunc(path string, data interface{}) string {
    return path
}

// WebMatrix @DB
func (le *LayoutEngine) dbFunc(data interface{}) interface{} {
    return nil
}

// WebMatrix @Query("SELECT * FROM users")
func (le *LayoutEngine) queryFunc(query string, args ...interface{}) []map[string]interface{} {
    // In production, this would execute the query
    return []map[string]interface{}{}
}

// WebMatrix @QueryValue("SELECT COUNT(*) FROM users")
func (le *LayoutEngine) queryValueFunc(query string, args ...interface{}) interface{} {
    // In production, this would return a single value
    return nil
}

// WebMatrix @Execute("UPDATE users SET name = @p0", "John")
func (le *LayoutEngine) executeFunc(query string, args ...interface{}) int64 {
    // In production, this would execute the query
    return 0
}

// WebMatrix @Session["key"]
func (le *LayoutEngine) sessionFunc(key string, data interface{}) interface{} {
    return nil
}

// WebMatrix @SessionValue("key")
func (le *LayoutEngine) sessionValueFunc(key string, data interface{}) interface{} {
    return nil
}

// WebMatrix @User
func (le *LayoutEngine) userFunc(data interface{}) interface{} {
    if layoutData, ok := data.(*LayoutData); ok {
        if user, ok := layoutData.Data["CurrentUser"]; ok {
            return user
        }
    }
    return nil
}

// WebMatrix @UserId
func (le *LayoutEngine) userIdFunc(data interface{}) int64 {
    if layoutData, ok := data.(*LayoutData); ok {
        if user, ok := layoutData.Data["CurrentUser"].(map[string]interface{}); ok {
            if id, ok := user["ID"].(int64); ok {
                return id
            }
        }
    }
    return 0
}

// WebMatrix @UserRole
func (le *LayoutEngine) userRoleFunc(data interface{}) string {
    if layoutData, ok := data.(*LayoutData); ok {
        if user, ok := layoutData.Data["CurrentUser"].(map[string]interface{}); ok {
            if role, ok := user["Role"].(string); ok {
                return role
            }
        }
    }
    return ""
}

// WebMatrix @Html helper
func (le *LayoutEngine) htmlFunc(data interface{}) interface{} {
    return nil
}

// WebMatrix @Raw HTML
func (le *LayoutEngine) rawFunc(html string) template.HTML {
    return template.HTML(html)
}

// WebMatrix @Validation
func (le *LayoutEngine) validationFunc(field string, data interface{}) string {
    return ""
}

// WebMatrix @ValidationSummary
func (le *LayoutEngine) validationSummaryFunc(data interface{}) string {
    return ""
}

// WebMatrix @AntiForgery
func (le *LayoutEngine) antiForgeryFunc(data interface{}) template.HTML {
    return template.HTML(`<input type="hidden" name="__RequestVerificationToken" value="` + generateToken() + `">`)
}

// WebMatrix @Url
func (le *LayoutEngine) urlFunc(path string, data interface{}) string {
    return path
}

// WebMatrix @Redirect
func (le *LayoutEngine) redirectFunc(url string, data interface{}) string {
    return url
}

func generateToken() string {
    return "sample-token"
}

func (le *LayoutEngine) AddFunc(name string, fn interface{}) {
    le.mu.Lock()
    defer le.mu.Unlock()
    le.funcMap[name] = fn
}

func (le *LayoutEngine) Render(w io.Writer, name string, data interface{}) error {
    pageTmpl, err := le.getTemplate(name)
    if err != nil {
        return err
    }

    layoutName := "default"
    if layoutData, ok := data.(*LayoutData); ok && layoutData.Layout != "" {
        layoutName = layoutData.Layout
    }

    layoutTmpl, err := le.getLayout(layoutName)
    if err != nil {
        return err
    }

    var contentBuf strings.Builder
    if err := pageTmpl.ExecuteTemplate(&contentBuf, "content", data); err != nil {
        return fmt.Errorf("failed to execute page content: %w", err)
    }

    if layoutData, ok := data.(*LayoutData); ok {
        layoutData.Content = template.HTML(contentBuf.String())
    }

    return layoutTmpl.Execute(w, data)
}

func (le *LayoutEngine) RenderPartial(w io.Writer, name string, data interface{}) error {
    tmpl, err := le.getPartial(name)
    if err != nil {
        return err
    }
    return tmpl.Execute(w, data)
}

func (le *LayoutEngine) getTemplate(name string) (*template.Template, error) {
    le.mu.RLock()
    tmpl, exists := le.templates[name]
    le.mu.RUnlock()

    if exists {
        return tmpl, nil
    }

    return le.loadTemplate(name)
}

func (le *LayoutEngine) loadTemplate(name string) (*template.Template, error) {
    le.mu.Lock()
    defer le.mu.Unlock()

    if tmpl, exists := le.templates[name]; exists {
        return tmpl, nil
    }

    pagePath := filepath.Join(le.basePath, "pages", name+".gohtml")
    
    if _, err := os.Stat(pagePath); os.IsNotExist(err) {
        pagePath = filepath.Join(le.basePath, name+".gohtml")
    }

    tmpl := template.New(name).Funcs(le.funcMap)

    tmpl, err := tmpl.ParseFiles(pagePath)
    if err != nil {
        return nil, fmt.Errorf("failed to parse page %s: %w", pagePath, err)
    }

    partialPaths, err := filepath.Glob(filepath.Join(le.partialPath, "*.gohtml"))
    if err == nil {
        for _, partialPath := range partialPaths {
            _, err = tmpl.ParseFiles(partialPath)
            if err != nil {
                return nil, fmt.Errorf("failed to parse partial %s: %w", partialPath, err)
            }
        }
    }

    le.templates[name] = tmpl
    return tmpl, nil
}

func (le *LayoutEngine) getLayout(name string) (*template.Template, error) {
    layoutPath := filepath.Join(le.layoutPath, name+".gohtml")
    tmpl := template.New(name).Funcs(le.funcMap)
    tmpl, err := tmpl.ParseFiles(layoutPath)
    if err != nil {
        return nil, fmt.Errorf("failed to parse layout %s: %w", layoutPath, err)
    }
    return tmpl, nil
}

func (le *LayoutEngine) getPartial(name string) (*template.Template, error) {
    partialPath := filepath.Join(le.partialPath, name+".gohtml")
    return template.ParseFiles(partialPath)
}

func (le *LayoutEngine) partialFunc(name string, data interface{}) template.HTML {
    var buf strings.Builder
    if err := le.RenderPartial(&buf, name, data); err != nil {
        return template.HTML(fmt.Sprintf("<!-- Error rendering partial %s: %v -->", name, err))
    }
    return template.HTML(buf.String())
}

func (le *LayoutEngine) includeFunc(name string, data interface{}) template.HTML {
    var buf strings.Builder
    if err := le.Render(&buf, name, data); err != nil {
        return template.HTML(fmt.Sprintf("<!-- Error including %s: %v -->", name, err))
    }
    return template.HTML(buf.String())
}

func (le *LayoutEngine) ClearCache() {
    le.mu.Lock()
    defer le.mu.Unlock()
    le.templates = make(map[string]*template.Template)
}

func NewLayoutData(title string) *LayoutData {
    return &LayoutData{
        Title:   title,
        Data:    make(map[string]interface{}),
        Section: make(map[string]template.HTML),
        Code:    make(map[string]interface{}),
    }
}

func (ld *LayoutData) Set(key string, value interface{}) {
    ld.Data[key] = value
}

func (ld *LayoutData) Get(key string) interface{} {
    return ld.Data[key]
}
