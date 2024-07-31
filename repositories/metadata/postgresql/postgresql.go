package postgresql

import (
	"context"
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	log "github.com/sirupsen/logrus"

	"github.com/teran/archived/repositories/metadata"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type repository struct {
	db *sql.DB
	tp func() time.Time
}

func New(db *sql.DB) metadata.Repository {
	return newWithTimeProvider(db, time.Now)
}

func newWithTimeProvider(db *sql.DB, tp func() time.Time) metadata.Repository {
	return &repository{
		db: db,
		tp: tp,
	}
}

func mapSQLErrors(err error) error {
	switch err {
	case sql.ErrNoRows:
		return metadata.ErrNotFound
	default:
		return err
	}
}

type queryRunner interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type execRunner interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type query interface {
	ToSql() (string, []interface{}, error)
}

func mkQuery(q query) (string, []any, error) {
	sql, args, err := q.ToSql()
	if err != nil {
		return "", nil, err
	}

	log.WithFields(log.Fields{
		"query": sql,
		"args":  args,
	}).Tracef("SQL query generated")

	return sql, args, nil
}

func selectQueryRow(ctx context.Context, db queryRunner, q sq.SelectBuilder) (sq.RowScanner, error) {
	sql, args, err := mkQuery(q)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	defer func() {
		log.WithFields(log.Fields{
			"query":    sql,
			"args":     args,
			"duration": time.Now().Sub(start),
		}).Tracef("SQL query executed")
	}()

	return db.QueryRowContext(ctx, sql, args...), nil
}

func selectQuery(ctx context.Context, db queryRunner, q sq.SelectBuilder) (*sql.Rows, error) {
	sql, args, err := mkQuery(q)
	if err != nil {
		return nil, err
	}

	return db.QueryContext(ctx, sql, args...)
}

func insertQuery(ctx context.Context, db execRunner, q sq.InsertBuilder) (sql.Result, error) {
	sql, args, err := mkQuery(q)
	if err != nil {
		return nil, err
	}

	return db.ExecContext(ctx, sql, args...)
}

func updateQuery(ctx context.Context, db execRunner, q sq.UpdateBuilder) (sql.Result, error) {
	sql, args, err := mkQuery(q)
	if err != nil {
		return nil, err
	}

	return db.ExecContext(ctx, sql, args...)
}

func deleteQuery(ctx context.Context, db execRunner, q sq.DeleteBuilder) (sql.Result, error) {
	sql, args, err := mkQuery(q)
	if err != nil {
		return nil, err
	}

	return db.ExecContext(ctx, sql, args...)
}
