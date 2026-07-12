package validation

import (
    "fmt"
    "regexp"
    "strconv"
    "strings"
    "time"
)

type Rule interface {
    Validate(data map[string]interface{}) error
    GetMessage() string
}

type Validator struct {
    rules  []Rule
    errors []string
}

func New() *Validator {
    return &Validator{
        rules:  make([]Rule, 0),
        errors: make([]string, 0),
    }
}

func (v *Validator) Validate(data map[string]interface{}) bool {
    v.errors = make([]string, 0)
    valid := true

    for _, rule := range v.rules {
        if err := rule.Validate(data); err != nil {
            v.errors = append(v.errors, rule.GetMessage())
            valid = false
        }
    }

    return valid
}

func (v *Validator) AddRule(rule Rule) {
    v.rules = append(v.rules, rule)
}

func (v *Validator) Errors() []string {
    return v.errors
}

// Required Rule
type RequiredRule struct {
    Field   string
    Message string
}

func (r *RequiredRule) Validate(data map[string]interface{}) error {
    value, exists := data[r.Field]
    if !exists || value == nil || value == "" {
        return fmt.Errorf("field %s is required", r.Field)
    }
    if str, ok := value.(string); ok && strings.TrimSpace(str) == "" {
        return fmt.Errorf("field %s is required", r.Field)
    }
    return nil
}

func (r *RequiredRule) GetMessage() string {
    if r.Message != "" {
        return r.Message
    }
    return fmt.Sprintf("%s is required", r.Field)
}

// Email Rule
type EmailRule struct {
    Field   string
    Message string
}

func (r *EmailRule) Validate(data map[string]interface{}) error {
    value, ok := data[r.Field].(string)
    if !ok {
        return fmt.Errorf("field %s must be a string", r.Field)
    }

    emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    if !emailRegex.MatchString(value) {
        return fmt.Errorf("field %s must be a valid email address", r.Field)
    }
    return nil
}

func (r *EmailRule) GetMessage() string {
    if r.Message != "" {
        return r.Message
    }
    return fmt.Sprintf("%s must be a valid email address", r.Field)
}

// Phone Rule
type PhoneRule struct {
    Field   string
    Message string
}

func (r *PhoneRule) Validate(data map[string]interface{}) error {
    value, ok := data[r.Field].(string)
    if !ok {
        return fmt.Errorf("field %s must be a string", r.Field)
    }

    phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
    if !phoneRegex.MatchString(value) {
        return fmt.Errorf("field %s must be a valid phone number", r.Field)
    }
    return nil
}

func (r *PhoneRule) GetMessage() string {
    if r.Message != "" {
        return r.Message
    }
    return fmt.Sprintf("%s must be a valid phone number", r.Field)
}

// Numeric Rule
type NumericRule struct {
    Field   string
    Message string
}

func (r *NumericRule) Validate(data map[string]interface{}) error {
    value := data[r.Field]
    switch v := value.(type) {
    case int, int8, int16, int32, int64, float32, float64:
        return nil
    case string:
        if _, err := strconv.ParseFloat(v, 64); err != nil {
            return fmt.Errorf("field %s must be numeric", r.Field)
        }
        return nil
    default:
        return fmt.Errorf("field %s must be numeric", r.Field)
    }
}

func (r *NumericRule) GetMessage() string {
    if r.Message != "" {
        return r.Message
    }
    return fmt.Sprintf("%s must be numeric", r.Field)
}

// Length Rule
type LengthRule struct {
    Field   string
    Min     int
    Max     int
    Message string
}

func (r *LengthRule) Validate(data map[string]interface{}) error {
    value, ok := data[r.Field].(string)
    if !ok {
        return fmt.Errorf("field %s must be a string", r.Field)
    }

    length := len(value)
    if r.Min > 0 && length < r.Min {
        return fmt.Errorf("field %s must be at least %d characters", r.Field, r.Min)
    }
    if r.Max > 0 && length > r.Max {
        return fmt.Errorf("field %s must be at most %d characters", r.Field, r.Max)
    }
    return nil
}

func (r *LengthRule) GetMessage() string {
    if r.Message != "" {
        return r.Message
    }
    if r.Min > 0 && r.Max > 0 {
        return fmt.Sprintf("%s must be between %d and %d characters", r.Field, r.Min, r.Max)
    }
    if r.Min > 0 {
        return fmt.Sprintf("%s must be at least %d characters", r.Field, r.Min)
    }
    if r.Max > 0 {
        return fmt.Sprintf("%s must be at most %d characters", r.Field, r.Max)
    }
    return fmt.Sprintf("%s has invalid length", r.Field)
}

// Range Rule
type RangeRule struct {
    Field   string
    Min     float64
    Max     float64
    Message string
}

func (r *RangeRule) Validate(data map[string]interface{}) error {
    var num float64
    value := data[r.Field]
    switch v := value.(type) {
    case int:
        num = float64(v)
    case int8:
        num = float64(v)
    case int16:
        num = float64(v)
    case int32:
        num = float64(v)
    case int64:
        num = float64(v)
    case float32:
        num = float64(v)
    case float64:
        num = v
    case string:
        var err error
        num, err = strconv.ParseFloat(v, 64)
        if err != nil {
            return fmt.Errorf("field %s must be numeric", r.Field)
        }
    default:
        return fmt.Errorf("field %s must be numeric", r.Field)
    }

    if r.Min > 0 && num < r.Min {
        return fmt.Errorf("field %s must be at least %v", r.Field, r.Min)
    }
    if r.Max > 0 && num > r.Max {
        return fmt.Errorf("field %s must be at most %v", r.Field, r.Max)
    }
    return nil
}

func (r *RangeRule) GetMessage() string {
    if r.Message != "" {
        return r.Message
    }
    if r.Min > 0 && r.Max > 0 {
        return fmt.Sprintf("%s must be between %v and %v", r.Field, r.Min, r.Max)
    }
    if r.Min > 0 {
        return fmt.Sprintf("%s must be at least %v", r.Field, r.Min)
    }
    if r.Max > 0 {
        return fmt.Sprintf("%s must be at most %v", r.Field, r.Max)
    }
    return fmt.Sprintf("%s has invalid range", r.Field)
}

// Date Rule
type DateRule struct {
    Field   string
    Format  string
    Message string
}

func (r *DateRule) Validate(data map[string]interface{}) error {
    value, ok := data[r.Field].(string)
    if !ok {
        return fmt.Errorf("field %s must be a string", r.Field)
    }

    format := r.Format
    if format == "" {
        format = "2006-01-02"
    }

    _, err := time.Parse(format, value)
    if err != nil {
        return fmt.Errorf("field %s must be a valid date in format %s", r.Field, format)
    }
    return nil
}

func (r *DateRule) GetMessage() string {
    if r.Message != "" {
        return r.Message
    }
    format := r.Format
    if format == "" {
        format = "YYYY-MM-DD"
    }
    return fmt.Sprintf("%s must be a valid date in format %s", r.Field, format)
}

// Regex Rule
type RegexRule struct {
    Field   string
    Pattern string
    Message string
}

func (r *RegexRule) Validate(data map[string]interface{}) error {
    value, ok := data[r.Field].(string)
    if !ok {
        return fmt.Errorf("field %s must be a string", r.Field)
    }

    regex := regexp.MustCompile(r.Pattern)
    if !regex.MatchString(value) {
        return fmt.Errorf("field %s does not match required pattern", r.Field)
    }
    return nil
}

func (r *RegexRule) GetMessage() string {
    if r.Message != "" {
        return r.Message
    }
    return fmt.Sprintf("%s does not match required pattern", r.Field)
}

// Custom Rule
type CustomRule struct {
    Field        string
    ValidateFunc func(interface{}) error
    Message      string
}

func (r *CustomRule) Validate(data map[string]interface{}) error {
    return r.ValidateFunc(data[r.Field])
}

func (r *CustomRule) GetMessage() string {
    if r.Message != "" {
        return r.Message
    }
    return fmt.Sprintf("%s validation failed", r.Field)
}
