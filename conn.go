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
#cgo CFLAGS: -DHAVE_FDATASYNC=1
#cgo CFLAGS: -DHAVE_PREAD=1 -DHAVE_PWRITE=1
#cgo CFLAGS: -DHAVE_USLEEP=1

#cgo LDFLAGS: -lm

#include "sqlite3.h"

static int enable_defensive(sqlite3 *db) {
	return sqlite3_db_config(db, SQLITE_DBCONFIG_DEFENSIVE, 1, (void*)0);
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
