package html

import (
    "fmt"
    "html/template"
    "strings"
    "time"
)

type HTMLHelper struct {
    csrfToken string
}

func NewHelper(csrfToken string) *HTMLHelper {
    return &HTMLHelper{
        csrfToken: csrfToken,
    }
}

// Tag builder functions
func Tag(name string, attrs map[string]string, content ...interface{}) template.HTML {
    attrStr := ""
    for key, value := range attrs {
        attrStr += fmt.Sprintf(" %s=\"%s\"", key, template.HTMLEscapeString(value))
    }

    var contentStr string
    for _, c := range content {
        switch v := c.(type) {
        case string:
            contentStr += v
        case template.HTML:
            contentStr += string(v)
        default:
            contentStr += fmt.Sprintf("%v", v)
        }
    }

    if contentStr == "" {
        return template.HTML(fmt.Sprintf("<%s%s />", name, attrStr))
    }
    return template.HTML(fmt.Sprintf("<%s%s>%s</%s>", name, attrStr, contentStr, name))
}

// Form helpers
func (h *HTMLHelper) Form(method, action string, content ...interface{}) template.HTML {
    attrs := map[string]string{
        "method": method,
        "action": action,
    }
    if method == "POST" || method == "post" {
        // Add CSRF token
        hidden := h.Hidden("csrf_token", h.csrfToken)
        content = append(content, hidden)
    }
    return Tag("form", attrs, content...)
}

func (h *HTMLHelper) Input(typ, name, value string, attrs ...map[string]string) template.HTML {
    attrMap := map[string]string{
        "type":  typ,
        "name":  name,
        "value": value,
    }
    for _, extra := range attrs {
        for k, v := range extra {
            attrMap[k] = v
        }
    }
    return Tag("input", attrMap)
}

func (h *HTMLHelper) Textbox(name, value string, attrs ...map[string]string) template.HTML {
    return h.Input("text", name, value, attrs...)
}

func (h *HTMLHelper) Password(name string, attrs ...map[string]string) template.HTML {
    return h.Input("password", name, "", attrs...)
}

func (h *HTMLHelper) Hidden(name, value string) template.HTML {
    return h.Input("hidden", name, value)
}

func (h *HTMLHelper) Email(name, value string, attrs ...map[string]string) template.HTML {
    return h.Input("email", name, value, attrs...)
}

func (h *HTMLHelper) Phone(name, value string, attrs ...map[string]string) template.HTML {
    return h.Input("tel", name, value, attrs...)
}

func (h *HTMLHelper) Number(name string, value float64, attrs ...map[string]string) template.HTML {
    strValue := ""
    if value != 0 {
        strValue = fmt.Sprintf("%v", value)
    }
    return h.Input("number", name, strValue, attrs...)
}

func (h *HTMLHelper) Money(name string, value float64, attrs ...map[string]string) template.HTML {
    strValue := ""
    if value != 0 {
        strValue = fmt.Sprintf("%.2f", value)
    }
    return h.Input("number", name, strValue, append(attrs, map[string]string{"step": "0.01"})...)
}

func (h *HTMLHelper) Date(name string, value time.Time, attrs ...map[string]string) template.HTML {
    strValue := ""
    if !value.IsZero() {
        strValue = value.Format("2006-01-02")
    }
    return h.Input("date", name, strValue, attrs...)
}

func (h *HTMLHelper) DateTime(name string, value time.Time, attrs ...map[string]string) template.HTML {
    strValue := ""
    if !value.IsZero() {
        strValue = value.Format("2006-01-02T15:04")
    }
    return h.Input("datetime-local", name, strValue, attrs...)
}

func (h *HTMLHelper) Textarea(name, value string, attrs ...map[string]string) template.HTML {
    attrMap := map[string]string{
        "name": name,
    }
    for _, extra := range attrs {
        for k, v := range extra {
            attrMap[k] = v
        }
    }
    attrStr := ""
    for key, val := range attrMap {
        attrStr += fmt.Sprintf(" %s=\"%s\"", key, template.HTMLEscapeString(val))
    }
    return template.HTML(fmt.Sprintf("<textarea%s>%s</textarea>", attrStr, template.HTMLEscapeString(value)))
}

func (h *HTMLHelper) Dropdown(name string, options map[string]string, selected string, attrs ...map[string]string) template.HTML {
    attrMap := map[string]string{
        "name": name,
    }
    for _, extra := range attrs {
        for k, v := range extra {
            attrMap[k] = v
        }
    }

    var opts strings.Builder
    for value, label := range options {
        selectedAttr := ""
        if value == selected {
            selectedAttr = " selected"
        }
        opts.WriteString(fmt.Sprintf("<option value=\"%s\"%s>%s</option>", 
            template.HTMLEscapeString(value), selectedAttr, template.HTMLEscapeString(label)))
    }

    attrStr := ""
    for key, val := range attrMap {
        attrStr += fmt.Sprintf(" %s=\"%s\"", key, template.HTMLEscapeString(val))
    }

    return template.HTML(fmt.Sprintf("<select%s>%s</select>", attrStr, opts.String()))
}

func (h *HTMLHelper) Checkbox(name string, checked bool, value string, attrs ...map[string]string) template.HTML {
    attrMap := map[string]string{
        "type":  "checkbox",
        "name":  name,
        "value": value,
    }
    if checked {
        attrMap["checked"] = "checked"
    }
    for _, extra := range attrs {
        for k, v := range extra {
            attrMap[k] = v
        }
    }
    return Tag("input", attrMap)
}

func (h *HTMLHelper) Radio(name, value string, checked bool, attrs ...map[string]string) template.HTML {
    attrMap := map[string]string{
        "type":  "radio",
        "name":  name,
        "value": value,
    }
    if checked {
        attrMap["checked"] = "checked"
    }
    for _, extra := range attrs {
        for k, v := range extra {
            attrMap[k] = v
        }
    }
    return Tag("input", attrMap)
}

func (h *HTMLHelper) Button(label string, typ string, attrs ...map[string]string) template.HTML {
    attrMap := map[string]string{
        "type": typ,
    }
    for _, extra := range attrs {
        for k, v := range extra {
            attrMap[k] = v
        }
    }
    return Tag("button", attrMap, template.HTML(label))
}

// Bootstrap helpers
func (h *HTMLHelper) Alert(message, alertType string, dismissible bool) template.HTML {
    classes := fmt.Sprintf("alert alert-%s", alertType)
    if dismissible {
        classes += " alert-dismissible fade show"
    }

    closeBtn := ""
    if dismissible {
        closeBtn = `<button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>`
    }

    return template.HTML(fmt.Sprintf(`<div class="%s" role="alert">%s%s</div>`, classes, message, closeBtn))
}

func (h *HTMLHelper) Card(title string, content template.HTML, footer ...template.HTML) template.HTML {
    var footerHTML string
    if len(footer) > 0 {
        footerHTML = fmt.Sprintf(`<div class="card-footer">%s</div>`, footer[0])
    }

    return template.HTML(fmt.Sprintf(`
<div class="card">
    <div class="card-header">%s</div>
    <div class="card-body">%s</div>
    %s
</div>`, title, content, footerHTML))
}

func (h *HTMLHelper) Badge(text, bgColor string) template.HTML {
    return template.HTML(fmt.Sprintf(`<span class="badge bg-%s">%s</span>`, bgColor, text))
}

func (h *HTMLHelper) Breadcrumb(items ...template.HTML) template.HTML {
    var itemsHTML strings.Builder
    itemsHTML.WriteString(`<nav aria-label="breadcrumb"><ol class="breadcrumb">`)
    for i, item := range items {
        if i == len(items)-1 {
            itemsHTML.WriteString(fmt.Sprintf(`<li class="breadcrumb-item active" aria-current="page">%s</li>`, item))
        } else {
            itemsHTML.WriteString(fmt.Sprintf(`<li class="breadcrumb-item">%s</li>`, item))
        }
    }
    itemsHTML.WriteString(`</ol></nav>`)
    return template.HTML(itemsHTML.String())
}

func (h *HTMLHelper) Pagination(currentPage, totalPages int, urlFunc func(page int) string) template.HTML {
    if totalPages <= 1 {
        return ""
    }

    var html strings.Builder
    html.WriteString(`<nav aria-label="Page navigation"><ul class="pagination justify-content-center">`)

    // Previous
    if currentPage > 1 {
        html.WriteString(fmt.Sprintf(`<li class="page-item"><a class="page-link" href="%s" hx-get="%s" hx-target="#content" hx-swap="outerHTML">&laquo;</a></li>`, 
            urlFunc(currentPage-1), urlFunc(currentPage-1)))
    } else {
        html.WriteString(`<li class="page-item disabled"><span class="page-link">&laquo;</span></li>`)
    }

    // Pages
    start := currentPage - 2
    if start < 1 {
        start = 1
    }
    end := currentPage + 2
    if end > totalPages {
        end = totalPages
    }

    if start > 1 {
        html.WriteString(fmt.Sprintf(`<li class="page-item"><a class="page-link" href="%s" hx-get="%s" hx-target="#content" hx-swap="outerHTML">1</a></li>`, 
            urlFunc(1), urlFunc(1)))
        if start > 2 {
            html.WriteString(`<li class="page-item disabled"><span class="page-link">...</span></li>`)
        }
    }

    for i := start; i <= end; i++ {
        if i == currentPage {
            html.WriteString(fmt.Sprintf(`<li class="page-item active"><span class="page-link">%d</span></li>`, i))
        } else {
            html.WriteString(fmt.Sprintf(`<li class="page-item"><a class="page-link" href="%s" hx-get="%s" hx-target="#content" hx-swap="outerHTML">%d</a></li>`, 
                urlFunc(i), urlFunc(i), i))
        }
    }

    if end < totalPages {
        if end < totalPages-1 {
            html.WriteString(`<li class="page-item disabled"><span class="page-link">...</span></li>`)
        }
        html.WriteString(fmt.Sprintf(`<li class="page-item"><a class="page-link" href="%s" hx-get="%s" hx-target="#content" hx-swap="outerHTML">%d</a></li>`, 
            urlFunc(totalPages), urlFunc(totalPages), totalPages))
    }

    // Next
    if currentPage < totalPages {
        html.WriteString(fmt.Sprintf(`<li class="page-item"><a class="page-link" href="%s" hx-get="%s" hx-target="#content" hx-swap="outerHTML">&raquo;</a></li>`, 
            urlFunc(currentPage+1), urlFunc(currentPage+1)))
    } else {
        html.WriteString(`<li class="page-item disabled"><span class="page-link">&raquo;</span></li>`)
    }

    html.WriteString(`</ul></nav>`)
    return template.HTML(html.String())
}

// HTMX helpers
func (h *HTMLHelper) HXGet(url, target, swap string) map[string]string {
    attrs := map[string]string{
        "hx-get": url,
    }
    if target != "" {
        attrs["hx-target"] = target
    }
    if swap != "" {
        attrs["hx-swap"] = swap
    }
    return attrs
}

func (h *HTMLHelper) HXPost(url, target, swap string) map[string]string {
    attrs := map[string]string{
        "hx-post": url,
    }
    if target != "" {
        attrs["hx-target"] = target
    }
    if swap != "" {
        attrs["hx-swap"] = swap
    }
    return attrs
}

func (h *HTMLHelper) HXPut(url, target, swap string) map[string]string {
    attrs := map[string]string{
        "hx-put": url,
    }
    if target != "" {
        attrs["hx-target"] = target
    }
    if swap != "" {
        attrs["hx-swap"] = swap
    }
    return attrs
}

func (h *HTMLHelper) HXDelete(url, target, swap string) map[string]string {
    attrs := map[string]string{
        "hx-delete": url,
    }
    if target != "" {
        attrs["hx-target"] = target
    }
    if swap != "" {
        attrs["hx-swap"] = swap
    }
    return attrs
}

func (h *HTMLHelper) HXTrigger(event string) map[string]string {
    return map[string]string{
        "hx-trigger": event,
    }
}

func (h *HTMLHelper) HXIndicator(selector string) map[string]string {
    return map[string]string{
        "hx-indicator": selector,
    }
}

func (h *HTMLHelper) HXConfirm(message string) map[string]string {
    return map[string]string{
        "hx-confirm": message,
    }
}

// Link helpers
func (h *HTMLHelper) Link(href, text string, attrs ...map[string]string) template.HTML {
    attrMap := map[string]string{
        "href": href,
    }
    for _, extra := range attrs {
        for k, v := range extra {
            attrMap[k] = v
        }
    }
    return Tag("a", attrMap, template.HTML(text))
}

// Icon helpers
func (h *HTMLHelper) Icon(iconName string) template.HTML {
    return template.HTML(fmt.Sprintf(`<i class="bi bi-%s"></i>`, iconName))
}