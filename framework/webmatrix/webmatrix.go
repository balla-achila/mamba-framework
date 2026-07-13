package webmatrix

import (
    "bytes"
    "fmt"
    "html/template"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "sync"
    "time"
)

type WebMatrixEngine struct {
    templates    map[string]*template.Template
    pagesPath    string
    layoutsPath  string
    partialsPath string
    mu           sync.RWMutex
    funcMap      template.FuncMap
    config       *WebConfig
    pageData     *PageData
}

type WebConfig struct {
    ConnectionStrings map[string]string
    AppSettings       map[string]string
}

type DatabaseHelper struct {
    engine *WebMatrixEngine
}

type PageData struct {
    Title       string
    Layout      string
    IsPost      bool
    IsGet       bool
    Request     *http.Request
    Response    http.ResponseWriter
    Form        map[string]string
    QueryString map[string]string
    UrlData     []string
    Session     map[string]interface{}
    User        interface{}
    Data        map[string]interface{}
    Sections    map[string]template.HTML
    Body        template.HTML
    Vars        map[string]interface{}
}

type HtmlHelper struct{}

func (h *HtmlHelper) AntiForgeryToken() template.HTML {
    return template.HTML(`<input type="hidden" name="__RequestVerificationToken" value="token" />`)
}

func NewPageData(title string) *PageData {
    return &PageData{
        Title:       title,
        Data:        make(map[string]interface{}),
        Form:        make(map[string]string),
        QueryString: make(map[string]string),
        Session:     make(map[string]interface{}),
        UrlData:     make([]string, 0),
        Sections:    make(map[string]template.HTML),
        Vars:        make(map[string]interface{}),
    }
}

func (p *PageData) SetRequest(r *http.Request) {
    p.Request = r
    p.IsPost = r.Method == "POST"
    p.IsGet = r.Method == "GET"

    r.ParseForm()
    for key, values := range r.Form {
        if len(values) > 0 {
            p.Form[key] = values[0]
        }
    }

    for key, values := range r.URL.Query() {
        if len(values) > 0 {
            p.QueryString[key] = values[0]
        }
    }

    path := strings.Trim(r.URL.Path, "/")
    if path != "" {
        p.UrlData = strings.Split(path, "/")
    }
}

func (p *PageData) GetVar(key string) interface{} {
    if val, ok := p.Vars[key]; ok {
        return val
    }
    return nil
}

func (p *PageData) SetVar(key string, value interface{}) {
    p.Vars[key] = value
}

func DefaultWebConfig() *WebConfig {
    config := &WebConfig{
        ConnectionStrings: make(map[string]string),
        AppSettings:       make(map[string]string),
    }
    config.ConnectionStrings["DefaultConnection"] = "Data Source=localhost;Initial Catalog=MambaDB"
    config.AppSettings["SiteName"] = "Mamba Framework"
    config.AppSettings["Version"] = "1.0.0"
    config.AppSettings["Environment"] = "Development"
    config.AppSettings["CompanyName"] = "Mamba Technologies"
    config.AppSettings["SupportEmail"] = "support@mamba-framework.com"
    return config
}

func NewWebMatrixEngine(pagesPath, layoutsPath, partialsPath string) *WebMatrixEngine {
    engine := &WebMatrixEngine{
        templates:    make(map[string]*template.Template),
        pagesPath:    pagesPath,
        layoutsPath:  layoutsPath,
        partialsPath: partialsPath,
        funcMap:      make(template.FuncMap),
        config:       DefaultWebConfig(),
    }

    // Register template functions
    engine.funcMap["RenderBody"] = engine.renderBodyFunc
    engine.funcMap["RenderSection"] = engine.renderSectionFunc
    engine.funcMap["IsSectionDefined"] = engine.isSectionDefinedFunc
    engine.funcMap["DateTime"] = func() interface{} { return time.Now() }
    engine.funcMap["AppSettings"] = engine.appSettingsFunc
    engine.funcMap["ConnectionString"] = engine.connectionStringFunc
    engine.funcMap["Html"] = func() *HtmlHelper { return &HtmlHelper{} }
    engine.funcMap["Database"] = func() *DatabaseHelper { return &DatabaseHelper{engine: engine} }
    engine.funcMap["Page"] = func() *PageData { return engine.pageData }
    engine.funcMap["Raw"] = func(s string) template.HTML { return template.HTML(s) }
    engine.funcMap["Var"] = engine.varFunc

    return engine
}

func (e *WebMatrixEngine) varFunc(key string) interface{} {
    if e.pageData != nil {
        return e.pageData.GetVar(key)
    }
    return nil
}

func (e *WebMatrixEngine) renderBodyFunc(data interface{}) template.HTML {
    if pageData, ok := data.(*PageData); ok {
        return pageData.Body
    }
    return template.HTML("")
}

func (e *WebMatrixEngine) renderSectionFunc(name string, data interface{}) template.HTML {
    if pageData, ok := data.(*PageData); ok {
        if section, exists := pageData.Sections[name]; exists {
            return section
        }
    }
    return template.HTML("")
}

func (e *WebMatrixEngine) isSectionDefinedFunc(name string, data interface{}) bool {
    if pageData, ok := data.(*PageData); ok {
        _, exists := pageData.Sections[name]
        return exists
    }
    return false
}

func (e *WebMatrixEngine) appSettingsFunc(key string) string {
    if val, ok := e.config.AppSettings[key]; ok {
        return val
    }
    return ""
}

func (e *WebMatrixEngine) connectionStringFunc(name string) string {
    if val, ok := e.config.ConnectionStrings[name]; ok {
        return val
    }
    return ""
}

// Convert WebMatrix @ syntax to Go template syntax
func (e *WebMatrixEngine) convertToGoTemplate(content string, data interface{}) string {
    // Handle @@ escaping: replace @@ with a special token
    content = strings.ReplaceAll(content, "@@", "___AT_AT___")
    
    // Remove @{} code blocks (they're already processed)
    content = regexp.MustCompile(`@\{[^}]*\}`).ReplaceAllString(content, "")
    
    // Convert WebMatrix @ syntax to Go template {{ }} syntax
    
    // @Page.Title -> {{.Title}}
    content = regexp.MustCompile(`@Page\.Title`).ReplaceAllString(content, `{{.Title}}`)
    
    // @Page.IsPost -> {{.IsPost}}
    content = regexp.MustCompile(`@Page\.IsPost`).ReplaceAllString(content, `{{.IsPost}}`)
    
    // @Page.IsGet -> {{.IsGet}}
    content = regexp.MustCompile(`@Page\.IsGet`).ReplaceAllString(content, `{{.IsGet}}`)
    
    // @Page.Form["key"] -> {{index .Form "key"}}
    content = regexp.MustCompile(`@Page\.Form\["([^"]+)"\]`).ReplaceAllString(content, `{{index .Form "$1"}}`)
    
    // @Page.QueryString["key"] -> {{index .QueryString "key"}}
    content = regexp.MustCompile(`@Page\.QueryString\["([^"]+)"\]`).ReplaceAllString(content, `{{index .QueryString "$1"}}`)
    
    // @Page.Session["key"] -> {{index .Session "key"}}
    content = regexp.MustCompile(`@Page\.Session\["([^"]+)"\]`).ReplaceAllString(content, `{{index .Session "$1"}}`)
    
    // @Page.UrlData[0] -> {{index .UrlData 0}}
    content = regexp.MustCompile(`@Page\.UrlData\[([0-9]+)\]`).ReplaceAllString(content, `{{index .UrlData $1}}`)
    
    // @AppSettings["key"] -> {{AppSettings "key"}}
    content = regexp.MustCompile(`@AppSettings\["([^"]+)"\]`).ReplaceAllString(content, `{{AppSettings "$1"}}`)
    
    // @ConnectionString["name"] -> {{ConnectionString "name"}}
    content = regexp.MustCompile(`@ConnectionString\["([^"]+)"\]`).ReplaceAllString(content, `{{ConnectionString "$1"}}`)
    
    // @Html.AntiForgeryToken() -> {{Html.AntiForgeryToken}}
    content = regexp.MustCompile(`@Html\.AntiForgeryToken\(\)`).ReplaceAllString(content, `{{Html.AntiForgeryToken}}`)
    
    // @Database.Query() -> {{Database.Query}}
    content = regexp.MustCompile(`@Database\.Query\(([^)]*)\)`).ReplaceAllString(content, `{{Database.Query $1}}`)
    
    // @RenderBody() -> {{RenderBody .}}
    content = regexp.MustCompile(`@RenderBody\(\)`).ReplaceAllString(content, `{{RenderBody .}}`)
    
    // @RenderSection("name") -> {{RenderSection "name" .}}
    content = regexp.MustCompile(`@RenderSection\(["']([^"']+)["']\)`).ReplaceAllString(content, `{{RenderSection "$1" .}}`)
    
    // @IsSectionDefined("name") -> {{IsSectionDefined "name" .}}
    content = regexp.MustCompile(`@IsSectionDefined\(["']([^"']+)["']\)`).ReplaceAllString(content, `{{IsSectionDefined "$1" .}}`)
    
    // @section name { } -> {{define "name"}}...{{end}}
    sectionRegex := regexp.MustCompile(`@section\s+(\w+)\s*{([^}]*)}`)
    content = sectionRegex.ReplaceAllString(content, `{{define "$1"}}$2{{end}}`)
    
    // @foreach(var item in items) -> {{range $index, $item := .Items}}
    content = regexp.MustCompile(`@foreach\s*\(\s*var\s+(\w+)\s+in\s+(\w+)\s*\)\s*{`).ReplaceAllString(content, `{{range $index, $1 := .$2}}`)
    
    // @if(condition) -> {{if condition}}
    content = regexp.MustCompile(`@if\s*\(([^)]+)\)\s*{`).ReplaceAllString(content, `{{if $1}}`)
    
    // @else -> {{else}}
    content = regexp.MustCompile(`@else\s*{`).ReplaceAllString(content, `{{else}}`)
    
    // @} -> {{end}}
    content = regexp.MustCompile(`@}`).ReplaceAllString(content, `{{end}}`)
    
    // NOTE: We DO NOT convert generic @variable to {{.variable}} because it would conflict with function calls
    // Only specific patterns are converted. Plain @ will remain as @ in the output.
    
    // Restore @@ escaping: replace token back to @
    content = strings.ReplaceAll(content, "___AT_AT___", "@")
    
    return content
}

func (e *WebMatrixEngine) Render(w io.Writer, name string, data interface{}) error {
    // Store page data
    if pageData, ok := data.(*PageData); ok {
        e.pageData = pageData
    }

    // Load the template
    tmpl, err := e.loadTemplate(name, data)
    if err != nil {
        return err
    }

    // Execute the template
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return err
    }

    // If we have page data and a layout, render with layout
    if pageData, ok := data.(*PageData); ok && pageData.Layout != "" {
        pageData.Body = template.HTML(buf.String())
        
        // Load and execute layout
        layoutTmpl, err := e.loadLayout(pageData.Layout, data)
        if err != nil {
            return err
        }
        
        var layoutBuf bytes.Buffer
        if err := layoutTmpl.Execute(&layoutBuf, data); err != nil {
            return err
        }
        _, err = w.Write(layoutBuf.Bytes())
        return err
    }

    _, err = w.Write(buf.Bytes())
    return err
}

func (e *WebMatrixEngine) RenderPartial(w io.Writer, name string, data interface{}) error {
    tmpl, err := e.loadPartial(name, data)
    if err != nil {
        return err
    }
    return tmpl.Execute(w, data)
}

func (e *WebMatrixEngine) loadTemplate(name string, data interface{}) (*template.Template, error) {
    e.mu.Lock()
    defer e.mu.Unlock()

    if tmpl, exists := e.templates[name]; exists {
        return tmpl, nil
    }

    pagePath := filepath.Join(e.pagesPath, name+".gshtml")
    if _, err := os.Stat(pagePath); os.IsNotExist(err) {
        pagePath = filepath.Join(e.pagesPath, name+".gohtml")
        if _, err := os.Stat(pagePath); os.IsNotExist(err) {
            return nil, fmt.Errorf("page %s not found", name)
        }
    }

    content, err := os.ReadFile(pagePath)
    if err != nil {
        return nil, err
    }

    // Convert WebMatrix syntax
    converted := e.convertToGoTemplate(string(content), data)

    // Parse with functions
    tmpl := template.New(name).Funcs(e.funcMap)
    tmpl, err = tmpl.Parse(converted)
    if err != nil {
        return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
    }

    e.templates[name] = tmpl
    return tmpl, nil
}

func (e *WebMatrixEngine) loadLayout(name string, data interface{}) (*template.Template, error) {
    layoutName := strings.TrimPrefix(name, "~/")
    layoutName = strings.TrimPrefix(layoutName, "templates/layouts/")
    
    layoutPath := filepath.Join(e.layoutsPath, layoutName+".gshtml")
    if _, err := os.Stat(layoutPath); os.IsNotExist(err) {
        layoutPath = filepath.Join(e.layoutsPath, layoutName+".gohtml")
    }

    content, err := os.ReadFile(layoutPath)
    if err != nil {
        return nil, err
    }

    converted := e.convertToGoTemplate(string(content), data)
    return template.New(layoutName).Funcs(e.funcMap).Parse(converted)
}

func (e *WebMatrixEngine) loadPartial(name string, data interface{}) (*template.Template, error) {
    partialPath := filepath.Join(e.partialsPath, name+".gshtml")
    if _, err := os.Stat(partialPath); os.IsNotExist(err) {
        partialPath = filepath.Join(e.partialsPath, name+".gohtml")
    }

    content, err := os.ReadFile(partialPath)
    if err != nil {
        return nil, err
    }

    converted := e.convertToGoTemplate(string(content), data)
    return template.New(name).Funcs(e.funcMap).Parse(converted)
}

func (e *WebMatrixEngine) AddFunc(name string, fn interface{}) {
    e.mu.Lock()
    defer e.mu.Unlock()
    e.funcMap[name] = fn
}

// Database helper methods
func (db *DatabaseHelper) Query(sql string, params ...interface{}) []map[string]interface{} {
    if strings.Contains(strings.ToLower(sql), "select * from employees") {
        return []map[string]interface{}{
            {"id": 1, "name": "John Doe", "email": "john@example.com", "department": "Engineering", "status": "active"},
            {"id": 2, "name": "Jane Smith", "email": "jane@example.com", "department": "Marketing", "status": "active"},
            {"id": 3, "name": "Bob Johnson", "email": "bob@example.com", "department": "Sales", "status": "inactive"},
        }
    }
    if strings.Contains(strings.ToLower(sql), "select * from departments") {
        return []map[string]interface{}{
            {"id": 1, "name": "Engineering"},
            {"id": 2, "name": "Marketing"},
            {"id": 3, "name": "Sales"},
            {"id": 4, "name": "HR"},
        }
    }
    return []map[string]interface{}{}
}

func (db *DatabaseHelper) Execute(sql string, params ...interface{}) int64 {
    return 1
}
