package utils

import (
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "net/url"
    "regexp"
    "strconv"
    "strings"
    "time"
)

// String utilities
func Slugify(s string) string {
    s = strings.ToLower(s)
    s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
    s = strings.Trim(s, "-")
    return s
}

func Truncate(s string, length int) string {
    if len(s) <= length {
        return s
    }
    return s[:length] + "..."
}

func IsEmpty(s string) bool {
    return strings.TrimSpace(s) == ""
}

func Contains(s, substr string) bool {
    return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// Number utilities
func ToFloat(s string) float64 {
    val, _ := strconv.ParseFloat(s, 64)
    return val
}

func ToInt(s string) int {
    val, _ := strconv.Atoi(s)
    return val
}

func FormatMoney(amount float64) string {
    return fmt.Sprintf("$%.2f", amount)
}

// Date utilities
func FormatDate(t time.Time) string {
    return t.Format("2006-01-02")
}

func FormatDateTime(t time.Time) string {
    return t.Format("2006-01-02 15:04:05")
}

func ParseDate(s string) (time.Time, error) {
    return time.Parse("2006-01-02", s)
}

// Random utilities
func GenerateRandomString(length int) string {
    b := make([]byte, length)
    rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)[:length]
}

func GenerateRandomNumber(min, max int) int {
    b := make([]byte, 8)
    rand.Read(b)
    val := int(b[0])<<56 | int(b[1])<<48 | int(b[2])<<40 | int(b[3])<<32 |
        int(b[4])<<24 | int(b[5])<<16 | int(b[6])<<8 | int(b[7])
    if val < 0 {
        val = -val
    }
    return min + val%(max-min+1)
}

// URL utilities
func BuildURL(base string, params map[string]string) string {
    if len(params) == 0 {
        return base
    }

    u, err := url.Parse(base)
    if err != nil {
        return base
    }

    q := u.Query()
    for key, value := range params {
        q.Set(key, value)
    }
    u.RawQuery = q.Encode()
    return u.String()
}

// Validation utilities
func IsEmail(email string) bool {
    regex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    return regex.MatchString(email)
}

func IsPhone(phone string) bool {
    regex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
    return regex.MatchString(phone)
}

func IsURL(urlStr string) bool {
    u, err := url.Parse(urlStr)
    return err == nil && u.Scheme != "" && u.Host != ""
}

// Slice utilities
func InSlice(str string, slice []string) bool {
    for _, item := range slice {
        if item == str {
            return true
        }
    }
    return false
}

func UniqueSlice(slice []string) []string {
    keys := make(map[string]bool)
    result := make([]string, 0)

    for _, item := range slice {
        if !keys[item] {
            keys[item] = true
            result = append(result, item)
        }
    }

    return result
}

// Map utilities
func GetMapString(m map[string]interface{}, key string, defaultValue string) string {
    if val, ok := m[key]; ok {
        if str, ok := val.(string); ok {
            return str
        }
    }
    return defaultValue
}

func GetMapInt(m map[string]interface{}, key string, defaultValue int) int {
    if val, ok := m[key]; ok {
        if num, ok := val.(int); ok {
            return num
        }
        if num, ok := val.(float64); ok {
            return int(num)
        }
    }
    return defaultValue
}
