package layout

import (
    "fmt"
    "html/template"
    "io"
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
}

type LayoutData struct {
    Title   string
    Content template.HTML
    Data    map[string]interface{}
    Section map[string]template.HTML
    Layout  string
}

type Layout interface {
    Render(w io.Writer, name string, data interface{}) error
    RenderPartial(w io.Writer, name string, data interface{}) error
    RenderLayout(w io.Writer, name string, data interface{}) error
    AddFunc(name string, fn interface{})
}

func New(basePath, layoutPath, partialPath string) *LayoutEngine {
    return &LayoutEngine{
        templates:   make(map[string]*template.Template),
        basePath:    basePath,
        layoutPath:  layoutPath,
        partialPath: partialPath,
    }
}

func (le *LayoutEngine) Render(w io.Writer, name string, data interface{}) error {
    tmpl, err := le.getTemplate(name)
    if err != nil {
        return err
    }

    // Check if data has layout specified
    var layoutName string
    if layoutData, ok := data.(*LayoutData); ok && layoutData.Layout != "" {
        layoutName = layoutData.Layout
    }

    if layoutName != "" {
        // Wrap content in layout
        layoutData, ok := data.(*LayoutData)
        if !ok {
            return fmt.Errorf("layout requires LayoutData type")
        }

        // Execute page content first to capture sections
        var contentBuf strings.Builder
        if err := tmpl.ExecuteTemplate(&contentBuf, "content", layoutData); err != nil {
            return err
        }
        layoutData.Content = template.HTML(contentBuf.String())

        // Execute layout
        layoutTmpl, err := le.getLayout(layoutName)
        if err != nil {
            return err
        }

        return layoutTmpl.Execute(w, layoutData)
    }

    return tmpl.Execute(w, data)
}

func (le *LayoutEngine) RenderPartial(w io.Writer, name string, data interface{}) error {
    tmpl, err := le.getPartial(name)
    if err != nil {
        return err
    }
    return tmpl.Execute(w, data)
}

func (le *LayoutEngine) RenderLayout(w io.Writer, name string, data interface{}) error {
    tmpl, err := le.getLayout(name)
    if err != nil {
        return err
    }
    return tmpl.Execute(w, data)
}

func (le *LayoutEngine) AddFunc(name string, fn interface{}) {
    le.mu.Lock()
    defer le.mu.Unlock()

    for _, tmpl := range le.templates {
        tmpl.Funcs(template.FuncMap{name: fn})
    }
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

    // Check again in case another goroutine loaded it
    if tmpl, exists := le.templates[name]; exists {
        return tmpl, nil
    }

    // Load the page template
    pagePath := filepath.Join(le.basePath, name+".gohtml")
    
    // Load all partials
    partialPaths, err := filepath.Glob(filepath.Join(le.partialPath, "*.gohtml"))
    if err != nil {
        return nil, err
    }

    // Prepare template with base functions
    tmpl := template.New(name).Funcs(template.FuncMap{
        "Layout": func(layout string) string {
            return layout
        },
        "Partial": func(partial string) string {
            return partial
        },
        "Include": func(partial string, data interface{}) template.HTML {
            // This will be implemented via rendering
            return template.HTML("")
        },
    })

    // Parse page
    tmpl, err = tmpl.ParseFiles(pagePath)
    if err != nil {
        return nil, fmt.Errorf("failed to parse page %s: %w", pagePath, err)
    }

    // Parse partials
    for _, partialPath := range partialPaths {
        _, err = tmpl.ParseFiles(partialPath)
        if err != nil {
            return nil, fmt.Errorf("failed to parse partial %s: %w", partialPath, err)
        }
    }

    le.templates[name] = tmpl
    return tmpl, nil
}

func (le *LayoutEngine) getLayout(name string) (*template.Template, error) {
    layoutPath := filepath.Join(le.layoutPath, name+".gohtml")
    
    tmpl := template.New(name).Funcs(template.FuncMap{
        "Content": func() template.HTML {
            return template.HTML("")
        },
        "Section": func(name string) template.HTML {
            return template.HTML("")
        },
        "Partial": func(partial string, data interface{}) template.HTML {
            return template.HTML("")
        },
        "Yield": func() template.HTML {
            return template.HTML("")
        },
    })

    tmpl, err := tmpl.ParseFiles(layoutPath)
    if err != nil {
        return nil, fmt.Errorf("failed to parse layout %s: %w", layoutPath, err)
    }

    return tmpl, nil
}

func (le *LayoutEngine) ClearCache() {
    le.mu.Lock()
    defer le.mu.Unlock()
    le.templates = make(map[string]*template.Template)
}

// LayoutData helper functions
func NewLayoutData(title string) *LayoutData {
    return &LayoutData{
        Title:   title,
        Data:    make(map[string]interface{}),
        Section: make(map[string]template.HTML),
    }
}

func (ld *LayoutData) Set(key string, value interface{}) {
    ld.Data[key] = value
}

func (ld *LayoutData) Get(key string) interface{} {
    return ld.Data[key]
}

func (ld *LayoutData) SetSection(name string, content template.HTML) {
    ld.Section[name] = content
}

func (ld *LayoutData) GetSection(name string) template.HTML {
    if content, ok := ld.Section[name]; ok {
        return content
    }
    return template.HTML("")
}