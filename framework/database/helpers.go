package database

import (
    "context"
    "fmt"
    "strings"
)

// BuildWhereClause creates a WHERE clause with named parameters
func BuildWhereClause(conditions map[string]interface{}) (string, []interface{}, error) {
    if len(conditions) == 0 {
        return "", nil, nil
    }

    clauses := make([]string, 0, len(conditions))
    args := make([]interface{}, 0, len(conditions))
    i := 1

    for key, value := range conditions {
        if value == nil {
            clauses = append(clauses, fmt.Sprintf("%s IS NULL", key))
        } else {
            clauses = append(clauses, fmt.Sprintf("%s = $%d", key, i))
            args = append(args, value)
            i++
        }
    }

    return strings.Join(clauses, " AND "), args, nil
}

// BuildOrderClause creates an ORDER BY clause
func BuildOrderClause(sort, order string) string {
    if sort == "" {
        return ""
    }
    if order == "" {
        order = "ASC"
    }
    return fmt.Sprintf("ORDER BY %s %s", sort, strings.ToUpper(order))
}

// BuildLimitClause creates a LIMIT clause
func BuildLimitClause(limit int) string {
    if limit <= 0 {
        return ""
    }
    return fmt.Sprintf("LIMIT %d", limit)
}

// BuildOffsetClause creates an OFFSET clause
func BuildOffsetClause(offset int) string {
    if offset <= 0 {
        return ""
    }
    return fmt.Sprintf("OFFSET %d", offset)
}

// BuildPaginationClause creates pagination clauses
func BuildPaginationClause(page, perPage int) (string, string) {
    if page <= 0 {
        page = 1
    }
    if perPage <= 0 {
        perPage = 20
    }
    offset := (page - 1) * perPage
    return BuildLimitClause(perPage), BuildOffsetClause(offset)
}

// NamedParamsToPositional converts named parameters to positional
func NamedParamsToPositional(query string, params map[string]interface{}) (string, []interface{}) {
    args := make([]interface{}, 0, len(params))
    i := 1

    for key, value := range params {
        placeholder := fmt.Sprintf(":%s", key)
        query = strings.ReplaceAll(query, placeholder, fmt.Sprintf("$%d", i))
        args = append(args, value)
        i++
    }

    return query, args
}

// InClause creates an IN clause with placeholders
func InClause(column string, values []interface{}) (string, []interface{}) {
    if len(values) == 0 {
        return "1=0", nil
    }

    placeholders := make([]string, len(values))
    for i := range values {
        placeholders[i] = fmt.Sprintf("$%d", i+1)
    }

    return fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ", ")), values
}
