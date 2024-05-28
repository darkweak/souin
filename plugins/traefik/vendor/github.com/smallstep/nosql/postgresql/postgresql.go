//go:build !nopgx
// +build !nopgx

package postgresql

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"
	pgxstdlib "github.com/jackc/pgx/v4/stdlib"
	"github.com/pkg/errors"
	"github.com/smallstep/nosql/database"
)

// DB is a wrapper over *sql.DB,
type DB struct {
	db *sql.DB
}

func quoteIdentifier(identifier string) string {
	parts := strings.Split(identifier, ".")
	return pgx.Identifier(parts).Sanitize()
}

func createDatabase(config *pgx.ConnConfig) error {
	db := config.Database
	if db == "" {
		// If no explicit database name is given, PostgreSQL defaults to the
		// database with the same name as the user.
		db = config.User
		if db == "" {
			return errors.New("error creating database: database name is missing")
		}
	}

	// The database "template1" is the default template for all new databases,
	// so it should always exist.
	tempConfig := config.Copy()
	tempConfig.Database = "template1"

	conn, err := pgx.ConnectConfig(context.Background(), tempConfig)
	if err != nil {
		return errors.Wrap(err, "error connecting to PostgreSQL")
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s", quoteIdentifier(db)))
	if err != nil {
		if !strings.Contains(err.Error(), "(SQLSTATE 42P04)") {
			return errors.Wrapf(err, "error creating database %s (if not exists)", db)
		}
	}

	return nil
}

// Open creates a Driver and connects to the database with the given address
// and access details.
func (db *DB) Open(dataSourceName string, opt ...database.Option) error {
	opts := &database.Options{}
	for _, o := range opt {
		if err := o(opts); err != nil {
			return err
		}
	}

	config, err := pgx.ParseConfig(dataSourceName)
	if err != nil {
		return errors.Wrap(err, "error parsing PostgreSQL DSN")
	}
	// An explicit database name overrides one parsed from the DSN.
	if opts.Database != "" {
		config.Database = opts.Database
	}

	// Attempt to open the database.
	db.db = pgxstdlib.OpenDB(*config)
	err = db.db.Ping()
	if err != nil && strings.Contains(err.Error(), "(SQLSTATE 3D000)") {
		// The database does not exist. Create it.
		err = createDatabase(config)
		if err != nil {
			return err
		}

		// Attempt to open the database again.
		db.db = pgxstdlib.OpenDB(*config)
		err = db.db.Ping()
	}
	if err != nil {
		return errors.Wrapf(err, "error connecting to PostgreSQL database")
	}

	return nil
}

// Close shutsdown the database driver.
func (db *DB) Close() error {
	return errors.WithStack(db.db.Close())
}

func getAllQry(bucket []byte) string {
	return fmt.Sprintf("SELECT * FROM %s", quoteIdentifier(string(bucket)))
}

func getQry(bucket []byte) string {
	return fmt.Sprintf("SELECT nvalue FROM %s WHERE nkey = $1;", quoteIdentifier(string(bucket)))
}

func getQryForUpdate(bucket []byte) string {
	return fmt.Sprintf("SELECT nvalue FROM %s WHERE nkey = $1 FOR UPDATE;", quoteIdentifier(string(bucket)))
}

func insertUpdateQry(bucket []byte) string {
	return fmt.Sprintf("INSERT INTO %s (nkey, nvalue) VALUES ($1, $2) ON CONFLICT (nkey) DO UPDATE SET nvalue = excluded.nvalue;", quoteIdentifier(string(bucket)))
}

func delQry(bucket []byte) string {
	return fmt.Sprintf("DELETE FROM %s WHERE nkey = $1;", quoteIdentifier(string(bucket)))
}

func createTableQry(bucket []byte) string {
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (nkey BYTEA CHECK (octet_length(nkey) <= 255), nvalue BYTEA, PRIMARY KEY (nkey));", quoteIdentifier(string(bucket)))
}

func deleteTableQry(bucket []byte) string {
	return fmt.Sprintf("DROP TABLE %s;", quoteIdentifier(string(bucket)))
}

// Get retrieves the column/row with given key.
func (db *DB) Get(bucket, key []byte) ([]byte, error) {
	var val string
	err := db.db.QueryRow(getQry(bucket), key).Scan(&val)
	switch {
	case err == sql.ErrNoRows:
		return nil, errors.Wrapf(database.ErrNotFound, "%s/%s not found", bucket, key)
	case err != nil:
		return nil, errors.Wrapf(err, "failed to get %s/%s", bucket, key)
	default:
		return []byte(val), nil
	}
}

// Set inserts the key and value into the given bucket(column).
func (db *DB) Set(bucket, key, value []byte) error {
	_, err := db.db.Exec(insertUpdateQry(bucket), key, value)
	if err != nil {
		return errors.Wrapf(err, "failed to set %s/%s", bucket, key)
	}
	return nil
}

// Del deletes a row from the database.
func (db *DB) Del(bucket, key []byte) error {
	_, err := db.db.Exec(delQry(bucket), key)
	return errors.Wrapf(err, "failed to delete %s/%s", bucket, key)
}

// List returns the full list of entries in a column.
func (db *DB) List(bucket []byte) ([]*database.Entry, error) {
	rows, err := db.db.Query(getAllQry(bucket))
	if err != nil {
		estr := err.Error()
		if strings.Contains(estr, "(SQLSTATE 42P01)") {
			return nil, errors.Wrapf(database.ErrNotFound, estr)
		}
		return nil, errors.Wrapf(err, "error querying table %s", bucket)
	}
	defer rows.Close()
	var (
		key, value string
		entries    []*database.Entry
	)
	for rows.Next() {
		err := rows.Scan(&key, &value)
		if err != nil {
			return nil, errors.Wrap(err, "error getting key and value from row")
		}
		entries = append(entries, &database.Entry{
			Bucket: bucket,
			Key:    []byte(key),
			Value:  []byte(value),
		})
	}
	err = rows.Err()
	if err != nil {
		return nil, errors.Wrap(err, "error accessing row")
	}
	return entries, nil
}

// CmpAndSwap modifies the value at the given bucket and key (to newValue)
// only if the existing (current) value matches oldValue.
func (db *DB) CmpAndSwap(bucket, key, oldValue, newValue []byte) ([]byte, bool, error) {
	sqlTx, err := db.db.Begin()
	if err != nil {
		return nil, false, errors.WithStack(err)
	}

	val, swapped, err := cmpAndSwap(sqlTx, bucket, key, oldValue, newValue)
	switch {
	case err != nil:
		if err := sqlTx.Rollback(); err != nil {
			return nil, false, errors.Wrapf(err, "failed to execute CmpAndSwap transaction on %s/%s and failed to rollback transaction", bucket, key)
		}
		return nil, false, err
	case swapped:
		if err := sqlTx.Commit(); err != nil {
			return nil, false, errors.Wrapf(err, "failed to commit PostgreSQL transaction")
		}
		return val, swapped, nil
	default:
		if err := sqlTx.Rollback(); err != nil {
			return nil, false, errors.Wrapf(err, "failed to rollback read-only CmpAndSwap transaction on %s/%s", bucket, key)
		}
		return val, swapped, err
	}
}

func cmpAndSwap(sqlTx *sql.Tx, bucket, key, oldValue, newValue []byte) ([]byte, bool, error) {
	var current []byte
	err := sqlTx.QueryRow(getQryForUpdate(bucket), key).Scan(&current)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}
	if !bytes.Equal(current, oldValue) {
		return current, false, nil
	}

	if _, err = sqlTx.Exec(insertUpdateQry(bucket), key, newValue); err != nil {
		return nil, false, errors.Wrapf(err, "failed to set %s/%s", bucket, key)
	}
	return newValue, true, nil
}

// Update performs multiple commands on one read-write transaction.
func (db *DB) Update(tx *database.Tx) error {
	sqlTx, err := db.db.Begin()
	if err != nil {
		return errors.WithStack(err)
	}
	rollback := func(err error) error {
		if rollbackErr := sqlTx.Rollback(); rollbackErr != nil {
			return errors.Wrap(err, "UPDATE failed, unable to rollback transaction")
		}
		return errors.Wrap(err, "UPDATE failed")
	}
	for _, q := range tx.Operations {
		// create or delete buckets
		switch q.Cmd {
		case database.CreateTable:
			_, err := sqlTx.Exec(createTableQry(q.Bucket))
			if err != nil {
				return rollback(errors.Wrapf(err, "failed to create table %s", q.Bucket))
			}
		case database.DeleteTable:
			_, err := sqlTx.Exec(deleteTableQry(q.Bucket))
			if err != nil {
				estr := err.Error()
				if strings.Contains(estr, "(SQLSTATE 42P01)") {
					return errors.Wrapf(database.ErrNotFound, estr)
				}
				return errors.Wrapf(err, "failed to delete table %s", q.Bucket)
			}
		case database.Get:
			var val string
			err := sqlTx.QueryRow(getQry(q.Bucket), q.Key).Scan(&val)
			switch {
			case err == sql.ErrNoRows:
				return rollback(errors.Wrapf(database.ErrNotFound, "%s/%s not found", q.Bucket, q.Key))
			case err != nil:
				return rollback(errors.Wrapf(err, "failed to get %s/%s", q.Bucket, q.Key))
			default:
				q.Result = []byte(val)
			}
		case database.Set:
			if _, err = sqlTx.Exec(insertUpdateQry(q.Bucket), q.Key, q.Value); err != nil {
				return rollback(errors.Wrapf(err, "failed to set %s/%s", q.Bucket, q.Key))
			}
		case database.Delete:
			if _, err = sqlTx.Exec(delQry(q.Bucket), q.Key); err != nil {
				return rollback(errors.Wrapf(err, "failed to delete %s/%s", q.Bucket, q.Key))
			}
		case database.CmpAndSwap:
			q.Result, q.Swapped, err = cmpAndSwap(sqlTx, q.Bucket, q.Key, q.CmpValue, q.Value)
			if err != nil {
				return rollback(errors.Wrapf(err, "failed to load-or-store %s/%s", q.Bucket, q.Key))
			}
		case database.CmpOrRollback:
			return database.ErrOpNotSupported
		default:
			return database.ErrOpNotSupported
		}
	}

	if err = errors.WithStack(sqlTx.Commit()); err != nil {
		return rollback(err)
	}
	return nil
}

// CreateTable creates a table in the database.
func (db *DB) CreateTable(bucket []byte) error {
	_, err := db.db.Exec(createTableQry(bucket))
	if err != nil {
		return errors.Wrapf(err, "failed to create table %s", bucket)
	}
	return nil
}

// DeleteTable deletes a table in the database.
func (db *DB) DeleteTable(bucket []byte) error {
	_, err := db.db.Exec(deleteTableQry(bucket))
	if err != nil {
		estr := err.Error()
		if strings.Contains(estr, "(SQLSTATE 42P01)") {
			return errors.Wrapf(database.ErrNotFound, estr)
		}
		return errors.Wrapf(err, "failed to delete table %s", bucket)
	}
	return nil
}
