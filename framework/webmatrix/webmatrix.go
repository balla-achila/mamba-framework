package webmatrix

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
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

// templateCacheItem stores the compiled template along with its raw code blocks
// so that request-scoped metadata can be re-extracted on cache hits.
type templateCacheItem struct {
	tmpl       *template.Template
	codeBlocks []string
}

// WebMatrixEngine is the main engine that handles template parsing and rendering
type WebMatrixEngine struct {
	templates    map[string]templateCacheItem // Fixed: updated to use templateCacheItem
	pagesPath    string
	layoutsPath  string
	partialsPath string
	mu           sync.RWMutex
	funcMap      template.FuncMap
	config       *WebConfig
	pageData     *PageData
	dbs          map[string]*sql.DB // connection name -> database connection pool
	dbMu         sync.RWMutex
}

// WebConfig holds configuration settings from web.config
type WebConfig struct {
	ConnectionStrings map[string]string
	AppSettings       map[string]string
}

// DatabaseHelper provides database operations with WebMatrix-style syntax
type DatabaseHelper struct {
	engine   *WebMatrixEngine
	conn     *sql.DB
	connName string
}

// PageData holds all data for a page request
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

// HtmlHelper provides HTML helpers like AntiForgeryToken
type HtmlHelper struct {
	pageData *PageData
}

// csrfCookieName is the cookie used to persist the anti-forgery token across
// the GET (render form) / POST (submit form) request pair (double-submit
// cookie pattern).
const csrfCookieName = "csrf_token"

// generateCSRFToken returns a cryptographically random, URL-safe token.
func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// AntiForgeryToken returns a hidden input containing a real, per-session
// random CSRF token (not a hardcoded value). If a token cookie already
// exists on the request it's reused so the value submitted on POST matches;
// otherwise a new token is generated and set as a cookie on the response.
func (h *HtmlHelper) AntiForgeryToken() template.HTML {
	token, err := generateCSRFToken()
	if err != nil {
		// crypto/rand failing is effectively unrecoverable; render nothing
		// rather than a fake/guessable token.
		return ""
	}

	if h.pageData != nil && h.pageData.Request != nil {
		if c, cookieErr := h.pageData.Request.Cookie(csrfCookieName); cookieErr == nil && c.Value != "" {
			token = c.Value
		} else if h.pageData.Response != nil {
			http.SetCookie(h.pageData.Response, &http.Cookie{
				Name:     csrfCookieName,
				Value:    token,
				Path:     "/",
				HttpOnly: false, // must be readable... actually not read by JS; kept false only so it round-trips as a normal cookie
				SameSite: http.SameSiteStrictMode,
				Secure:   h.pageData.Request.TLS != nil,
			})
		}
	}

	return template.HTML(`<input type="hidden" name="__RequestVerificationToken" value="` + template.HTMLEscapeString(token) + `" />`)
}

// VerifyAntiForgeryToken checks the submitted form token against the
// csrf_token cookie set by AntiForgeryToken. Call this in POST handlers
// before trusting form data.
func VerifyAntiForgeryToken(r *http.Request) bool {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	r.ParseForm()
	submitted := r.FormValue("__RequestVerificationToken")
	if submitted == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(submitted)) == 1
}

// NewPageData creates a new PageData instance with default values
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

// SetRequest populates PageData from the HTTP request
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

// GetVar retrieves a variable from the @{} code block
func (p *PageData) GetVar(key string) interface{} {
	if val, ok := p.Vars[key]; ok {
		return val
	}
	return nil
}

// SetVar stores a variable from the @{} code block
func (p *PageData) SetVar(key string, value interface{}) {
	p.Vars[key] = value
}

// DefaultWebConfig returns a default configuration
func DefaultWebConfig() *WebConfig {
	config := &WebConfig{
		ConnectionStrings: make(map[string]string),
		AppSettings:       make(map[string]string),
	}
	config.ConnectionStrings["DefaultConnection"] = "postgres://user:pass@localhost/mambadb?sslmode=disable"
	config.ConnectionStrings["MambaDB"] = "postgres://mamba:mamba@localhost/mamba?sslmode=disable"
	config.AppSettings["SiteName"] = "Mamba Framework"
	config.AppSettings["Version"] = "1.0.0"
	config.AppSettings["Environment"] = "Development"
	config.AppSettings["CompanyName"] = "Mamba Technologies"
	config.AppSettings["SupportEmail"] = "support@mamba-framework.com"
	return config
}

// NewWebMatrixEngine creates a new WebMatrix engine
func NewWebMatrixEngine(pagesPath, layoutsPath, partialsPath string) *WebMatrixEngine {
	engine := &WebMatrixEngine{
		templates:    make(map[string]templateCacheItem), // Fixed: updated map initialization
		pagesPath:    pagesPath,
		layoutsPath:  layoutsPath,
		partialsPath: partialsPath,
		funcMap:      make(template.FuncMap),
		config:       DefaultWebConfig(),
		dbs:          make(map[string]*sql.DB),
	}

	// Register template functions
	engine.funcMap["RenderBody"] = engine.renderBodyFunc
	engine.funcMap["RenderSection"] = engine.renderSectionFunc
	engine.funcMap["IsSectionDefined"] = engine.isSectionDefinedFunc
	engine.funcMap["DateTime"] = func() interface{} { return time.Now() }
	engine.funcMap["AppSettings"] = engine.appSettingsFunc
	engine.funcMap["ConnectionString"] = engine.connectionStringFunc
	engine.funcMap["Html"] = func() *HtmlHelper { return &HtmlHelper{pageData: engine.pageData} }
	engine.funcMap["Database"] = engine.databaseFunc
	engine.funcMap["Page"] = func() *PageData { return engine.pageData }
	engine.funcMap["Raw"] = func(s string) template.HTML { return template.HTML(s) }
	engine.funcMap["Var"] = engine.varFunc

	return engine
}

// SetDB registers a database connection with a name (e.g., "DefaultConnection")
func (e *WebMatrixEngine) SetDB(name string, db *sql.DB) {
	e.dbMu.Lock()
	defer e.dbMu.Unlock()
	e.dbs[name] = db
}

// GetDB returns the database connection for the given name
func (e *WebMatrixEngine) GetDB(name string) *sql.DB {
	e.dbMu.RLock()
	defer e.dbMu.RUnlock()
	return e.dbs[name]
}

// databaseFunc returns a DatabaseHelper for templates
func (e *WebMatrixEngine) databaseFunc() *DatabaseHelper {
	return &DatabaseHelper{engine: e}
}

// varFunc retrieves a variable by key
func (e *WebMatrixEngine) varFunc(key string) interface{} {
	if e.pageData != nil {
		return e.pageData.GetVar(key)
	}
	return nil
}

// renderBodyFunc renders the page body
func (e *WebMatrixEngine) renderBodyFunc(data interface{}) template.HTML {
	if pageData, ok := data.(*PageData); ok {
		return pageData.Body
	}
	return template.HTML("")
}

// renderSectionFunc renders a named section
func (e *WebMatrixEngine) renderSectionFunc(name string, data interface{}) template.HTML {
	if pageData, ok := data.(*PageData); ok {
		if section, exists := pageData.Sections[name]; exists {
			return section
		}
	}
	return template.HTML("")
}

// isSectionDefinedFunc checks if a section is defined
func (e *WebMatrixEngine) isSectionDefinedFunc(name string, data interface{}) bool {
	if pageData, ok := data.(*PageData); ok {
		_, exists := pageData.Sections[name]
		return exists
	}
	return false
}

// appSettingsFunc returns an app setting by key
func (e *WebMatrixEngine) appSettingsFunc(key string) string {
	if val, ok := e.config.AppSettings[key]; ok {
		return val
	}
	return ""
}

// connectionStringFunc returns a connection string by name
func (e *WebMatrixEngine) connectionStringFunc(name string) string {
	if val, ok := e.config.ConnectionStrings[name]; ok {
		return val
	}
	return ""
}

// ConnectionString returns the named connection string from config.
func (e *WebMatrixEngine) ConnectionString(name string) string {
	return e.connectionStringFunc(name)
}

// AppSetting returns the named app setting from config.
func (e *WebMatrixEngine) AppSetting(name string) string {
	return e.appSettingsFunc(name)
}

// =============================================================================
// DatabaseHelper methods (WebMatrix-style database operations)
// =============================================================================

// Open opens a database connection using the named connection string
func (db *DatabaseHelper) Open(connName string) *DatabaseHelper {
	db.connName = connName
	existing := db.engine.GetDB(connName)
	if existing != nil {
		db.conn = existing
		return db
	}

	connStr := db.engine.connectionStringFunc(connName)
	if connStr == "" {
		return db
	}

	sqlDB, err := sql.Open("postgres", connStr)
	if err != nil {
		return db
	}
	if err := sqlDB.Ping(); err != nil {
		return db
	}
	db.engine.SetDB(connName, sqlDB)
	db.conn = sqlDB
	return db
}

// Query executes a SQL query and returns a slice of maps
func (db *DatabaseHelper) Query(sqlQuery string, args ...interface{}) []map[string]interface{} {
	if db.conn != nil {
		rows, err := db.conn.Query(sqlQuery, args...)
		if err != nil {
			return []map[string]interface{}{}
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return []map[string]interface{}{}
		}

		var results []map[string]interface{}
		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}
			if err := rows.Scan(valuePtrs...); err != nil {
				continue
			}
			row := make(map[string]interface{})
			for i, col := range columns {
				if b, ok := values[i].([]byte); ok {
					row[col] = string(b)
				} else {
					row[col] = values[i]
				}
			}
			results = append(results, row)
		}
		return results
	}

	return db.fallbackQuery(sqlQuery)
}

// fallbackQuery returns sample data for demo purposes
func (db *DatabaseHelper) fallbackQuery(sqlQuery string) []map[string]interface{} {
	lower := strings.ToLower(sqlQuery)
	if strings.Contains(lower, "select * from employees") {
		return []map[string]interface{}{
			{"id": 1, "name": "John Doe", "email": "john@example.com", "department": "Engineering", "status": "active"},
			{"id": 2, "name": "Jane Smith", "email": "jane@example.com", "department": "Marketing", "status": "active"},
			{"id": 3, "name": "Bob Johnson", "email": "bob@example.com", "department": "Sales", "status": "inactive"},
		}
	}
	if strings.Contains(lower, "select * from departments") {
		return []map[string]interface{}{
			{"id": 1, "name": "Engineering"},
			{"id": 2, "name": "Marketing"},
			{"id": 3, "name": "Sales"},
			{"id": 4, "name": "HR"},
		}
	}
	return []map[string]interface{}{}
}

// QuerySingle returns a single row as a map
func (db *DatabaseHelper) QuerySingle(sqlQuery string, args ...interface{}) map[string]interface{} {
	rows := db.Query(sqlQuery, args...)
	if len(rows) > 0 {
		return rows[0]
	}
	return map[string]interface{}{}
}

// QueryValue returns a single value from the first column of the first row.
// Uses column index (not the Query() map, whose key order is randomized by
// Go's map implementation) so "first column" is actually the first column.
func (db *DatabaseHelper) QueryValue(sqlQuery string, args ...interface{}) interface{} {
	if db.conn != nil {
		rows, err := db.conn.Query(sqlQuery, args...)
		if err != nil {
			return nil
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil || len(columns) == 0 {
			return nil
		}
		if !rows.Next() {
			return nil
		}

		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil
		}

		if b, ok := values[0].([]byte); ok {
			return string(b)
		}
		return values[0]
	}

	// No live connection: using demo/fallback data, which is map-based and
	// has no real column order. Best effort: return "id", the conventional
	// first column in every fallback dataset, when present.
	rows := db.fallbackQuery(sqlQuery)
	if len(rows) > 0 {
		if id, ok := rows[0]["id"]; ok {
			return id
		}
		for _, v := range rows[0] {
			return v
		}
	}
	return nil
}

// Execute runs an INSERT, UPDATE, or DELETE query and returns rows affected
func (db *DatabaseHelper) Execute(sqlQuery string, args ...interface{}) int64 {
	if db.conn != nil {
		result, err := db.conn.Exec(sqlQuery, args...)
		if err != nil {
			return 0
		}
		affected, _ := result.RowsAffected()
		return affected
	}
	return 1
}

// =============================================================================
// Template parsing and rendering
// =============================================================================

// parseCodeBlocks collects all raw blocks without modifying the target layout structure immediately.
// parseCodeBlocks finds @{ ... } blocks using brace-depth counting (via
// findMatchingBrace) instead of a naive [^}]* regex, so blocks containing
// nested braces (e.g. an `if (...) { ... }` inside the code block) are
// captured in full instead of being truncated at the first inner `}`.
func (e *WebMatrixEngine) parseCodeBlocks(content string) (string, []string) {
	var blocks []string
	var out strings.Builder
	pos := 0

	for {
		idx := strings.Index(content[pos:], "@{")
		if idx == -1 {
			out.WriteString(content[pos:])
			break
		}
		start := pos + idx
		out.WriteString(content[pos:start])

		openIdx := start + 1 // index of the '{'
		closeIdx, ok := findMatchingBrace(content, openIdx)
		if !ok {
			// No matching brace found; treat the rest as literal to avoid
			// dropping content or looping forever.
			out.WriteString(content[start:])
			pos = len(content)
			break
		}

		blocks = append(blocks, strings.TrimSpace(content[openIdx+1:closeIdx]))
		pos = closeIdx + 1
	}

	return out.String(), blocks
}

// executeCodeBlocks processes a set of pre-discovered code blocks against request-scoped data.
func (e *WebMatrixEngine) executeCodeBlocks(blocks []string, data interface{}) {
	for _, code := range blocks {
		e.extractVariables(code, data)
		e.executeConditionalBlocks(code, data)
		e.extractLayoutAndTitle(code, data)
	}
}

var codeIfRegex = regexp.MustCompile(`if\s*\(`)

// executeConditionalBlocks finds if/else blocks inside a @{ } code block
// and, for whichever branch's condition is true, applies var declarations
// and plain reassignments (e.g. `saved = true;`, no "var" keyword) found in
// that branch's body. Without this, a code block's `if` is just inert text
// to the engine -- it's never evaluated, so a variable only ever keeps its
// initial value, regardless of what the if/else was meant to do (e.g. the
// common "if (IsPost && field is valid) { saved = true; }" idiom).
func (e *WebMatrixEngine) executeConditionalBlocks(code string, data interface{}) {
	pos := 0
	for {
		loc := codeIfRegex.FindStringIndex(code[pos:])
		if loc == nil {
			return
		}
		openParen := pos + loc[1] - 1
		closeParen, ok := findMatchingParen(code, openParen)
		if !ok {
			return
		}
		cond := code[openParen+1 : closeParen]

		j := closeParen + 1
		for j < len(code) && isSpaceByte(code[j]) {
			j++
		}
		if j >= len(code) || code[j] != '{' {
			pos = closeParen + 1
			continue
		}
		bodyEnd, ok := findMatchingBrace(code, j)
		if !ok {
			return
		}
		body := code[j+1 : bodyEnd]

		condResult := e.evalCodeCondition(cond, data)

		k := bodyEnd + 1
		for k < len(code) && isSpaceByte(code[k]) {
			k++
		}
		elseBody := ""
		hasElse := false
		nextPos := bodyEnd + 1
		if strings.HasPrefix(code[k:], "else") {
			k2 := k + len("else")
			for k2 < len(code) && isSpaceByte(code[k2]) {
				k2++
			}
			if k2 < len(code) && code[k2] == '{' {
				if elseEnd, ok2 := findMatchingBrace(code, k2); ok2 {
					elseBody = code[k2+1 : elseEnd]
					hasElse = true
					nextPos = elseEnd + 1
				}
			}
		}

		var chosenBody string
		if condResult {
			chosenBody = body
		} else if hasElse {
			chosenBody = elseBody
		}

		if chosenBody != "" {
			e.extractVariables(chosenBody, data)
			e.executeReassignments(chosenBody, data)
			e.executeConditionalBlocks(chosenBody, data) // supports nesting
		}

		pos = nextPos
	}
}

// isNullOrEmptyRegex matches the WebMatrix/C#-style string.IsNullOrEmpty(x)
// null-check idiom used inside @{ } code block conditions.
var isNullOrEmptyRegex = regexp.MustCompile(`^string\.IsNullOrEmpty\((.+)\)$`)

// evalCodeCondition evaluates a restricted subset of C#-like boolean
// expressions found inside @{ } code blocks: &&, ||, ! negation, IsPost /
// IsGet, string.IsNullOrEmpty(x), and bare variable truthiness. It is not a
// general expression evaluator -- only what's needed for the common
// "if (IsPost && !string.IsNullOrEmpty(field)) { ... }" validation idiom.
func (e *WebMatrixEngine) evalCodeCondition(cond string, data interface{}) bool {
	cond = strings.TrimSpace(cond)

	if strings.Contains(cond, "&&") {
		for _, part := range strings.Split(cond, "&&") {
			if !e.evalCodeCondition(part, data) {
				return false
			}
		}
		return true
	}
	if strings.Contains(cond, "||") {
		for _, part := range strings.Split(cond, "||") {
			if e.evalCodeCondition(part, data) {
				return true
			}
		}
		return false
	}

	negate := false
	for strings.HasPrefix(cond, "!") {
		negate = true
		cond = strings.TrimSpace(cond[1:])
	}

	var result bool
	switch {
	case cond == "IsPost" || cond == "Page.IsPost":
		if pageData, ok := data.(*PageData); ok {
			result = pageData.IsPost
		}
	case cond == "IsGet" || cond == "Page.IsGet":
		if pageData, ok := data.(*PageData); ok {
			result = pageData.IsGet
		}
	default:
		if m := isNullOrEmptyRegex.FindStringSubmatch(cond); m != nil {
			val := e.lookupCodeVar(strings.TrimSpace(m[1]), data)
			result = val == "" || val == nil
		} else {
			result = isTruthy(e.lookupCodeVar(cond, data))
		}
	}

	if negate {
		return !result
	}
	return result
}

// lookupCodeVar resolves a bare identifier previously assigned via `var` (or
// a plain reassignment) in a code block to its current value.
func (e *WebMatrixEngine) lookupCodeVar(name string, data interface{}) interface{} {
	name = strings.TrimSpace(name)
	if pageData, ok := data.(*PageData); ok {
		if v, ok := pageData.Vars[name]; ok {
			return v
		}
	}
	return nil
}

func isTruthy(v interface{}) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	default:
		return true
	}
}

// reassignRegex matches plain "identifier = expr;" statements (no "var"
// keyword) -- an update to a variable already declared earlier in the code
// block, typically inside a conditional branch.
var reassignRegex = regexp.MustCompile(`(?m)^\s*(\w+)\s*=\s*([^;=][^;]*);`)

func (e *WebMatrixEngine) executeReassignments(code string, data interface{}) {
	pageData, ok := data.(*PageData)
	if !ok {
		return
	}
	for _, match := range reassignRegex.FindAllStringSubmatch(code, -1) {
		varName := strings.TrimSpace(match[1])
		varValue := strings.TrimSpace(match[2])
		pageData.SetVar(varName, e.evaluateExpression(varValue, data))
	}
}

// extractVariables extracts var declarations from the code block
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

// extractLayoutAndTitle extracts Layout and Page.Title from the code block
func (e *WebMatrixEngine) extractLayoutAndTitle(code string, data interface{}) {
	pageData, ok := data.(*PageData)
	if !ok {
		return
	}

	layoutRegex := regexp.MustCompile(`Layout\s*=\s*["']([^"']+)["']`)
	if matches := layoutRegex.FindStringSubmatch(code); len(matches) > 1 {
		pageData.Layout = matches[1]
	}

	titleRegex := regexp.MustCompile(`Page\.Title\s*=\s*["']([^"']+)["']`)
	if matches := titleRegex.FindStringSubmatch(code); len(matches) > 1 {
		pageData.Title = matches[1]
	}
}

// evaluateExpression evaluates simple expressions in the code block
func (e *WebMatrixEngine) evaluateExpression(expr string, data interface{}) interface{} {
	expr = strings.TrimSpace(expr)

	// Boolean literals must become real Go bools -- otherwise the string
	// "false" is a non-empty string, which Go templates treat as truthy,
	// silently inverting every `var x = false;` / `x = false;` in the page.
	if expr == "true" {
		return true
	}
	if expr == "false" {
		return false
	}

	if strings.HasPrefix(expr, `"`) && strings.HasSuffix(expr, `"`) {
		return expr[1 : len(expr)-1]
	}

	appSettingsRegex := regexp.MustCompile(`AppSettings\["([^"]+)"\]`)
	if matches := appSettingsRegex.FindStringSubmatch(expr); len(matches) > 1 {
		return e.appSettingsFunc(matches[1])
	}

	connStringRegex := regexp.MustCompile(`ConnectionString\["([^"]+)"\]`)
	if matches := connStringRegex.FindStringSubmatch(expr); len(matches) > 1 {
		return e.connectionStringFunc(matches[1])
	}

	if expr == "Page.IsPost" {
		if pageData, ok := data.(*PageData); ok {
			return pageData.IsPost
		}
	}

	if expr == "Page.IsGet" {
		if pageData, ok := data.(*PageData); ok {
			return pageData.IsGet
		}
	}

	if expr == "Page.User" {
		if pageData, ok := data.(*PageData); ok {
			return pageData.User
		}
	}

	formRegex := regexp.MustCompile(`Page\.Form\["([^"]+)"\]`)
	if matches := formRegex.FindStringSubmatch(expr); len(matches) > 1 {
		if pageData, ok := data.(*PageData); ok {
			if val, ok := pageData.Form[matches[1]]; ok {
				return val
			}
		}
		return ""
	}

	queryStringRegex := regexp.MustCompile(`Page\.QueryString\["([^"]+)"\]`)
	if matches := queryStringRegex.FindStringSubmatch(expr); len(matches) > 1 {
		if pageData, ok := data.(*PageData); ok {
			if val, ok := pageData.QueryString[matches[1]]; ok {
				return val
			}
		}
		return ""
	}

	sessionRegex := regexp.MustCompile(`Page\.Session\["([^"]+)"\]`)
	if matches := sessionRegex.FindStringSubmatch(expr); len(matches) > 1 {
		if pageData, ok := data.(*PageData); ok {
			if val, ok := pageData.Session[matches[1]]; ok {
				return val
			}
		}
		return ""
	}

	// Not a recognized construct: rather than silently returning the raw,
	// unresolved expression text as if it were a real (non-empty) value --
	// which would make e.g. string.IsNullOrEmpty(x) see a "value" that was
	// never actually resolved -- fall back to looking it up as a
	// previously-set code-block variable, defaulting to "".
	if v := e.lookupCodeVar(expr, data); v != nil {
		return v
	}
	return ""
}

var controlHeaderRegex = regexp.MustCompile(`@if\s*\(([^)]+)\)\s*|@foreach\s*\(\s*var\s+(\w+)\s+in\s+(\w+)\s*\)\s*`)
var ifHeaderRegex = regexp.MustCompile(`@if\s*\(([^)]+)\)`)
var foreachHeaderRegex = regexp.MustCompile(`@foreach\s*\(\s*var\s+(\w+)\s+in\s+(\w+)\s*\)`)
var elseHeaderRegex = regexp.MustCompile(`^@?else\s*`)

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func findMatchingBrace(content string, openIdx int) (int, bool) {
	depth := 0
	for i := openIdx; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return -1, false
}

// convertCondition converts a C#-style boolean expression (using &&, ||, !,
// parens, and identifiers/keywords like IsPost, Form["x"], or bare local
// variables set in a @{ } code block) into a Go template boolean expression
// using the and/or/not functions, since Go templates have no &&/||/!
// operators. Without this, any @if using those operators (as opposed to a
// single bare condition) produces template source that fails to parse.
func convertCondition(cond string) string {
	tokens := tokenizeCondition(cond)
	pos := 0
	return parseConditionOr(tokens, &pos)
}

func tokenizeCondition(s string) []string {
	var tokens []string
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case isSpaceByte(c):
			i++
		case c == '(' || c == ')':
			tokens = append(tokens, string(c))
			i++
		case c == '&' && i+1 < len(s) && s[i+1] == '&':
			tokens = append(tokens, "&&")
			i += 2
		case c == '|' && i+1 < len(s) && s[i+1] == '|':
			tokens = append(tokens, "||")
			i += 2
		case c == '!':
			tokens = append(tokens, "!")
			i++
		default:
			// Atom: read until the next operator/paren/space at bracket
			// depth 0. Brackets are tracked so atoms like Form["a b"] read
			// as a single token.
			j := i
			depth := 0
			for j < len(s) {
				ch := s[j]
				if ch == '[' {
					depth++
				} else if ch == ']' {
					depth--
				} else if depth == 0 {
					if ch == '(' || ch == ')' || isSpaceByte(ch) {
						break
					}
					if ch == '&' && j+1 < len(s) && s[j+1] == '&' {
						break
					}
					if ch == '|' && j+1 < len(s) && s[j+1] == '|' {
						break
					}
					if ch == '!' {
						break
					}
				}
				j++
			}
			if j == i {
				j++ // avoid an infinite loop on an unexpected character
			}
			tokens = append(tokens, s[i:j])
			i = j
		}
	}
	return tokens
}

func parseConditionOr(tokens []string, pos *int) string {
	terms := []string{parseConditionAnd(tokens, pos)}
	for *pos < len(tokens) && tokens[*pos] == "||" {
		*pos++
		terms = append(terms, parseConditionAnd(tokens, pos))
	}
	if len(terms) == 1 {
		return terms[0]
	}
	return "(or " + strings.Join(terms, " ") + ")"
}

func parseConditionAnd(tokens []string, pos *int) string {
	terms := []string{parseConditionUnary(tokens, pos)}
	for *pos < len(tokens) && tokens[*pos] == "&&" {
		*pos++
		terms = append(terms, parseConditionUnary(tokens, pos))
	}
	if len(terms) == 1 {
		return terms[0]
	}
	return "(and " + strings.Join(terms, " ") + ")"
}

func parseConditionUnary(tokens []string, pos *int) string {
	if *pos < len(tokens) && tokens[*pos] == "!" {
		*pos++
		return "(not " + parseConditionUnary(tokens, pos) + ")"
	}
	return parseConditionPrimary(tokens, pos)
}

func parseConditionPrimary(tokens []string, pos *int) string {
	if *pos < len(tokens) && tokens[*pos] == "(" {
		*pos++
		inner := parseConditionOr(tokens, pos)
		if *pos < len(tokens) && tokens[*pos] == ")" {
			*pos++
		}
		return inner
	}
	if *pos >= len(tokens) {
		return "false"
	}
	atom := tokens[*pos]
	*pos++
	return convertAtom(atom)
}

var (
	atomIsPostRegex      = regexp.MustCompile(`^(?:Page\.)?[Ii]sPost$`)
	atomIsGetRegex       = regexp.MustCompile(`^(?:Page\.)?[Ii]sGet$`)
	atomFormRegex        = regexp.MustCompile(`^(?:Page\.)?Form\["([^"]+)"\]$`)
	atomQueryStringRegex = regexp.MustCompile(`^(?:Page\.)?QueryString\["([^"]+)"\]$`)
	atomSessionRegex     = regexp.MustCompile(`^(?:Page\.)?Session\["([^"]+)"\]$`)
	atomIdentifierRegex  = regexp.MustCompile(`^\w+$`)
)

// convertAtom converts a single condition operand into Go template syntax.
func convertAtom(atom string) string {
	atom = strings.TrimSpace(atom)
	switch {
	case atomIsPostRegex.MatchString(atom):
		return ".IsPost"
	case atomIsGetRegex.MatchString(atom):
		return ".IsGet"
	}
	if m := atomFormRegex.FindStringSubmatch(atom); m != nil {
		return `(index .Form "` + m[1] + `")`
	}
	if m := atomQueryStringRegex.FindStringSubmatch(atom); m != nil {
		return `(index .QueryString "` + m[1] + `")`
	}
	if m := atomSessionRegex.FindStringSubmatch(atom); m != nil {
		return `(index .Session "` + m[1] + `")`
	}
	if atomIdentifierRegex.MatchString(atom) {
		// A bare identifier is a local variable set in a @{ } code block.
		return `(Var "` + atom + `")`
	}
	// Unsupported construct (e.g. an arbitrary method call): fall back to
	// a literal false rather than emitting template source that fails to
	// parse at all.
	return "false"
}

// ternaryRegex matches `@(<lhs> == "<val>" ? "<true>" : "<false>")`, the
// common WebMatrix idiom for conditionally emitting an attribute like
// `selected` or `checked`.
var ternaryRegex = regexp.MustCompile(`@\(\s*([^=()]+?)\s*==\s*"([^"]*)"\s*\?\s*"([^"]*)"\s*:\s*"([^"]*)"\s*\)`)

// convertTernaryExpressions converts @(x == "y" ? "a" : "b") into an
// {{if eq ... }}a{{else}}b{{end}} block. This construct isn't a bare @word
// or an @if/@foreach block, so without this it passes through untouched
// and renders as literal, unparsed WebMatrix syntax in the page output.
func convertTernaryExpressions(content string) string {
	return ternaryRegex.ReplaceAllStringFunc(content, func(m string) string {
		parts := ternaryRegex.FindStringSubmatch(m)
		lhs, val, whenTrue, whenFalse := parts[1], parts[2], parts[3], parts[4]
		return `{{if eq ` + convertAtom(strings.TrimSpace(lhs)) + ` "` + val + `"}}` + whenTrue + `{{else}}` + whenFalse + `{{end}}`
	})
}

func findMatchingParen(content string, openIdx int) (int, bool) {
	depth := 0
	for i := openIdx; i < len(content); i++ {
		switch content[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return -1, false
}

func convertControlFlow(content string) string {
	var out strings.Builder
	pos := 0

	for pos < len(content) {
		loc := controlHeaderRegex.FindStringIndex(content[pos:])
		if loc == nil {
			out.WriteString(content[pos:])
			break
		}
		start := pos + loc[0]
		headerEnd := pos + loc[1]
		out.WriteString(content[pos:start])

		isIf := strings.HasPrefix(content[start:], "@if")

		j := headerEnd
		for j < len(content) && isSpaceByte(content[j]) {
			j++
		}
		if j >= len(content) || content[j] != '{' {
			out.WriteString(content[start:headerEnd])
			pos = headerEnd
			continue
		}

		bodyEnd, ok := findMatchingBrace(content, j)
		if !ok {
			out.WriteString(content[start:])
			pos = len(content)
			break
		}
		body := content[j+1 : bodyEnd]
		convertedBody := convertControlFlow(body)

		if isIf {
			m := ifHeaderRegex.FindStringSubmatch(content[start:headerEnd])
			out.WriteString("{{if " + convertCondition(m[1]) + "}}")
		} else {
			m := foreachHeaderRegex.FindStringSubmatch(content[start:headerEnd])
			out.WriteString(fmt.Sprintf(`{{range $index, $%s := (Var "%s")}}`, m[1], m[2]))
		}
		out.WriteString(convertedBody)

		pos = bodyEnd + 1

		if isIf {
			k := pos
			for k < len(content) && isSpaceByte(content[k]) {
				k++
			}
			if m := elseHeaderRegex.FindString(content[k:]); m != "" {
				k2 := k + len(m)
				for k2 < len(content) && isSpaceByte(content[k2]) {
					k2++
				}
				if k2 < len(content) && content[k2] == '{' {
					if elseBodyEnd, ok := findMatchingBrace(content, k2); ok {
						elseBody := content[k2+1 : elseBodyEnd]
						out.WriteString("{{else}}")
						out.WriteString(convertControlFlow(elseBody))
						pos = elseBodyEnd + 1
					}
				}
			}
		}

		out.WriteString("{{end}}")
	}

	return out.String()
}

// convertToGoTemplate converts WebMatrix @ syntax to Go template syntax.
// Fixed: Removed embedded processCodeBlocks call since it's now handled top-level.
func (e *WebMatrixEngine) convertToGoTemplate(content string) string {
	processed := content

	processed = strings.ReplaceAll(processed, "@@", "___AT_AT___")

	processed = regexp.MustCompile(`@Page\.Title`).ReplaceAllString(processed, `{{.Title}}`)
	processed = regexp.MustCompile(`@Page\.IsPost`).ReplaceAllString(processed, `{{.IsPost}}`)
	processed = regexp.MustCompile(`@Page\.IsGet`).ReplaceAllString(processed, `{{.IsGet}}`)
	processed = regexp.MustCompile(`@Page\.Form\["([^"]+)"\]`).ReplaceAllString(processed, `{{index .Form "$1"}}`)
	processed = regexp.MustCompile(`@Page\.QueryString\["([^"]+)"\]`).ReplaceAllString(processed, `{{index .QueryString "$1"}}`)
	processed = regexp.MustCompile(`@Page\.Session\["([^"]+)"\]`).ReplaceAllString(processed, `{{index .Session "$1"}}`)
	processed = regexp.MustCompile(`@Page\.UrlData\[([0-9]+)\]`).ReplaceAllString(processed, `{{index .UrlData $1}}`)
	processed = regexp.MustCompile(`@AppSettings\["([^"]+)"\]`).ReplaceAllString(processed, `{{AppSettings "$1"}}`)
	processed = regexp.MustCompile(`@ConnectionString\["([^"]+)"\]`).ReplaceAllString(processed, `{{ConnectionString "$1"}}`)
	processed = regexp.MustCompile(`@Html\.AntiForgeryToken\(\)`).ReplaceAllString(processed, `{{(Html).AntiForgeryToken}}`)
	processed = regexp.MustCompile(`@Database\.Query\(([^)]*)\)`).ReplaceAllString(processed, `{{(Database).Query $1}}`)
	processed = regexp.MustCompile(`@Database\.Open\(([^)]*)\)`).ReplaceAllString(processed, `{{(Database).Open $1}}`)
	processed = regexp.MustCompile(`@Database\.Execute\(([^)]*)\)`).ReplaceAllString(processed, `{{(Database).Execute $1}}`)
	processed = regexp.MustCompile(`@RenderBody\(\)`).ReplaceAllString(processed, `{{RenderBody .}}`)
	processed = regexp.MustCompile(`@RenderSection\(["']([^"']+)["']\)`).ReplaceAllString(processed, `{{RenderSection "$1" .}}`)
	processed = regexp.MustCompile(`@IsSectionDefined\(["']([^"']+)["']\)`).ReplaceAllString(processed, `{{IsSectionDefined "$1" .}}`)

	sectionRegex := regexp.MustCompile(`@section\s+(\w+)\s*{([^}]*)}`)
	processed = sectionRegex.ReplaceAllString(processed, `{{define "$1"}}$2{{end}}`)

	// @(x == "y" ? "a" : "b") ternary expressions (e.g. selected/checked
	// attributes) -- must run before the generic bare-@word pass below,
	// since it consumes the whole @( ... ) span itself.
	processed = convertTernaryExpressions(processed)

	// @(x != null ? x.Field : "fallback") -- e.g. a personalized greeting
	// that falls back to a default when the value isn't set.
	processed = convertNullTernaryExpressions(processed)

	// @DateTime.Now.ToString("<c# format>") -- must also run before the
	// generic bare-@word pass, which would otherwise match only "@DateTime"
	// and leave ".Now.ToString(...)" behind as literal, unrendered text.
	processed = convertDateTimeExpressions(processed)

	processed = convertControlFlow(processed)

	// Built-in helpers are consumed by the specific patterns above; any
	// occurrence reaching this point in one of these names wasn't in a
	// recognized shape (e.g. `@DateTime` with no `.Now.ToString(...)`) and
	// is left as-is rather than silently treated as a page variable lookup,
	// which would otherwise render as empty and hide the real problem.
	processed = regexp.MustCompile(`@(\w+)`).ReplaceAllStringFunc(processed, func(m string) string {
		word := m[1:]
		if reservedFuncNames[word] {
			return m
		}
		return `{{Var "` + word + `"}}`
	})

	processed = strings.ReplaceAll(processed, "___AT_AT___", "@")

	return processed
}

var reservedFuncNames = map[string]bool{
	"RenderBody": true, "RenderSection": true, "IsSectionDefined": true,
	"DateTime": true, "AppSettings": true, "ConnectionString": true,
	"Html": true, "Database": true, "Page": true, "Raw": true, "Var": true,
}

// csharpDateTimeRegex matches @DateTime.Now.ToString("<c# format>"), the
// WebMatrix idiom for formatting the current time.
var csharpDateTimeRegex = regexp.MustCompile(`@DateTime\.Now\.ToString\("([^"]+)"\)`)

func convertDateTimeExpressions(content string) string {
	return csharpDateTimeRegex.ReplaceAllStringFunc(content, func(m string) string {
		parts := csharpDateTimeRegex.FindStringSubmatch(m)
		return `{{(DateTime).Format "` + csharpDateFormatToGo(parts[1]) + `"}}`
	})
}

// csharpDateFormatToGo converts a common subset of C#/WebMatrix date format
// tokens into Go's reference-time format. Not exhaustive -- covers the
// tokens likely to appear in practice (yyyy, MM, dd, HH, mm, ss).
func csharpDateFormatToGo(f string) string {
	replacer := strings.NewReplacer(
		"yyyy", "2006",
		"MM", "01",
		"dd", "02",
		"HH", "15",
		"mm", "04",
		"ss", "05",
	)
	return replacer.Replace(f)
}

// ternaryNullRegex matches `@(<var> != null ? <expr> : "<fallback>")`, used
// e.g. for a personalized greeting that falls back to a default when a
// value isn't set.
var ternaryNullRegex = regexp.MustCompile(`@\(\s*(\w+)\s*!=\s*null\s*\?\s*([^:]+?)\s*:\s*"([^"]*)"\s*\)`)

func convertNullTernaryExpressions(content string) string {
	return ternaryNullRegex.ReplaceAllStringFunc(content, func(m string) string {
		parts := ternaryNullRegex.FindStringSubmatch(m)
		varName, trueExpr, falseStr := parts[1], strings.TrimSpace(parts[2]), parts[3]
		return `{{if (Var "` + varName + `")}}` + convertDottedExpr(trueExpr) + `{{else}}` + falseStr + `{{end}}`
	})
}

// convertDottedExpr converts a simple "name" or "name.Field" operand into
// Go template syntax: a bare page variable lookup, or a field access on one.
func convertDottedExpr(expr string) string {
	if idx := strings.Index(expr, "."); idx != -1 {
		base, field := expr[:idx], expr[idx+1:]
		return `{{(Var "` + base + `").` + field + `}}`
	}
	return `{{Var "` + expr + `"}}`
}

// Render renders a page template with optional layout
func (e *WebMatrixEngine) Render(w io.Writer, name string, data interface{}) error {
	if pageData, ok := data.(*PageData); ok {
		e.pageData = pageData
		// Render is called with the real http.ResponseWriter in normal
		// request handling; capture it so template helpers (e.g.
		// AntiForgeryToken) can set cookies. Falls back to nil (e.g. in
		// tests rendering to a bytes.Buffer), which AntiForgeryToken
		// already handles.
		if rw, ok := w.(http.ResponseWriter); ok {
			pageData.Response = rw
		}
	}

	tmpl, err := e.loadTemplate(name, data)
	if err != nil {
		return err
	}

	var bodyBuf bytes.Buffer
	if err := tmpl.Execute(&bodyBuf, data); err != nil {
		return err
	}

	if pageData, ok := data.(*PageData); ok && pageData.Layout != "" {
		pageData.Body = template.HTML(bodyBuf.String())
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

	_, err = w.Write(bodyBuf.Bytes())
	return err
}

// RenderPartial renders a partial template
func (e *WebMatrixEngine) RenderPartial(w io.Writer, name string, data interface{}) error {
	tmpl, err := e.loadPartial(name, data)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, data)
}

// loadTemplate loads a page template from the pages directory
// Fixed: Re-runs executeCodeBlocks on every call, cache hit or cache miss.
func (e *WebMatrixEngine) loadTemplate(name string, data interface{}) (*template.Template, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if item, exists := e.templates[name]; exists {
		// Cache Hit: Replay the extracted code blocks onto the fresh request's context
		e.executeCodeBlocks(item.codeBlocks, data)
		return item.tmpl, nil
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

	// Cache Miss: Parse code blocks separately from Go syntax substitution conversion
	strippedContent, codeBlocks := e.parseCodeBlocks(string(content))
	e.executeCodeBlocks(codeBlocks, data)

	converted := e.convertToGoTemplate(strippedContent)
	tmpl := template.New(name).Funcs(e.funcMap)
	tmpl, err = tmpl.Parse(converted)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	// Store both compiled layout and parsed raw code metadata blocks
	e.templates[name] = templateCacheItem{
		tmpl:       tmpl,
		codeBlocks: codeBlocks,
	}
	return tmpl, nil
}

// loadLayout loads a layout template from the layouts directory
func (e *WebMatrixEngine) loadLayout(name string, data interface{}) (*template.Template, error) {
	layoutName := strings.TrimPrefix(name, "~/")
	layoutName = strings.TrimPrefix(layoutName, "templates/layouts/")

	layoutPath := filepath.Join(e.layoutsPath, layoutName+".gshtml")
	if _, err := os.Stat(layoutPath); os.IsNotExist(err) {
		layoutPath = filepath.Join(e.layoutsPath, layoutName+".gohtml")
		if _, err := os.Stat(layoutPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("layout %s not found", layoutName)
		}
	}

	content, err := os.ReadFile(layoutPath)
	if err != nil {
		return nil, err
	}

	// Strip block arrays and convert layouts as standard text
	strippedContent, blocks := e.parseCodeBlocks(string(content))
	e.executeCodeBlocks(blocks, data)
	converted := e.convertToGoTemplate(strippedContent)
	return template.New(layoutName).Funcs(e.funcMap).Parse(converted)
}

// loadPartial loads a partial template from the partials directory
func (e *WebMatrixEngine) loadPartial(name string, data interface{}) (*template.Template, error) {
	partialPath := filepath.Join(e.partialsPath, name+".gshtml")
	if _, err := os.Stat(partialPath); os.IsNotExist(err) {
		partialPath = filepath.Join(e.partialsPath, name+".gohtml")
		if _, err := os.Stat(partialPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("partial %s not found", name)
		}
	}

	content, err := os.ReadFile(partialPath)
	if err != nil {
		return nil, err
	}

	strippedContent, blocks := e.parseCodeBlocks(string(content))
	e.executeCodeBlocks(blocks, data)
	converted := e.convertToGoTemplate(strippedContent)
	return template.New(name).Funcs(e.funcMap).Parse(converted)
}

// AddFunc adds a custom template function
func (e *WebMatrixEngine) AddFunc(name string, fn interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.funcMap[name] = fn
}
