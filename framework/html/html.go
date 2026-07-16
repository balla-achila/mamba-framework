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

// Tag builder
func Tag(name string, attrs map[string]string, content ...interface{}) template.HTML {
	attrStr := ""
	for key, value := range attrs {
		attrStr += fmt.Sprintf(" %s=\"%s\"", key, template.HTMLEscapeString(value))
	}

	var contentStr string
	for _, c := range content {
		switch v := c.(type) {
		case string:
			// Plain strings are untrusted text and must be escaped; only
			// template.HTML values are treated as pre-trusted raw markup.
			contentStr += template.HTMLEscapeString(v)
		case template.HTML:
			contentStr += string(v)
		default:
			contentStr += template.HTMLEscapeString(fmt.Sprintf("%v", v))
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
	return Tag("button", attrMap, label)
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

	return template.HTML(fmt.Sprintf(`<div class="%s" role="alert">%s%s</div>`, classes, template.HTMLEscapeString(message), closeBtn))
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
</div>`, template.HTMLEscapeString(title), content, footerHTML))
}

func (h *HTMLHelper) Badge(text, bgColor string) template.HTML {
	return template.HTML(fmt.Sprintf(`<span class="badge bg-%s">%s</span>`, bgColor, text))
}

func (h *HTMLHelper) Link(href, text string, attrs ...map[string]string) template.HTML {
	attrMap := map[string]string{
		"href": href,
	}
	for _, extra := range attrs {
		for k, v := range extra {
			attrMap[k] = v
		}
	}
	return Tag("a", attrMap, text)
}

func (h *HTMLHelper) Icon(iconName string) template.HTML {
	return template.HTML(fmt.Sprintf(`<i class="bi bi-%s"></i>`, iconName))
}
