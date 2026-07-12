package validation

import (
    "fmt"
    "regexp"
    "strconv"
    "strings"
    "time"
)

type Rule interface {
    Validate(value interface{}) error
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