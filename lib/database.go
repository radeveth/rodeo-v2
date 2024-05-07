package lib

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// ErrDatabaseNotFound is returned by database methods returning
// a single entity when no entity matched the given parameters
var ErrDatabaseNotFound = errors.New("Database: Can't find entity for given parameters")

// Database represents a connection to a PostgreSQL database
type Database struct {
	ctx  *Ctx
	url  string
	conn *sqlx.DB
}

// NewDatabase setsup a connection to a PostgreSQL database
func NewDatabase(url string) *Database {
	db := &Database{url: url}
	if err := db.Connect(); err != nil {
		panic(fmt.Sprintf("database postgres: %v", err))
	}
	return db
}

func NewDatabaseNoConnect(url string) *Database {
	return &Database{url: url}
}

func (db *Database) Connect() error {
	if db.conn != nil {
		db.Close()
	}
	sourceName, err := pq.ParseURL(db.url)
	if err != nil {
		return err
	}
	conn, err := sqlx.Open("postgres", sourceName)
	if err != nil {
		return err
	}
	db.conn = conn
	return nil
}

func (db *Database) Close() {
	db.conn.Close()
	db.conn = nil
}

func (db *Database) WithCtx(ctx *Ctx) *Database {
	return &Database{ctx: ctx, url: db.url, conn: db.conn}
}

// Connection returns the underlying sqlx connection
func (db *Database) Connection() *sqlx.DB {
	return db.conn
}

// Execute simply runs a SQL statement without caring about the results
func (db *Database) Execute(query string, values ...interface{}) {
	Check(db.ExecuteErr(query, values...))
}

// ExecuteErr simply runs a SQL statement without caring about the results
func (db *Database) ExecuteErr(query string, values ...interface{}) error {
	//Log("debug", "executing sql", J{"sql": query})
	_, err := db.conn.Exec(query, values...)
	return err
}

// First returns the first entity for the given SQL query. It must be passed a non-nil struct.
func (db *Database) First(result interface{}, query string, values ...interface{}) {
	Check(db.FirstErr(result, query, values...))
}

// FirstErr returns the first entity for the given SQL query. It must be passed a non-nil struct.
func (db *Database) FirstErr(result interface{}, query string, values ...interface{}) error {
	//Log("debug", "executing sql", J{"sql": query})
	return replaceNotFoundError(db.conn.Get(result, query, values...))
}

// All returns all entities for the given SQL query. It must be passed a non-nil pointer to array of struct.
func (db *Database) All(result interface{}, query string, values ...interface{}) {
	Check(db.AllErr(result, query, values...))
}

// AllErr returns all entities for the given SQL query. It must be passed a non-nil pointer to array of struct.
func (db *Database) AllErr(result interface{}, query string, values ...interface{}) error {
	//Log("debug", "executing sql", J{"sql": query})
	return db.conn.Select(result, query, values...)
}

// FirstWhere returns the first entity for the given SQL where condition. It must be passed a non-nil struct.
func (db *Database) FirstWhere(model interface{}, where string, values ...interface{}) {
	Check(db.FirstWhereErr(model, where, values...))
}

// FirstWhereErr returns the first entity for the given SQL where condition. It must be passed a non-nil struct.
func (db *Database) FirstWhereErr(model interface{}, where string, values ...interface{}) error {
	err := db.MustFirstWhereErr(model, where, values...)
	if err == ErrDatabaseNotFound {
		return nil
	}
	return err
}

// MustFirstWhere returns the first entity for the given SQL where condition. It must be passed a non-nil struct.
func (db *Database) MustFirstWhere(model interface{}, where string, values ...interface{}) {
	Check(db.MustFirstWhereErr(model, where, values...))
}

// MustFirstWhereErr returns the first entity for the given SQL where condition. It must be passed a non-nil struct.
func (db *Database) MustFirstWhereErr(model interface{}, where string, values ...interface{}) error {
	table := tableNameFor(model)
	sql := fmt.Sprintf("SELECT * FROM %s WHERE %s", table, where)
	//Log("debug", "executing sql", J{"sql": sql})
	return replaceNotFoundError(db.conn.Get(model, sql, values...))
}

// AllWhere returns all entities for the given SQL where condition. It must be passed a non-nil pointer to array of struct.
func (db *Database) AllWhere(result interface{}, where string, values ...interface{}) {
	Check(db.AllWhereErr(result, where, values...))
}

// AllWhereErr returns all entities for the given SQL where condition. It must be passed a non-nil pointer to array of struct.
func (db *Database) AllWhereErr(result interface{}, where string, values ...interface{}) error {
	table := tableNameFor(result)
	sql := fmt.Sprintf("SELECT * FROM %s WHERE %s", table, where)
	//Log("debug", "executing sql", J{"sql": sql})
	return db.conn.Select(result, sql, values...)
}

// Put upserts the given entity into the database
func (db *Database) Put(model interface{}) {
	Check(db.PutErr(model))
}

// PutErr upserts the given entity into the database
func (db *Database) PutErr(model interface{}) error {
	cols := []string{}
	args := []interface{}{}
	table := tableNameFor(model)

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("DB.Put: Value is not a struct: %#v", model)
	}
	vt := v.Type()
	for i := 0; i < vt.NumField(); i++ {
		cols = append(cols, StringToSnakeCase(vt.Field(i).Name))
		args = append(args, v.Field(i).Interface())
	}

	join := strings.Join
	vars := []string{}
	sets := []string{}
	for i, c := range cols {
		vars = append(vars, "$"+IntToString(int64(i+1)))
		sets = append(sets, c+" = "+vars[i])
	}
	sql := "insert into %s (%s) values (%s) on conflict (id) do update set %s returning *"
	sql = fmt.Sprintf(sql, table, join(cols, ", "), join(vars, ", "), join(sets, ", "))
	return db.FirstErr(model, sql, args...)
}

// Delete deletes the given entity from the database
func (db *Database) Delete(model interface{}) {
	Check(db.DeleteErr(model))
}

// DeleteErr deletes the given entity from the database
func (db *Database) DeleteErr(model interface{}) error {
	table := tableNameFor(model)
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return db.ExecuteErr(fmt.Sprintf("DELETE FROM %s WHERE id = $1", table), v.Field(0).Interface())
}

func tableNameFor(model interface{}) string {
	parts := strings.Split(fmt.Sprintf("%T", model), ".")
	parts = strings.Split(StringToSnakeCase(parts[len(parts)-1]), "_")
	for i, v := range parts {
		if v[len(v)-1] == 'y' {
			parts[i] = v[:len(v)-1] + "ies"
		} else {
			parts[i] = v + "s"
		}
	}
	return strings.Join(parts, "_")
}

func replaceNotFoundError(err error) error {
	if err == sql.ErrNoRows {
		return ErrDatabaseNotFound
	}
	return err
}
