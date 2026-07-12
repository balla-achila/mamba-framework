package database

import (
    "fmt"
    "strings"
)

type QueryBuilder struct {
    table   string
    columns []string
    where   []string
    args    []interface{}
    order   string
    limit   int
    offset  int
    joins   []string
    groupBy []string
    having  []string
}

func NewQueryBuilder(table string) *QueryBuilder {
    return &QueryBuilder{
        table: table,
        where: make([]string, 0),
        args:  make([]interface{}, 0),
        joins: make([]string, 0),
        groupBy: make([]string, 0),
        having: make([]string, 0),
    }
}

func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
    qb.columns = columns
    return qb
}

func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
    if len(qb.where) == 0 {
        qb.where = append(qb.where, condition)
    } else {
        qb.where = append(qb.where, fmt.Sprintf("AND %s", condition))
    }
    qb.args = append(qb.args, args...)
    return qb
}

func (qb *QueryBuilder) OrWhere(condition string, args ...interface{}) *QueryBuilder {
    qb.where = append(qb.where, fmt.Sprintf("OR %s", condition))
    qb.args = append(qb.args, args...)
    return qb
}

func (qb *QueryBuilder) WhereIn(column string, values []interface{}) *QueryBuilder {
    if len(values) == 0 {
        qb.where = append(qb.where, "1=0")
        return qb
    }

    placeholders := make([]string, len(values))
    for i := range values {
        placeholders[i] = fmt.Sprintf("$%d", len(qb.args)+i+1)
    }
    qb.args = append(qb.args, values...)
    qb.where = append(qb.where, fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ", ")))
    return qb
}

func (qb *QueryBuilder) WhereBetween(column string, start, end interface{}) *QueryBuilder {
    qb.where = append(qb.where, fmt.Sprintf("%s BETWEEN $%d AND $%d", column, len(qb.args)+1, len(qb.args)+2))
    qb.args = append(qb.args, start, end)
    return qb
}

func (qb *QueryBuilder) WhereLike(column, pattern string) *QueryBuilder {
    qb.where = append(qb.where, fmt.Sprintf("%s ILIKE $%d", column, len(qb.args)+1))
    qb.args = append(qb.args, pattern)
    return qb
}

func (qb *QueryBuilder) OrderBy(column, direction string) *QueryBuilder {
    if direction == "" {
        direction = "ASC"
    }
    qb.order = fmt.Sprintf("ORDER BY %s %s", column, strings.ToUpper(direction))
    return qb
}

func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
    qb.limit = limit
    return qb
}

func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
    qb.offset = offset
    return qb
}

func (qb *QueryBuilder) Join(table, condition string) *QueryBuilder {
    qb.joins = append(qb.joins, fmt.Sprintf("JOIN %s ON %s", table, condition))
    return qb
}

func (qb *QueryBuilder) LeftJoin(table, condition string) *QueryBuilder {
    qb.joins = append(qb.joins, fmt.Sprintf("LEFT JOIN %s ON %s", table, condition))
    return qb
}

func (qb *QueryBuilder) RightJoin(table, condition string) *QueryBuilder {
    qb.joins = append(qb.joins, fmt.Sprintf("RIGHT JOIN %s ON %s", table, condition))
    return qb
}

func (qb *QueryBuilder) GroupBy(columns ...string) *QueryBuilder {
    qb.groupBy = append(qb.groupBy, columns...)
    return qb
}

func (qb *QueryBuilder) Having(condition string, args ...interface{}) *QueryBuilder {
    qb.having = append(qb.having, condition)
    qb.args = append(qb.args, args...)
    return qb
}

func (qb *QueryBuilder) BuildSelect() (string, []interface{}) {
    var columns string
    if len(qb.columns) > 0 {
        columns = strings.Join(qb.columns, ", ")
    } else {
        columns = "*"
    }

    query := fmt.Sprintf("SELECT %s FROM %s", columns, qb.table)

    if len(qb.joins) > 0 {
        query += " " + strings.Join(qb.joins, " ")
    }

    if len(qb.where) > 0 {
        query += " WHERE " + strings.Join(qb.where, " ")
    }

    if len(qb.groupBy) > 0 {
        query += " GROUP BY " + strings.Join(qb.groupBy, ", ")
    }

    if len(qb.having) > 0 {
        query += " HAVING " + strings.Join(qb.having, " ")
    }

    if qb.order != "" {
        query += " " + qb.order
    }

    if qb.limit > 0 {
        query += fmt.Sprintf(" LIMIT %d", qb.limit)
    }

    if qb.offset > 0 {
        query += fmt.Sprintf(" OFFSET %d", qb.offset)
    }

    return query, qb.args
}

func (qb *QueryBuilder) BuildCount() (string, []interface{}) {
    query := fmt.Sprintf("SELECT COUNT(*) FROM %s", qb.table)

    if len(qb.joins) > 0 {
        query += " " + strings.Join(qb.joins, " ")
    }

    if len(qb.where) > 0 {
        query += " WHERE " + strings.Join(qb.where, " ")
    }

    return query, qb.args
}
