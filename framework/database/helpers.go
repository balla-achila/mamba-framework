package database

import (
    "fmt"
    "strings"
)

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

func BuildOrderClause(sort, order string) string {
    if sort == "" {
        return ""
    }
    if order == "" {
        order = "ASC"
    }
    return fmt.Sprintf("ORDER BY %s %s", sort, strings.ToUpper(order))
}

func BuildLimitClause(limit int) string {
    if limit <= 0 {
        return ""
    }
    return fmt.Sprintf("LIMIT %d", limit)
}

func BuildOffsetClause(offset int) string {
    if offset <= 0 {
        return ""
    }
    return fmt.Sprintf("OFFSET %d", offset)
}

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
