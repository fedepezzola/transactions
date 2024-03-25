package database

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

// UniqueViolation error code
// lib/pq errorCodeNames
// https://github.com/lib/pq/blob/master/error.go#L178
const UniqueViolation = "23505"

// Set of error variables for CRUD operations.
var (
	ErrDBNotFound        = errors.New("not found")
	ErrDBDuplicatedEntry = errors.New("duplicated entry")
)

// Config is the required properties to use the database.
type Config struct {
	User         string
	Password     string
	Host         string
	Name         string
	MaxIdleConns int
	MaxOpenConns int
	DisableTLS   bool
}

// Open knows how to open a database connection based on the configuration.
func Open(cfg Config) (*sqlx.DB, error) {
	u := GetDataSourceName(cfg)

	db, err := sqlx.Open("postgres", u.String())
	if err != nil {
		return nil, fmt.Errorf("error in Open: %w", err)
	}
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetMaxOpenConns(cfg.MaxOpenConns)

	return db, nil
}

// GetDataSourceName gets a data source name from configuration.
func GetDataSourceName(cfg Config) url.URL {
	sslMode := "require"
	if cfg.DisableTLS {
		sslMode = "disable"
	}

	q := make(url.Values)
	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     cfg.Host,
		Path:     cfg.Name,
		RawQuery: q.Encode(),
	}
	return u
}

// StatusCheck returns nil if it can successfully talk to the database. It
// returns a non-nil error otherwise.
func StatusCheck(ctx context.Context, db *sqlx.DB) error {
	// First check we can ping the database.
	var pingError error
	for attempts := 1; ; attempts++ {
		pingError = db.Ping()
		if pingError == nil {
			break
		}
		time.Sleep(time.Duration(attempts) * 100 * time.Millisecond)
		if ctx.Err() != nil {
			return fmt.Errorf("error in StatusCheck database ping: %w", ctx.Err())
		}
	}

	// Make sure we didn't timeout or be cancelled.
	if ctx.Err() != nil {
		return fmt.Errorf("error in StatusCheck database timeout: %w", ctx.Err())
	}

	// Run a simple query to determine connectivity. Running this query forces a
	// round trip through the database.
	const q = `SELECT true`
	var tmp bool
	err := db.QueryRowContext(ctx, q).Scan(&tmp)
	if err != nil {
		return fmt.Errorf("error in StatusCheck database query: %w", err)
	}
	return nil
}

// Transactor interface needed to begin transaction.
type Transactor interface {
	Beginx() (*sqlx.Tx, error)
}

// WithinTran runs passed function and do commit/rollback at the end.
func WithinTran(ctx context.Context, log *zap.SugaredLogger, db Transactor, fn func(sqlx.ExtContext) error) error {

	// Begin the transaction.
	log.Infow("begin tran")
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("begin tran: %w", err)
	}

	// Mark to the defer function a rollback is required.
	mustRollback := true

	// Set up a defer function for rolling back the transaction. If
	// mustRollback is true it means the call to fn failed, and we
	// need to roll back the transaction.
	defer func() {
		if mustRollback {
			log.Infow("rollback tran")
			if err := tx.Rollback(); err != nil {
				log.Errorw("unable to rollback tran", "ERROR", err)
			}
		}
	}()
	// Execute the code inside the transaction. If the function
	// fails, return the error and the defer function will roll back.
	if err := fn(tx); err != nil {
		// Checks if the error is of code 23505 (unique_violation).
		var pqError *pq.Error
		if ok := errors.As(err, &pqError); ok && pqError.Code == UniqueViolation {
			return ErrDBDuplicatedEntry
		}
		return fmt.Errorf("exec tran: %w", err)
	}

	// Disarm the deferred rollback.
	mustRollback = false

	// Commit the transaction.
	log.Infow("commit tran")
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tran: %w", err)
	}

	return nil
}

// NamedExecContext is a helper function to execute a CUD operation with
// logging and tracing.
func NamedExecContext(ctx context.Context, log *zap.SugaredLogger, db sqlx.ExtContext, query string, data any) error {
	q := queryString(query, data)
	log.Infow("database.NamedExecContext", "query", q)

	if _, err := sqlx.NamedExecContext(ctx, db, query, data); err != nil {
		// Checks if the error is of code 23505 (unique_violation).
		var pqError *pq.Error
		if ok := errors.As(err, &pqError); ok && pqError.Code == UniqueViolation {
			return ErrDBDuplicatedEntry
		}
		return fmt.Errorf("database error on exec: %w", err)
	}

	return nil
}

// NamedQuerySlice is a helper function for executing queries that return a
// collection of data to be unmarshalled into a slice.
func NamedQuerySlice[T any](ctx context.Context, log *zap.SugaredLogger, db sqlx.ExtContext, query string, data any, dest *[]T) error {
	q := queryString(query, data)
	log.Infow("database.NamedQuerySlice", "query", q)

	rows, err := sqlx.NamedQueryContext(ctx, db, query, data)
	if err != nil {
		return fmt.Errorf("database error on slice query: %w", err)
	}
	defer rows.Close()

	var slice []T
	for rows.Next() {
		v := new(T)
		if err := rows.StructScan(v); err != nil {
			return fmt.Errorf("database error walking slice query rows: %w", err)
		}
		slice = append(slice, *v)
	}
	*dest = slice

	return nil
}

// NamedQueryStruct is a helper function for executing queries that return a
// single value to be unmarshalled into a struct type.
func NamedQueryStruct(ctx context.Context, log *zap.SugaredLogger, db sqlx.ExtContext, query string, data any, dest any) error {
	q := queryString(query, data)
	log.Infow("database.NamedQueryStruct", "query", q)

	rows, err := sqlx.NamedQueryContext(ctx, db, query, data)
	if err != nil {
		return fmt.Errorf("database error on struct query: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return ErrDBNotFound
	}

	if err := rows.StructScan(dest); err != nil {
		return fmt.Errorf("database error walking struct query rows: %w", err)
	}

	return nil
}

// queryString provides a pretty print version of the query and parameters.
func queryString(query string, args ...any) string {
	query, params, err := sqlx.Named(query, args)
	if err != nil {
		return err.Error()
	}

	for _, param := range params {
		var value string
		switch v := param.(type) {
		case string:
			value = fmt.Sprintf("%q", v)
		case []byte:
			value = fmt.Sprintf("%q", string(v))
		default:
			value = fmt.Sprintf("%v", v)
		}
		query = strings.Replace(query, "?", value, 1)
	}

	query = strings.ReplaceAll(query, "\t", "")
	query = strings.ReplaceAll(query, "\n", " ")

	return strings.Trim(query, " ")
}
