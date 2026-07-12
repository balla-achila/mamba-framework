package database

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/balla-achila/mamba-framework/framework/config"
    "github.com/balla-achila/mamba-framework/framework/logger"
)

type Database struct {
    pool    *pgxpool.Pool
    config  *config.DatabaseConfig
    logger  logger.Logger
    timeout time.Duration
}

type DB interface {
    Query(ctx context.Context, query string, args ...interface{}) (*Rows, error)
    QueryRow(ctx context.Context, query string, args ...interface{}) Row
    Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
    Insert(ctx context.Context, table string, data map[string]interface{}) (int64, error)
    Update(ctx context.Context, table string, data map[string]interface{}, where string, args ...interface{}) (int64, error)
    Delete(ctx context.Context, table string, where string, args ...interface{}) (int64, error)
    Begin(ctx context.Context) (*Tx, error)
    Close() error
}

type NoOpDB struct{}

func NewNoOp() DB {
    return &NoOpDB{}
}

func (n *NoOpDB) Query(ctx context.Context, query string, args ...interface{}) (*Rows, error) {
    return nil, nil
}
func (n *NoOpDB) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
    return nil
}
func (n *NoOpDB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
    return nil, nil
}
func (n *NoOpDB) Insert(ctx context.Context, table string, data map[string]interface{}) (int64, error) {
    return 1, nil
}
func (n *NoOpDB) Update(ctx context.Context, table string, data map[string]interface{}, where string, args ...interface{}) (int64, error) {
    return 1, nil
}
func (n *NoOpDB) Delete(ctx context.Context, table string, where string, args ...interface{}) (int64, error) {
    return 1, nil
}
func (n *NoOpDB) Begin(ctx context.Context) (*Tx, error) {
    return nil, nil
}
func (n *NoOpDB) Close() error {
    return nil
}

func New(cfg *config.DatabaseConfig, log logger.Logger) (DB, error) {
    dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)

    poolConfig, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to parse database config: %w", err)
    }

    poolConfig.MaxConns = int32(cfg.MaxConnections)
    poolConfig.MinConns = int32(cfg.MinConnections)
    poolConfig.MaxConnIdleTime = time.Duration(cfg.MaxIdleTime) * time.Second
    poolConfig.MaxConnLifetime = time.Duration(cfg.MaxLifeTime) * time.Second

    pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create connection pool: %w", err)
    }

    if err := pool.Ping(context.Background()); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    return &Database{
        pool:    pool,
        config:  cfg,
        logger:  log,
        timeout: time.Duration(cfg.QueryTimeout) * time.Second,
    }, nil
}

func (db *Database) Query(ctx context.Context, query string, args ...interface{}) (*Rows, error) {
    ctx, cancel := context.WithTimeout(ctx, db.timeout)
    defer cancel()

    rows, err := db.pool.Query(ctx, query, args...)
    if err != nil {
        db.logger.Error("Query failed", "error", err, "query", query)
        return nil, err
    }

    return &Rows{rows: rows}, nil
}

func (db *Database) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
    ctx, cancel := context.WithTimeout(ctx, db.timeout)
    defer cancel()

    return db.pool.QueryRow(ctx, query, args...)
}

func (db *Database) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
    ctx, cancel := context.WithTimeout(ctx, db.timeout)
    defer cancel()

    result, err := db.pool.Exec(ctx, query, args...)
    if err != nil {
        db.logger.Error("Exec failed", "error", err, "query", query)
        return nil, err
    }

    return &pgxResult{result}, nil
}

func (db *Database) Insert(ctx context.Context, table string, data map[string]interface{}) (int64, error) {
    if len(data) == 0 {
        return 0, fmt.Errorf("no data to insert")
    }

    columns := make([]string, 0, len(data))
    values := make([]interface{}, 0, len(data))
    placeholders := make([]string, 0, len(data))

    i := 1
    for key, value := range data {
        columns = append(columns, key)
        values = append(values, value)
        placeholders = append(placeholders, fmt.Sprintf("$%d", i))
        i++
    }

    query := fmt.Sprintf(
        "INSERT INTO %s (%s) VALUES (%s) RETURNING id",
        table,
        join(columns, ", "),
        join(placeholders, ", "),
    )

    var id int64
    err := db.QueryRow(ctx, query, values...).Scan(&id)
    if err != nil {
        db.logger.Error("Insert failed", "error", err, "table", table)
        return 0, err
    }

    return id, nil
}

func (db *Database) Update(ctx context.Context, table string, data map[string]interface{}, where string, args ...interface{}) (int64, error) {
    if len(data) == 0 {
        return 0, fmt.Errorf("no data to update")
    }

    setClauses := make([]string, 0, len(data))
    values := make([]interface{}, 0, len(data))
    i := len(args) + 1

    for key, value := range data {
        setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, i))
        values = append(values, value)
        i++
    }

    values = append(args, values...)
    query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", table, join(setClauses, ", "), where)

    result, err := db.Exec(ctx, query, values...)
    if err != nil {
        db.logger.Error("Update failed", "error", err, "table", table)
        return 0, err
    }

    return result.RowsAffected()
}

func (db *Database) Delete(ctx context.Context, table string, where string, args ...interface{}) (int64, error) {
    query := fmt.Sprintf("DELETE FROM %s WHERE %s", table, where)

    result, err := db.Exec(ctx, query, args...)
    if err != nil {
        db.logger.Error("Delete failed", "error", err, "table", table)
        return 0, err
    }

    return result.RowsAffected()
}

func (db *Database) Begin(ctx context.Context) (*Tx, error) {
    tx, err := db.pool.Begin(ctx)
    if err != nil {
        db.logger.Error("Begin transaction failed", "error", err)
        return nil, err
    }

    return &Tx{tx: tx, logger: db.logger}, nil
}

func (db *Database) Close() error {
    db.pool.Close()
    return nil
}

func join(strs []string, sep string) string {
    result := ""
    for i, s := range strs {
        if i > 0 {
            result += sep
        }
        result += s
    }
    return result
}

type Rows struct {
    rows pgx.Rows
}

func (r *Rows) Next() bool {
    return r.rows.Next()
}

func (r *Rows) Scan(dest ...interface{}) error {
    return r.rows.Scan(dest...)
}

func (r *Rows) Close() error {
    r.rows.Close()
    return nil
}

type Row interface {
    Scan(dest ...interface{}) error
}

type Tx struct {
    tx     pgx.Tx
    logger logger.Logger
}

func (t *Tx) Query(ctx context.Context, query string, args ...interface{}) (*Rows, error) {
    rows, err := t.tx.Query(ctx, query, args...)
    if err != nil {
        t.logger.Error("Transaction query failed", "error", err, "query", query)
        return nil, err
    }
    return &Rows{rows: rows}, nil
}

func (t *Tx) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
    return t.tx.QueryRow(ctx, query, args...)
}

func (t *Tx) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
    result, err := t.tx.Exec(ctx, query, args...)
    if err != nil {
        t.logger.Error("Transaction exec failed", "error", err, "query", query)
        return nil, err
    }
    return &pgxResult{result}, nil
}

func (t *Tx) Commit(ctx context.Context) error {
    err := t.tx.Commit(ctx)
    if err != nil {
        t.logger.Error("Transaction commit failed", "error", err)
    }
    return err
}

func (t *Tx) Rollback(ctx context.Context) error {
    err := t.tx.Rollback(ctx)
    if err != nil {
        t.logger.Error("Transaction rollback failed", "error", err)
    }
    return err
}

type pgxResult struct {
    commandTag pgx.CommandTag
}

func (r *pgxResult) LastInsertId() (int64, error) {
    return 0, fmt.Errorf("LastInsertId not supported")
}

func (r *pgxResult) RowsAffected() (int64, error) {
    return r.commandTag.RowsAffected(), nil
}