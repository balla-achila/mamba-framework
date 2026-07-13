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

// Extract Layout, Page.Title, and variables from @{} block
func (e *WebMatrixEngine) processCodeBlocks(content string, data interface{}) (string, interface{}) {
    codeBlockRegex := regexp.MustCompile(`@\{([^}]*)\}`)
    
    processed := content
    matches := codeBlockRegex.FindAllStringSubmatch(content, -1)
    
    for _, match := range matches {
        if len(match) > 1 {
            code := strings.TrimSpace(match[1])
            e.extractVariables(code, data)
            e.extractLayoutAndTitle(code, data)
        }
    }
    
    // Remove @{} blocks
    processed = codeBlockRegex.ReplaceAllString(processed, "")
    
    return processed, data
}

func (e *WebMatrixEngine) extractVariables(code string, data interface{}) {
    varRegex := regexp.MustCompile(`var\s+(\w+)\s*=\s*([^;]+);?`)
    matches := varRegex.FindAllStringSubmatch(code, -1)
    
    pageData, ok := data.(*PageData)
    if !ok {
        return
    }
    
    for _, match := range matches {
        if len(match) > 2 {
            varName := strings.TrimSpace(match[1])
            varValue := strings.TrimSpace(match[2])
            value := e.evaluateExpression(varValue, data)
            pageData.SetVar(varName, value)
        }
    }
}

func (e *WebMatrixEngine) extractLayoutAndTitle(code string, data interface{}) {
    pageData, ok := data.(*PageData)
    if !ok {
        return
    }
    
    // Extract Layout = "..."
    layoutRegex := regexp.MustCompile(`Layout\s*=\s*["']([^"']+)["']`)
    if matches := layoutRegex.FindStringSubmatch(code); len(matches) > 1 {
        pageData.Layout = matches[1]
    }
    
    // Extract Page.Title = "..."
    titleRegex := regexp.MustCompile(`Page\.Title\s*=\s*["']([^"']+)["']`)
    if matches := titleRegex.FindStringSubmatch(code); len(matches) > 1 {
        pageData.Title = matches[1]
    }
}

func (e *WebMatrixEngine) evaluateExpression(expr string, data interface{}) interface{} {
    expr = strings.TrimSpace(expr)
    
    // String literal
    if strings.HasPrefix(expr, `"`) && strings.HasSuffix(expr, `"`) {
        return expr[1 : len(expr)-1]
    }
    
    // AppSettings["key"]
    appSettingsRegex := regexp.MustCompile(`AppSettings\["([^"]+)"\]`)
    if matches := appSettingsRegex.FindStringSubmatch(expr); len(matches) > 1 {
        return e.appSettingsFunc(matches[1])
    }
    
    // ConnectionString["name"]
    connStringRegex := regexp.MustCompile(`ConnectionString\["([^"]+)"\]`)
    if matches := connStringRegex.FindStringSubmatch(expr); len(matches) > 1 {
        return e.connectionStringFunc(matches[1])
    }
    
    // Page.IsPost
    if expr == "Page.IsPost" {
        if pageData, ok := data.(*PageData); ok {
            return pageData.IsPost
        }
    }
    
    // Page.IsGet
    if expr == "Page.IsGet" {
        if pageData, ok := data.(*PageData); ok {
            return pageData.IsGet
        }
    }
    
    // Page.Form["key"]
    formRegex := regexp.MustCompile(`Page\.Form\["([^"]+)"\]`)
    if matches := formRegex.FindStringSubmatch(expr); len(matches) > 1 {
        if pageData, ok := data.(*PageData); ok {
            if val, ok := pageData.Form[matches[1]]; ok {
                return val
            }
        }
    }
    
    // Page.Session["key"]
    sessionRegex := regexp.MustCompile(`Page\.Session\["([^"]+)"\]`)
    if matches := sessionRegex.FindStringSubmatch(expr); len(matches) > 1 {
        if pageData, ok := data.(*PageData); ok {
            if val, ok := pageData.Session[matches[1]]; ok {
                return val
            }
        }
    }
    
    return expr
}

// Convert WebMatrix @ syntax to Go template syntax
func (e *WebMatrixEngine) convertToGoTemplate(content string, data interface{}) string {
    // Process @{} code blocks first
    processed, _ := e.processCodeBlocks(content, data)
    
    // Handle @@ escaping
    processed = strings.ReplaceAll(processed, "@@", "___AT_AT___")
    
    // Convert WebMatrix @ syntax to Go template {{ }} syntax
    
    // @Page.Title -> {{.Title}}
    processed = regexp.MustCompile(`@Page\.Title`).ReplaceAllString(processed, `{{.Title}}`)
    
    // @Page.IsPost -> {{.IsPost}}
    processed = regexp.MustCompile(`@Page\.IsPost`).ReplaceAllString(processed, `{{.IsPost}}`)
    
    // @Page.IsGet -> {{.IsGet}}
    processed = regexp.MustCompile(`@Page\.IsGet`).ReplaceAllString(processed, `{{.IsGet}}`)
    
    // @Page.Form["key"] -> {{index .Form "key"}}
    processed = regexp.MustCompile(`@Page\.Form\["([^"]+)"\]`).ReplaceAllString(processed, `{{index .Form "$1"}}`)
    
    // @Page.QueryString["key"] -> {{index .QueryString "key"}}
    processed = regexp.MustCompile(`@Page\.QueryString\["([^"]+)"\]`).ReplaceAllString(processed, `{{index .QueryString "$1"}}`)
    
    // @Page.Session["key"] -> {{index .Session "key"}}
    processed = regexp.MustCompile(`@Page\.Session\["([^"]+)"\]`).ReplaceAllString(processed, `{{index .Session "$1"}}`)
    
    // @Page.UrlData[0] -> {{index .UrlData 0}}
    processed = regexp.MustCompile(`@Page\.UrlData\[([0-9]+)\]`).ReplaceAllString(processed, `{{index .UrlData $1}}`)
    
    // @AppSettings["key"] -> {{AppSettings "key"}}
    processed = regexp.MustCompile(`@AppSettings\["([^"]+)"\]`).ReplaceAllString(processed, `{{AppSettings "$1"}}`)
    
    // @ConnectionString["name"] -> {{ConnectionString "name"}}
    processed = regexp.MustCompile(`@ConnectionString\["([^"]+)"\]`).ReplaceAllString(processed, `{{ConnectionString "$1"}}`)
    
    // @Html.AntiForgeryToken() -> {{Html.AntiForgeryToken}}
    processed = regexp.MustCompile(`@Html\.AntiForgeryToken\(\)`).ReplaceAllString(processed, `{{Html.AntiForgeryToken}}`)
    
    // @Database.Query() -> {{Database.Query}}
    processed = regexp.MustCompile(`@Database\.Query\(([^)]*)\)`).ReplaceAllString(processed, `{{Database.Query $1}}`)
    
    // @RenderBody() -> {{RenderBody .}}
    processed = regexp.MustCompile(`@RenderBody\(\)`).ReplaceAllString(processed, `{{RenderBody .}}`)
    
    // @RenderSection("name") -> {{RenderSection "name" .}}
    processed = regexp.MustCompile(`@RenderSection\(["']([^"']+)["']\)`).ReplaceAllString(processed, `{{RenderSection "$1" .}}`)
    
    // @IsSectionDefined("name") -> {{IsSectionDefined "name" .}}
    processed = regexp.MustCompile(`@IsSectionDefined\(["']([^"']+)["']\)`).ReplaceAllString(processed, `{{IsSectionDefined "$1" .}}`)
    
    // @section name { } -> {{define "name"}}...{{end}}
    sectionRegex := regexp.MustCompile(`@section\s+(\w+)\s*{([^}]*)}`)
    processed = sectionRegex.ReplaceAllString(processed, `{{define "$1"}}$2{{end}}`)
    
    // @foreach(var item in items) -> {{range $index, $item := .Items}}
    processed = regexp.MustCompile(`@foreach\s*\(\s*var\s+(\w+)\s+in\s+(\w+)\s*\)\s*{`).ReplaceAllString(processed, `{{range $index, $1 := .$2}}`)
    
    // @if(condition) -> {{if condition}}
    processed = regexp.MustCompile(`@if\s*\(([^)]+)\)\s*{`).ReplaceAllString(processed, `{{if $1}}`)
    
    // @else -> {{else}}
    processed = regexp.MustCompile(`@else\s*{`).ReplaceAllString(processed, `{{else}}`)
    
    // @} -> {{end}}
    processed = regexp.MustCompile(`@}`).ReplaceAllString(processed, `{{end}}`)
    
    // Restore @@ escaping
    processed = strings.ReplaceAll(processed, "___AT_AT___", "@")
    
    return processed
}

func (e *WebMatrixEngine) Render(w io.Writer, name string, data interface{}) error {
    // Store page data
    if pageData, ok := data.(*PageData); ok {
        e.pageData = pageData
    }

    // Load the page template
    tmpl, err := e.loadTemplate(name, data)
    if err != nil {
        return err
    }

    // Execute the page template to capture its output
    var bodyBuf bytes.Buffer
    if err := tmpl.Execute(&bodyBuf, data); err != nil {
        return err
    }

    // If we have page data and a layout, render with layout
    if pageData, ok := data.(*PageData); ok && pageData.Layout != "" {
        // Store the body content
        pageData.Body = template.HTML(bodyBuf.String())
        
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

    // If no layout, just write the body
    _, err = w.Write(bodyBuf.Bytes())
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
