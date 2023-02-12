package leaderelection

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

// PostgresqlBackend is the implementation of the backend interface
// using the posgresql database.
type PostgresqlBackend struct {
	key   string
	query string
	db    backendConn
}

// NewPostgresqlBackend create a new backend using a postgresql
// database. To configure the connection use one of  WithConnString, WithConn or WithConnPool
func NewPostgresqlBackend(ctx context.Context, key string, opts ...PostgresqlOption) (Backend, error) {

	var settings PostgresqlSettings = getDefaultSettings()
	for _, opt := range opts {
		opt.Apply(&settings)
	}

	var conn backendConn
	var err error
	switch {
	case settings.ConnString != "":
		conn, err = pgxpool.New(ctx, settings.ConnString)
		if err != nil {
			return nil, errors.Wrap(err, "unable to connect to database")
		}
	case settings.Conn != nil:
		conn = settings.Conn
	case settings.Connpool != nil:
		conn = settings.Connpool
	default:
		return nil, errors.New("not connection settings found, you must set one of ConnString or Conn or Connpool")
	}

	conn.Exec(ctx, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		key             VARCHAR(24) PRIMARY KEY,
		leader_id       VARCHAR(36),
		valid_until     TIMESTAMPTZ
	)`, settings.TableName))

	query := fmt.Sprintf(`INSERT INTO %s as t (key, leader_id, valid_until) VALUES ($1, $2, NOW() at time zone 'utc' + $3 * INTERVAL '1 seconds'  ) 
	 ON CONFLICT (key) DO
	UPDATE SET leader_id = $2, valid_until =  NOW() at time zone 'utc' + $3 * INTERVAL '1 seconds'
	WHERE t.valid_until < NOW() at time zone 'utc' OR t.leader_id = $2`, settings.TableName)

	return &PostgresqlBackend{
		query: query,
		key:   key,
		db:    conn,
	}, nil
}

// WriteEntry attempts to insert or update a record into the database
// using condition to evaluate the TTL.
// Returns true if the a row was updated, false if not.
// If error is not nil mean something unexpected happend.
func (b *PostgresqlBackend) WriteEntry(ctx context.Context, leaderID string, ttl time.Duration) (bool, error) {

	result, err := b.db.Exec(ctx, b.query, b.key, leaderID, int64(ttl.Seconds()))
	if err != nil {
		return false, err
	}

	return result.RowsAffected() == 1, nil
}

// PostgresqlBackend defines the common values for configuration
// of the postgresql backend.
type PostgresqlSettings struct {
	Conn         *pgx.Conn
	Connpool     *pgxpool.Pool
	ConnString   string
	TableName    string
	DatabaseName string
}

// backendConn is the common interface used to support
// both the pgx.Conn and pgxpool.Pool as input settings.
type backendConn interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
}

// PostgresqlOption is an option for configure the
// postgresql backend.
type PostgresqlOption interface {
	Apply(*PostgresqlSettings)
}

func getDefaultSettings() PostgresqlSettings {
	return PostgresqlSettings{
		TableName: "leaderlock",
	}
}

type withConnString struct {
	conn string
}

func (o *withConnString) Apply(s *PostgresqlSettings) {
	s.ConnString = o.conn
}

// WithConnString returns a PostgresqlOption for configuring
// the backend to use a connections sttring. A connection
// pool will be created using the setings
func WithConnString(conn string) PostgresqlOption {
	return &withConnString{conn: conn}
}

type withConn struct {
	conn *pgx.Conn
}

func (o *withConn) Apply(s *PostgresqlSettings) {
	s.Conn = o.conn
}

// WithConn returns a PostgresqlOption for configuring
// the backend to use a connection
func WithConn(conn *pgx.Conn) PostgresqlOption {
	return &withConn{conn: conn}
}

type withConnPool struct {
	conn *pgxpool.Pool
}

func (o *withConnPool) Apply(s *PostgresqlSettings) {
	s.Connpool = o.conn
}

// WithConnPool returns a PostgresqlOption for configuring
// the backend to use a connection pool
func WithConnPool(conn *pgxpool.Pool) PostgresqlOption {
	return &withConnPool{conn: conn}
}

type withTableName struct {
	tableName string
}

func (o *withTableName) Apply(s *PostgresqlSettings) {
	s.TableName = o.tableName
}

// WithTableName returns a PostgresqlOption for configuring
// the backed table name
func WithTableName(tableName string) PostgresqlOption {
	return &withTableName{tableName: tableName}
}
