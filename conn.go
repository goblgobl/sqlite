package sqlite

/*
#cgo CFLAGS: -std=gnu99
#cgo CFLAGS: -O2

// MISC
#cgo CFLAGS: -DNDEBUG=1
#cgo CFLAGS: -DSQLITE_DQS=0
#cgo CFLAGS: -DSQLITE_CORE=1
#cgo CFLAGS: -DSQLITE_DEFAULT_MEMSTATUS=0
#cgo CFLAGS: -DSQLITE_DEFAULT_WAL_SYNCHRONOUS=1
#cgo CFLAGS: -DSQLITE_USE_URI=1
#cgo CFLAGS: -DSQLITE_USE_ALLOCA=1
#cgo CFLAGS: -DSQLITE_THREADSAFE=2
#cgo CFLAGS: -DSQLITE_TEMP_STORE=3

// FEATURES
#cgo CFLAGS: -DSQLITE_SOUNDEX=1
#cgo CFLAGS: -DSQLITE_ENABLE_API_ARMOR=1
#cgo CFLAGS: -DSQLITE_ENABLE_FTS5=1
#cgo CFLAGS: -DSQLITE_ENABLE_GEOPOLY=1
#cgo CFLAGS: -DSQLITE_ENABLE_JSON1=1
#cgo CFLAGS: -DSQLITE_ENABLE_PREUPDATE_HOOK
#cgo CFLAGS: -DSQLITE_ENABLE_PREUPDATE_HOOK
#cgo CFLAGS: -DSQLITE_ENABLE_RTREE=1
#cgo CFLAGS: -DSQLITE_ENABLE_SESSION
#cgo CFLAGS: -DSQLITE_ENABLE_STAT4=1
#cgo CFLAGS: -DSQLITE_ENABLE_UNLOCK_NOTIFY
#cgo CFLAGS: -DSQLITE_ENABLE_UPDATE_DELETE_LIMIT=1

// LIMITS
#cgo CFLAGS: -DSQLITE_MAX_ATTACHED=0
#cgo CFLAGS: -DSQLITE_MAX_COLUMN=200
#cgo CFLAGS: -DSQLITE_MAX_COMPOUND_SELECT=10
#cgo CFLAGS: -DSQLITE_MAX_EXPR_DEPTH=50
#cgo CFLAGS: -DSQLITE_MAX_FUNCTION_ARG=16
#cgo CFLAGS: -DSQLITE_MAX_LENGTH=1000000
#cgo CFLAGS: -DSQLITE_MAX_LIKE_PATTERN_LENGTH=100
#cgo CFLAGS: -DSQLITE_MAX_SQL_LENGTH=5000
#cgo CFLAGS: -DSQLITE_MAX_TRIGGER_DEPTH=10
#cgo CFLAGS: -DSQLITE_MAX_VARIABLE_NUMBER=200
#cgo CFLAGS: -DSQLITE_MAX_VDBE_OP=25000
#cgo CFLAGS: -DSQLITE_MAX_SQL_LENGTH=5000
#cgo CFLAGS: -DSQLITE_MAX_ALLOCATION_SIZE=20971520

// OTHER HARDENING
#cgo CFLAGS: -DSQLITE_PRINTF_PRECISION_LIMIT=10000
#cgo CFLAGS: -DSQLITE_DEFAULT_FILE_PERMISSIONS=0600

// DISABLED
#cgo CFLAGS: -DSQLITE_LIKE_DOESNT_MATCH_BLOBS
#cgo CFLAGS: -DSQLITE_OMIT_AUTOINIT=1
#cgo CFLAGS: -DSQLITE_OMIT_DECLTYPE
#cgo CFLAGS: -DSQLITE_OMIT_DEPRECATED=1
#cgo CFLAGS: -DSQLITE_OMIT_LOAD_EXTENSION=1
#cgo CFLAGS: -DSQLITE_OMIT_PROGRESS_CALLBACK=1
#cgo CFLAGS: -DSQLITE_OMIT_SHARED_CACHE
#cgo CFLAGS: -DSQLITE_OMIT_TRACE=1
#cgo CFLAGS: -DSQLITE_OMIT_UTF16=1

// PLATFORM FEATURES
#cgo linux LDFLAGS: -lm
#cgo openbsd LDFLAGS: -lm
#cgo linux,!android CFLAGS: -DHAVE_FDATASYNC=1
#cgo linux,!android CFLAGS: -DHAVE_PREAD=1 -DHAVE_PWRITE=1
#cgo darwin CFLAGS: -DHAVE_FDATASYNC=1
#cgo darwin CFLAGS: -DHAVE_PREAD=1 -DHAVE_PWRITE=1
#cgo windows LDFLAGS: -Wl,-Bstatic -lwinpthread -Wl,-Bdynamic

// Fix for BusyTimeout on *nix systems.
#cgo !windows CFLAGS: -DHAVE_USLEEP=1

// Fix "_localtime32(0): not defined" linker error.
#cgo windows,386 CFLAGS: -D_localtime32=localtime

#include "sqlite3.h"
#include <strings.h>

static int enable_defensive(sqlite3 *db) {
	return sqlite3_db_config(db, SQLITE_DBCONFIG_DEFENSIVE, 1, (void*)0);
}

static int sqlkite_user_stmt(sqlite3_context *context, const char *sql, sqlite3_stmt **stmt) {
	int rc;
	sqlite3 *db = sqlite3_context_db_handle(context);

	rc = sqlite3_prepare_v2(db, sql, -1, stmt, 0);
	if (rc != SQLITE_OK) {
		const char* errMsg = sqlite3_mprintf("sqlkite_user.prepare - %s", sqlite3_errmsg(db));
		sqlite3_result_error(context, errMsg, -1);
		sqlite3_free((void *)errMsg);
		return rc;
	}

	rc = sqlite3_step(*stmt);
	if (rc == SQLITE_ROW) {
		return rc;
	}
	if (rc == SQLITE_DONE) {
		// this is fine, but we can close stmt for the caller
		sqlite3_finalize(*stmt);
		return rc;
	}

	// an error
	sqlite3_finalize(*stmt);
	const char* errMsg = sqlite3_mprintf("sqlkite_user.step - %s", sqlite3_errmsg(db));
	sqlite3_result_error(context, errMsg, -1);
	sqlite3_free((void *)errMsg);
	return rc;
}

static void sqlkite_user_result(sqlite3_context *context, const char *sql){
	sqlite3_stmt *stmt;

	// Will sqlite3_result_error if needed, and finalize stmt unless there's a row
	int rc = sqlkite_user_stmt(context, sql, &stmt);

	if (rc == SQLITE_ROW) {
		size_t len = sqlite3_column_bytes(stmt, 0);
		const char *value = (const char*)sqlite3_column_text(stmt, 0);
		sqlite3_result_text(context, value, len, SQLITE_TRANSIENT);
		sqlite3_finalize(stmt);
	} else if (rc == SQLITE_DONE) {
		sqlite3_result_null(context);
	}
}

static char *sqlkite_user_value(sqlite3_context *context, const char *sql){
	sqlite3_stmt *stmt;
	char *value = NULL;

	// Will sqlite3_result_error if needed, and finalize stmt unless there's a row
	int rc = sqlkite_user_stmt(context, sql, &stmt);
	if (rc == SQLITE_ROW) {
		size_t len = sqlite3_column_bytes(stmt, 0);
		if (len != 0) {
			value = (char*)sqlite3_malloc64(len);
			if (value) {
				memcpy(value, sqlite3_column_text(stmt, 0), len);
			} else {
				sqlite3_result_error_nomem(context);
			}
		}
		sqlite3_finalize(stmt);
	}
	return value;
}

static void sqlkite_assert_value(sqlite3_context *context, const char *sql, int argc, sqlite3_value **argv){
	if (sqlite3_value_type(argv[0]) != SQLITE_TEXT) {
		sqlite3_result_error(context, "sqlkite_assert requires a text argument", -1);
		return;
	}

	int valid = -1;
	const char *actual = sqlkite_user_value(context, sql);
	if (actual) {
		const char *target = (const char*)sqlite3_value_text(argv[0]);
		valid = sqlite3_stricmp(actual, target);
		sqlite3_free((void *)actual);
	}

	if (valid != 0) {
		sqlite3_result_error(context, "sqlkite_row_access", -1);
	}
}

static void sqlkite_user_id(sqlite3_context *context, int argc, sqlite3_value **argv){
	return sqlkite_user_result(context, "select user_id from sqlkite_user");
}

static void sqlkite_user_role(sqlite3_context *context, int argc, sqlite3_value **argv){
	return sqlkite_user_result(context, "select role from sqlkite_user");
}

static void sqlkite_assert_user_id(sqlite3_context *context, int argc, sqlite3_value **argv){
	sqlkite_assert_value(context, "select user_id from sqlkite_user", argc, argv);
}

static void sqlkite_assert_user_role(sqlite3_context *context, int argc, sqlite3_value **argv){
	sqlkite_assert_value(context, "select role from sqlkite_user", argc, argv);
}

static char *sqlkite_escape_literal(char *value){
	return sqlite3_mprintf("%Q", value);
}

static int registerFunctions(sqlite3 *db) {
	int rc;
	rc = sqlite3_create_function(db, "sqlkite_user_id", 0, SQLITE_UTF8, NULL, &sqlkite_user_id, NULL, NULL);
	if (rc != SQLITE_OK) {
		return rc;
	}

	rc = sqlite3_create_function(db, "sqlkite_user_role", 0, SQLITE_UTF8, NULL, &sqlkite_user_role, NULL, NULL);
	if (rc != SQLITE_OK) {
		return rc;
	}

	rc = sqlite3_create_function(db, "sqlkite_assert_user_id", 1, SQLITE_UTF8, NULL, &sqlkite_assert_user_id, NULL, NULL);
	if (rc != SQLITE_OK) {
		return rc;
	}

	rc = sqlite3_create_function(db, "sqlkite_assert_user_role", 1, SQLITE_UTF8, NULL, &sqlkite_assert_user_role, NULL, NULL);
	if (rc != SQLITE_OK) {
		return rc;
	}
	return SQLITE_OK;
}
*/
import "C"

import (
	"errors"
	"os"
	"reflect"
	"time"
	"unsafe"
)

var (
	txBegin          = cStr(Terminate("begin"))
	txBeginExclusive = cStr(Terminate("begin exclusive"))
	txCommit         = cStr(Terminate("commit"))
	txRollback       = cStr(Terminate("rollback"))

	ErrNoRows = errors.New("no rows in result set")
)

func init() {
	if rc := C.sqlite3_initialize(); rc != C.SQLITE_OK {
		panic(errorFromCode(nil, rc))
	}
}

type Scanner interface {
	Scan(dest ...any) error
}

func Terminate(sql string) string {
	return sql + "\x00"
}

type Conn struct {
	db *C.sqlite3
}

func Memory() (Conn, error) {
	return Open(":memory:", false)
}

func EscapeLiteral(value string) string {
	str := C.sqlkite_escape_literal(C.CString(value))
	escaped := C.GoString(str)
	C.sqlite3_free(unsafe.Pointer(str))
	return escaped
}

func Open(name string, create bool) (Conn, error) {
	name = Terminate(name)

	flags := C.SQLITE_OPEN_READWRITE | C.SQLITE_OPEN_EXRESCODE
	if create {
		flags |= C.SQLITE_OPEN_CREATE
	}

	var db *C.sqlite3
	rc := C.sqlite3_open_v2(cStr(name), &db, C.int(flags), nil)

	if rc != C.SQLITE_OK {
		C.sqlite3_close_v2(db)
		if !create && rc == C.SQLITE_CANTOPEN {
			return Conn{}, os.ErrNotExist
		}
		return Conn{}, errorFromCode(nil, rc)
	}

	if rc = C.enable_defensive(db); rc != C.SQLITE_OK {
		err := errorFromCode(db, rc)
		C.sqlite3_close_v2(db)
		return Conn{}, err
	}

	if rc := C.registerFunctions(db); rc != C.SQLITE_OK {
		err := errorFromCode(db, rc)
		C.sqlite3_close_v2(db)
		return Conn{}, err
	}

	return Conn{db: db}, nil
}

func (c *Conn) Close() error {
	if db := c.db; db != nil {
		if rc := C.sqlite3_close_v2(db); rc != C.SQLITE_OK {
			return errorFromCode(db, rc)
		}
	}
	return nil
}

func (c Conn) Prepare(sql []byte, args ...any) (*Stmt, error) {
	db := c.db
	var stmt *C.sqlite3_stmt
	rc := C.sqlite3_prepare_v2(db, cStrFromBytes(sql), C.int(len(sql)), &stmt, nil)
	if rc != C.SQLITE_OK {
		return nil, prepareError(db, rc, string(sql), args)
	}

	if stmt == nil {
		return nil, nil
	}

	cColumnCount := C.sqlite3_column_count(stmt)
	columnCount := int(cColumnCount)
	columnTypes := make([]byte, columnCount)
	s := &Stmt{
		db:           db,
		stmt:         stmt,
		columnTypes:  columnTypes,
		columnCount:  columnCount,
		cColumnCount: cColumnCount,
		cColumnTypes: (*C.uchar)(cBytes(columnTypes)),
	}

	if len(args) > 0 {
		if err := s.Bind(args...); err != nil {
			s.Close()
			return nil, err
		}
	}

	return s, nil
}

func (c Conn) RowB(sql []byte, args ...any) Row {
	stmt, err := c.Prepare(sql, args...)
	return Row{Stmt: stmt, err: err}
}

func (c Conn) Row(sql string, args ...any) Row {
	return c.RowB(s2b(sql), args...)
}

func (c Conn) RowsB(sql []byte, args ...any) Rows {
	stmt, err := c.Prepare(sql, args...)
	return Rows{Stmt: stmt, err: err}
}

func (c Conn) Rows(sql string, args ...any) Rows {
	return c.RowsB(s2b(sql), args...)
}

func (c Conn) ExecB(sql []byte, args ...any) error {
	if len(args) == 0 {
		return c.ExecTerminated(append(sql, '\x00'))
	}
	return c.execArgs(sql, args...)
}

func (c Conn) ExecTerminated(sql []byte) error {
	return c.exec(cStrFromBytes(sql))
}

func (c Conn) Exec(sql string, args ...any) error {
	if len(args) == 0 {
		return c.exec(cStr(Terminate(sql)))
	}
	return c.execArgs(s2b(sql), args...)
}

func (c Conn) MustExec(sql string, args ...any) {
	if err := c.Exec(sql, args...); err != nil {
		panic(err)
	}
}

func (c Conn) exec(sql *C.char) error {
	if rc := C.sqlite3_exec(c.db, sql, nil, nil, nil); rc != C.SQLITE_OK {
		return errorFromCode(c.db, rc)
	}
	return nil
}

func (c Conn) execArgs(sql []byte, args ...any) error {
	s, err := c.Prepare(sql)
	if err != nil {
		return err
	}
	if s == nil {
		return nil
	}
	defer s.Close()

	if err = s.Bind(args...); err != nil {
		return err
	}

	if err = s.StepToCompletion(); err != nil {
		return err
	}

	return nil
}

func (c Conn) Transaction(f func() error) error {
	if err := c.exec(txBeginExclusive); err != nil {
		return err
	}

	err := f()
	if err != nil {
		err2 := c.exec(txRollback)
		if err2 == nil {
			return err
		}
		return err
	}

	if err = c.exec(txCommit); err != nil {
		return err
	}
	return nil
}

func (c Conn) LastInsertRowID() int {
	return int(C.sqlite3_last_insert_rowid(c.db))
}

func (c Conn) Changes() int {
	return int(C.sqlite3_changes(c.db))
}

func (c Conn) BusyTimeout(d time.Duration) {
	C.sqlite3_busy_timeout(c.db, C.int(d.Milliseconds()))
}

func cStr(s string) *C.char {
	h := (*reflect.StringHeader)(unsafe.Pointer(&s))
	return (*C.char)(unsafe.Pointer(h.Data))
}

func cStrFromBytes(b []byte) *C.char {
	h := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	return (*C.char)(unsafe.Pointer(h.Data))
}

func cBytes(b []byte) unsafe.Pointer {
	h := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	return unsafe.Pointer(h.Data)
}

func s2b(s string) (b []byte) {
	/* #nosec G103 */
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	/* #nosec G103 */
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh.Data = sh.Data
	bh.Cap = sh.Len
	bh.Len = sh.Len
	return b
}
