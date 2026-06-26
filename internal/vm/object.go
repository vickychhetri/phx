package vm

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"phx/internal/ast"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type ObjectType string

const (
	INTEGER_OBJ = "INTEGER"
	FLOAT_OBJ   = "FLOAT"
	BOOLEAN_OBJ = "BOOLEAN"
	STRING_OBJ  = "STRING"
	NULL_OBJ    = "NULL"
	ERROR_OBJ   = "ERROR"
	FUNCTION_OBJ = "FUNCTION"
	BUILTIN_OBJ  = "BUILTIN"
	RETURN_VALUE_OBJ = "RETURN_VALUE"
	CLASS_OBJ   = "CLASS"
	INSTANCE_OBJ = "INSTANCE"
	BREAK_OBJ   = "BREAK"
	CONTINUE_OBJ = "CONTINUE"
	THREAD_OBJ  = "THREAD"
	CHANNEL_OBJ = "CHANNEL"
	ARRAY_OBJ   = "ARRAY"
	EXCEPTION_OBJ = "EXCEPTION"
)

type ExceptionObject struct {
	Value Object
}

func (eo *ExceptionObject) Type() ObjectType { return EXCEPTION_OBJ }
func (eo *ExceptionObject) Inspect() string  { return eo.Value.Inspect() }


type Object interface {
	Type() ObjectType
	Inspect() string
}

type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

const (
	intCacheMin = -100
	intCacheMax = 5000000
)

var intCache [intCacheMax - intCacheMin + 1]*Integer

func init() {
	for i := intCacheMin; i <= intCacheMax; i++ {
		intCache[i-intCacheMin] = &Integer{Value: int64(i)}
	}
}

func NewInteger(val int64) *Integer {
	if val >= intCacheMin && val <= intCacheMax {
		return intCache[val-intCacheMin]
	}
	return &Integer{Value: val}
}

type Float struct {
	Value float64
}

func (f *Float) Type() ObjectType { return FLOAT_OBJ }
func (f *Float) Inspect() string  { return fmt.Sprintf("%g", f.Value) }

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

type String struct {
	Value string
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "NULL" }

type Error struct {
	Message string
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return "ERROR: " + e.Message }

type Function struct {
	Parameters []*ast.Parameter
	Body       *ast.BlockStatement
	Env        *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string  { return "function" }

type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

type Class struct {
	Name      string
	Methods   map[string]*Function
	Constants map[string]Object
}

func (c *Class) Type() ObjectType { return CLASS_OBJ }
func (c *Class) Inspect() string  { return "class " + c.Name }

type ObjectInstance struct {
	Class  *Class
	Fields map[string]Object
}

func (oi *ObjectInstance) Type() ObjectType { return INSTANCE_OBJ }
func (oi *ObjectInstance) Inspect() string  { return "object of class " + oi.Class.Name }

func NewObjectInstance(class *Class) *ObjectInstance {
	return &ObjectInstance{
		Class:  class,
		Fields: make(map[string]Object),
	}
}

type Break struct{}
func (b *Break) Type() ObjectType { return BREAK_OBJ }
func (b *Break) Inspect() string  { return "break" }

type Continue struct{}
func (c *Continue) Type() ObjectType { return CONTINUE_OBJ }
func (c *Continue) Inspect() string  { return "continue" }

type Thread struct {
	Done chan struct{}
	Val  Object
	Err  Object
}
func (t *Thread) Type() ObjectType { return THREAD_OBJ }
func (t *Thread) Inspect() string  { return "<Thread>" }

type Channel struct {
	Ch chan Object
}
func (c *Channel) Type() ObjectType { return CHANNEL_OBJ }
func (c *Channel) Inspect() string  { return "<Channel>" }

type BuiltinFunction func(args []Object, env *Environment, out io.Writer) Object

type Builtin struct {
	Fn BuiltinFunction
}
func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "<builtin function>" }

type ArrayPair struct {
	Key   Object
	Value Object
}

type Array struct {
	Pairs []*ArrayPair
}
func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
	var out bytes.Buffer
	out.WriteString("Array ( ")
	for _, pair := range a.Pairs {
		out.WriteString(fmt.Sprintf("[%s] => %s ", pair.Key.Inspect(), pair.Value.Inspect()))
	}
	out.WriteString(")")
	return out.String()
}

type PHXExceptionObject struct {
	Message string
	Code    int64
}

func (e *PHXExceptionObject) Type() ObjectType { return INSTANCE_OBJ }
func (e *PHXExceptionObject) Inspect() string  { return "Exception: " + e.Message }

type FileObject struct {
	file *os.File
}

func (f *FileObject) Type() ObjectType { return INSTANCE_OBJ }
func (f *FileObject) Inspect() string  { return "FileResource" }

type DatabaseObject struct {
	path string
	data map[string][]map[string]Object
}

func (db *DatabaseObject) Type() ObjectType { return INSTANCE_OBJ }
func (db *DatabaseObject) Inspect() string  { return "DatabaseResource" }

func (db *DatabaseObject) saveToJSON() error {
	if db.path == "" {
		return fmt.Errorf("no database path")
	}
	serializable := make(map[string][]map[string]interface{})
	for table, rows := range db.data {
		serializableRows := []map[string]interface{}{}
		for _, row := range rows {
			serializableRow := make(map[string]interface{})
			for col, val := range row {
				switch v := val.(type) {
				case *Integer:
					serializableRow[col] = v.Value
				case *Float:
					serializableRow[col] = v.Value
				case *Boolean:
					serializableRow[col] = v.Value
				case *String:
					serializableRow[col] = v.Value
				default:
					serializableRow[col] = val.Inspect()
				}
			}
			serializableRows = append(serializableRows, serializableRow)
		}
		serializable[table] = serializableRows
	}
	bytes, err := json.MarshalIndent(serializable, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(db.path, bytes, 0666)
}

func (db *DatabaseObject) loadFromJSON(bytes []byte) {
	var raw map[string][]map[string]interface{}
	err := json.Unmarshal(bytes, &raw)
	if err != nil {
		return
	}
	for table, rows := range raw {
		dbRows := []map[string]Object{}
		for _, row := range rows {
			dbRow := make(map[string]Object)
			for col, val := range row {
				switch v := val.(type) {
				case float64:
					if v == float64(int64(v)) {
						dbRow[col] = &Integer{Value: int64(v)}
					} else {
						dbRow[col] = &Float{Value: v}
					}
				case string:
					dbRow[col] = &String{Value: v}
				case bool:
					if v {
						dbRow[col] = &Boolean{Value: true}
					} else {
						dbRow[col] = &Boolean{Value: false}
					}
				default:
					dbRow[col] = &String{Value: fmt.Sprintf("%v", val)}
				}
			}
			dbRows = append(dbRows, dbRow)
		}
		db.data[table] = dbRows
	}
}

func (db *DatabaseObject) executeExec(query string) error {
	q := strings.TrimSpace(query)
	upperQ := strings.ToUpper(q)

	if strings.HasPrefix(upperQ, "CREATE TABLE") {
		re := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(\w+)`)
		matches := re.FindStringSubmatch(q)
		if len(matches) < 2 {
			return fmt.Errorf("invalid CREATE TABLE query")
		}
		tableName := matches[1]
		db.data[tableName] = []map[string]Object{}
		return db.saveToJSON()
	}

	if strings.HasPrefix(upperQ, "INSERT INTO") {
		re := regexp.MustCompile(`(?i)INSERT\s+INTO\s+(\w+)\s*\(([^)]+)\)\s*VALUES\s*\(([^)]+)\)`)
		matches := re.FindStringSubmatch(q)
		if len(matches) < 4 {
			return fmt.Errorf("invalid INSERT INTO query syntax")
		}
		tableName := matches[1]
		colStr := matches[2]
		valStr := matches[3]

		cols := splitAndTrim(colStr, ",")
		vals := splitAndTrimValues(valStr)

		if len(cols) != len(vals) {
			return fmt.Errorf("column count does not match value count")
		}

		table, ok := db.data[tableName]
		if !ok {
			return fmt.Errorf("table not found: %s", tableName)
		}

		row := make(map[string]Object)
		for i, col := range cols {
			val := vals[i]
			row[col] = parseValueToPHXObject(val)
		}
		db.data[tableName] = append(table, row)
		return db.saveToJSON()
	}

	if strings.HasPrefix(upperQ, "DELETE FROM") {
		re := regexp.MustCompile(`(?i)DELETE\s+FROM\s+(\w+)(?:\s+WHERE\s+(.+))?`)
		matches := re.FindStringSubmatch(q)
		if len(matches) < 2 {
			return fmt.Errorf("invalid DELETE query")
		}
		tableName := matches[1]
		whereClause := ""
		if len(matches) > 2 {
			whereClause = matches[2]
		}

		table, ok := db.data[tableName]
		if !ok {
			return fmt.Errorf("table not found: %s", tableName)
		}

		if whereClause == "" {
			db.data[tableName] = []map[string]Object{}
		} else {
			filtered := []map[string]Object{}
			col, val, err := parseWhereClause(whereClause)
			if err != nil {
				return err
			}
			for _, row := range table {
				rowVal, ok := row[col]
				if ok && matchValue(rowVal, val) {
					continue
				}
				filtered = append(filtered, row)
			}
			db.data[tableName] = filtered
		}
		return db.saveToJSON()
	}

	return fmt.Errorf("unsupported write query: %s", query)
}

func (db *DatabaseObject) executeQuery(query string) (Object, error) {
	q := strings.TrimSpace(query)
	upperQ := strings.ToUpper(q)

	if strings.HasPrefix(upperQ, "SELECT") {
		re := regexp.MustCompile(`(?i)SELECT\s+\*\s+FROM\s+(\w+)(?:\s+WHERE\s+(.+))?`)
		matches := re.FindStringSubmatch(q)
		if len(matches) < 2 {
			return nil, fmt.Errorf("invalid SELECT query")
		}
		tableName := matches[1]
		whereClause := ""
		if len(matches) > 2 {
			whereClause = matches[2]
		}

		table, ok := db.data[tableName]
		if !ok {
			return nil, fmt.Errorf("table not found: %s", tableName)
		}

		results := []*ArrayPair{}
		idx := int64(0)

		var col, val string
		var err error
		if whereClause != "" {
			col, val, err = parseWhereClause(whereClause)
			if err != nil {
				return nil, err
			}
		}

		for _, row := range table {
			if whereClause != "" {
				rowVal, ok := row[col]
				if !ok || !matchValue(rowVal, val) {
					continue
				}
			}

			rowPairs := []*ArrayPair{}
			for colName, colVal := range row {
				rowPairs = append(rowPairs, &ArrayPair{
					Key:   &String{Value: colName},
					Value: colVal,
				})
			}
			rowArray := &Array{Pairs: rowPairs}

			results = append(results, &ArrayPair{
				Key:   &Integer{Value: idx},
				Value: rowArray,
			})
			idx++
		}

		return &Array{Pairs: results}, nil
	}

	return nil, fmt.Errorf("unsupported read query: %s", query)
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	res := []string{}
	for _, p := range parts {
		res = append(res, strings.TrimSpace(p))
	}
	return res
}

func splitAndTrimValues(s string) []string {
	vals := []string{}
	var current strings.Builder
	inQuotes := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '\'' {
			inQuotes = !inQuotes
			current.WriteByte(ch)
		} else if ch == ',' && !inQuotes {
			vals = append(vals, strings.TrimSpace(current.String()))
			current.Reset()
		} else {
			current.WriteByte(ch)
		}
	}
	vals = append(vals, strings.TrimSpace(current.String()))
	return vals
}

func parseValueToPHXObject(val string) Object {
	if strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'") {
		return &String{Value: val[1 : len(val)-1]}
	}
	if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
		return &String{Value: val[1 : len(val)-1]}
	}

	if val == "true" || val == "TRUE" {
		return &Boolean{Value: true}
	}
	if val == "false" || val == "FALSE" {
		return &Boolean{Value: false}
	}

	if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
		return &Integer{Value: intVal}
	}

	if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
		return &Float{Value: floatVal}
	}

	return &String{Value: val}
}

func parseWhereClause(clause string) (string, string, error) {
	parts := strings.Split(clause, "=")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid WHERE clause")
	}
	col := strings.TrimSpace(parts[0])
	val := strings.TrimSpace(parts[1])
	if strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'") {
		val = val[1 : len(val)-1]
	}
	return col, val, nil
}

func matchValue(obj Object, matchVal string) bool {
	switch v := obj.(type) {
	case *String:
		return v.Value == matchVal
	case *Integer:
		return fmt.Sprintf("%d", v.Value) == matchVal
	case *Float:
		return fmt.Sprintf("%g", v.Value) == matchVal
	case *Boolean:
		return fmt.Sprintf("%t", v.Value) == matchVal
	default:
		return obj.Inspect() == matchVal
	}
}

type MySQLObject struct {
	dbConn    *sql.DB
	connected bool
}

func (m *MySQLObject) Type() ObjectType { return INSTANCE_OBJ }
func (m *MySQLObject) Inspect() string  { return "MySQLConnection" }

func (m *MySQLObject) connect(host, user, password, db string) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", user, password, host, db)
	conn, err := sql.Open("mysql", dsn)
	if err == nil {
		err = conn.Ping()
	}
	if err != nil {
		// Try to create database if not exists
		dsnNoDB := fmt.Sprintf("%s:%s@tcp(%s)/?parseTime=true", user, password, host)
		connNoDB, err2 := sql.Open("mysql", dsnNoDB)
		if err2 == nil {
			if err2 = connNoDB.Ping(); err2 == nil {
				_, _ = connNoDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", db))
				connNoDB.Close()
				conn, err = sql.Open("mysql", dsn)
				if err == nil {
					err = conn.Ping()
				}
			}
		}
	}
	if err != nil {
		return err
	}
	m.dbConn = conn
	m.connected = true
	return nil
}

func (m *MySQLObject) executeExec(query string) error {
	_, err := m.dbConn.Exec(query)
	return err
}

func (m *MySQLObject) executeQuery(query string) (Object, error) {
	rows, err := m.dbConn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSQLRows(rows)
}

func (m *MySQLObject) close() {
	if m.dbConn != nil {
		m.dbConn.Close()
		m.dbConn = nil
	}
	m.connected = false
}

type PostgresObject struct {
	dbConn    *sql.DB
	connected bool
}

func (p *PostgresObject) Type() ObjectType { return INSTANCE_OBJ }
func (p *PostgresObject) Inspect() string  { return "PostgreSQLConnection" }

func (p *PostgresObject) connect(host, user, password, db string, port int64) error {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, db)
	conn, err := sql.Open("postgres", dsn)
	if err == nil {
		err = conn.Ping()
	}
	if err != nil {
		// Try to create DB
		dsnNoDB := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable", host, port, user, password)
		connNoDB, err2 := sql.Open("postgres", dsnNoDB)
		if err2 == nil {
			if err2 = connNoDB.Ping(); err2 == nil {
				var exists bool
				_ = connNoDB.QueryRow("SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = $1)", db).Scan(&exists)
				if !exists {
					_, _ = connNoDB.Exec(fmt.Sprintf("CREATE DATABASE %s", db))
				}
				connNoDB.Close()
				conn, err = sql.Open("postgres", dsn)
				if err == nil {
					err = conn.Ping()
				}
			}
		}
	}
	if err != nil {
		return err
	}
	p.dbConn = conn
	p.connected = true
	return nil
}

func (p *PostgresObject) executeExec(query string) error {
	_, err := p.dbConn.Exec(query)
	return err
}

func (p *PostgresObject) executeQuery(query string) (Object, error) {
	rows, err := p.dbConn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSQLRows(rows)
}

func (p *PostgresObject) close() {
	if p.dbConn != nil {
		p.dbConn.Close()
		p.dbConn = nil
	}
	p.connected = false
}

func scanSQLRows(rows *sql.Rows) (Object, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	results := []*ArrayPair{}
	idx := int64(0)

	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		rowPairs := []*ArrayPair{}
		for i, colName := range cols {
			val := columns[i]
			var phxObj Object
			if val == nil {
				phxObj = NULL
			} else {
				switch v := val.(type) {
				case int64:
					phxObj = &Integer{Value: v}
				case int:
					phxObj = &Integer{Value: int64(v)}
				case float64:
					phxObj = &Float{Value: v}
				case bool:
					phxObj = &Boolean{Value: v}
				case []byte:
					phxObj = &String{Value: string(v)}
				case string:
					phxObj = &String{Value: v}
				default:
					phxObj = &String{Value: fmt.Sprintf("%v", v)}
				}
			}

			rowPairs = append(rowPairs, &ArrayPair{
				Key:   &String{Value: colName},
				Value: phxObj,
			})
		}
		rowArray := &Array{Pairs: rowPairs}
		results = append(results, &ArrayPair{
			Key:   &Integer{Value: idx},
			Value: rowArray,
		})
		idx++
	}

	return &Array{Pairs: results}, nil
}

type MongoObject struct {
	uri         string
	connected   bool
	dbName      string
	collName    string
	collections map[string][]Object
}

func (m *MongoObject) Type() ObjectType { return INSTANCE_OBJ }
func (m *MongoObject) Inspect() string  { return "MongoDBConnection" }



