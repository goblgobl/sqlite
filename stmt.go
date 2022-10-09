package sqlite

/*
#include "sqlite3.h"

// Retrieve all columns types
static void column_types(sqlite3_stmt *s, unsigned char p[], int n) {
	for (int i = 0; i < n; ++i, ++p) {
		*p = sqlite3_column_type(s, i);
	}
}

static int empty_string(sqlite3_stmt *s, int i) {
	return sqlite3_bind_text(s, i, "", 0, SQLITE_STATIC);
}
*/
import "C"

import (
	"fmt"
	"reflect"
	"time"
	"unsafe"
)

// When reading values into a RawByte, the slice is owned by sqlite and will
// only be valid until the next call on the statement is made.
type RawBytes []byte

type Option[T any] struct {
	Value T
	Valid bool
}

type Stmt struct {
	stmt         *C.sqlite3_stmt
	db           *C.sqlite3
	columnCount  int
	columnTypes  []byte
	cColumnTypes *C.uchar
	cColumnCount C.int
}

func (s *Stmt) Close() error {
	rc := C.sqlite3_finalize(s.stmt)
	if rc != C.SQLITE_OK {
		return errorFromCode(s.db, rc)
	}
	return nil
}

func (s *Stmt) Exec(args ...interface{}) error {
	if err := s.Bind(args...); err != nil {
		s.Reset()
		return err
	}

	if err := s.StepToCompletion(); err != nil {
		s.Reset()
		return err
	}

	if err := s.Reset(); err != nil {
		return err
	}

	return nil
}

func (s *Stmt) Bind(args ...interface{}) error {
	stmt := s.stmt
	for i, v := range args {
		var rc C.int
		bindIndex := C.int(i + 1)

		if v == nil {
			rc = C.sqlite3_bind_null(stmt, bindIndex)
			if rc != C.SQLITE_OK {
				return errorFromCode(s.db, rc)
			}
			continue
		}
		switch v := v.(type) {
		case int:
			rc = C.sqlite3_bind_int64(stmt, bindIndex, C.sqlite3_int64(v))
		case string:
			if v == "" {
				rc = C.empty_string(stmt, bindIndex)
			} else {
				rc = C.sqlite3_bind_text(stmt, bindIndex, cStr(v), C.int(len(v)), C.SQLITE_TRANSIENT)
			}
		case uint16:
			rc = C.sqlite3_bind_int64(stmt, bindIndex, C.sqlite3_int64(int64(v)))
		case uint32:
			rc = C.sqlite3_bind_int64(stmt, bindIndex, C.sqlite3_int64(int64(v)))
		case uint64:
			// OMG!!
			rc = C.sqlite3_bind_int64(stmt, bindIndex, C.sqlite3_int64(int64(v)))
		case int64:
			rc = C.sqlite3_bind_int64(stmt, bindIndex, C.sqlite3_int64(v))
		case float64:
			rc = C.sqlite3_bind_double(stmt, bindIndex, C.double(v))
		case bool:
			var sqliteBool int64
			if v {
				sqliteBool = 1
			}
			rc = C.sqlite3_bind_int64(stmt, bindIndex, C.sqlite3_int64(sqliteBool))
		case []byte:
			if len(v) == 0 {
				rc = C.sqlite3_bind_zeroblob(stmt, bindIndex, 0)
			} else {
				rc = C.sqlite3_bind_blob(stmt, bindIndex, cBytes(v), C.int(len(v)), C.SQLITE_TRANSIENT)
			}
		case time.Time:
			rc = C.sqlite3_bind_int64(stmt, bindIndex, C.sqlite3_int64(v.Unix()))
		default:
			return Error{Code: C.SQLITE_MISUSE, Message: fmt.Sprintf("unsupported type %T (index: %d)", v, i)}
		}
		if rc != C.SQLITE_OK {
			return errorFromCode(s.db, rc)
		}
	}
	return nil
}

func (s *Stmt) Scan(dst ...interface{}) error {
	for i, v := range dst {
		if err := s.scan(i, v); err != nil {
			return err
		}
	}
	return nil
}

func (s *Stmt) Row(dst ...interface{}) (bool, error) {
	hasRow, err := s.Step()
	if err != nil {
		return false, err
	}
	if !hasRow {
		return false, nil
	}
	for i, v := range dst {
		if err := s.scan(i, v); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (s *Stmt) Reset() error {
	if rc := C.sqlite3_reset(s.stmt); rc != C.SQLITE_OK {
		return errorFromCode(s.db, rc)
	}
	return nil
}

func (s *Stmt) ClearBindings() error {
	if rc := C.sqlite3_clear_bindings(s.stmt); rc != C.SQLITE_OK {
		return errorFromCode(s.db, rc)
	}
	return nil
}

func (s *Stmt) Step() (bool, error) {
	stmt := s.stmt
	rc := C.sqlite3_step(stmt)
	if rc == C.SQLITE_ROW {
		C.column_types(stmt, s.cColumnTypes, s.cColumnCount)
		return true, nil
	}

	if rc == C.SQLITE_DONE {
		return false, nil
	}

	return false, errorFromCode(s.db, rc)
}

func (s *Stmt) StepToCompletion() error {
	stmt := s.stmt
	for {
		rc := C.sqlite3_step(stmt)
		if rc == C.SQLITE_ROW {
			continue
		}
		if rc == C.SQLITE_DONE {
			break
		}
		return errorFromCode(s.db, rc)

	}
	return nil
}

func (s *Stmt) ColumnTypes() []byte {
	return s.columnTypes
}

func (s *Stmt) scan(i int, v interface{}) error {
	var err error
	switch v := v.(type) {
	case *string:
		*v, err = s.ColumnText(i)
	case *Option[string]:
		if s.columnTypes[i] == C.SQLITE_NULL {
			*v = Option[string]{}
		} else {
			var n string
			n, err = s.ColumnText(i)
			*v = Option[string]{Value: n, Valid: true}
		}
	case *int:
		*v = s.ColumnInt(i)
	case *Option[int]:
		if s.columnTypes[i] == C.SQLITE_NULL {
			*v = Option[int]{}
		} else {
			n := s.ColumnInt64(i)
			*v = Option[int]{Value: int(n), Valid: true}
		}
	case *int64:
		*v = s.ColumnInt64(i)
	case *Option[int64]:
		if s.columnTypes[i] == C.SQLITE_NULL {
			*v = Option[int64]{}
		} else {
			n := s.ColumnInt64(i)
			*v = Option[int64]{Value: n, Valid: true}
		}
	case *float64:
		*v = s.ColumnDouble(i)
	case *Option[float64]:
		if s.columnTypes[i] == C.SQLITE_NULL {
			*v = Option[float64]{}
		} else {
			n := s.ColumnDouble(i)
			*v = Option[float64]{Value: n, Valid: true}
		}
	case *bool:
		*v = s.ColumnInt64(i) != 0
	case *Option[bool]:
		if s.columnTypes[i] == C.SQLITE_NULL {
			*v = Option[bool]{}
		} else {
			n := s.ColumnInt64(i) != 0
			*v = Option[bool]{Value: n, Valid: true}
		}
	case *[]byte:
		*v, err = s.ColumnBytes(i)
	case *RawBytes:
		*v, err = s.ColumnRawBytes(i)
	case *time.Time:
		*v = time.Unix(s.ColumnInt64(i), 0)
	case *Option[time.Time]:
		if s.columnTypes[i] == C.SQLITE_NULL {
			*v = Option[time.Time]{}
		} else {
			n := s.ColumnInt64(i)
			*v = Option[time.Time]{Value: time.Unix(n, 0), Valid: true}
		}
	default:
		return Error{Code: C.SQLITE_MISUSE, Message: fmt.Sprintf("cannot scan into %T (index: %d)", v, i)}
	}
	if err != nil {
		return err
	}
	return nil
}

func (s *Stmt) ColumnBytes(i int) ([]byte, error) {
	if s.columnTypes[i] == C.SQLITE_NULL {
		return nil, nil
	}

	n := C.sqlite3_column_bytes(s.stmt, C.int(i))
	if n == 0 {
		return nil, nil
	}

	p := C.sqlite3_column_blob(s.stmt, C.int(i))
	if p == nil {
		db := s.db
		rc := C.sqlite3_errcode(db)
		return nil, errorFromCode(db, rc)
	}

	return C.GoBytes(p, n), nil
}

func (s *Stmt) ColumnRawBytes(i int) (RawBytes, error) {
	if s.columnTypes[i] == C.SQLITE_NULL {
		return nil, nil
	}

	n := int(C.sqlite3_column_bytes(s.stmt, C.int(i)))
	if n == 0 {
		return nil, nil
	}

	p := C.sqlite3_column_blob(s.stmt, C.int(i))
	if p == nil {
		db := s.db
		rc := C.sqlite3_errcode(db)
		return nil, errorFromCode(db, rc)
	}

	var b RawBytes
	h := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	h.Data = uintptr(p)
	h.Len = n
	h.Cap = n
	return b, nil
}

func (s *Stmt) ColumnDouble(i int) float64 {
	return float64(C.sqlite3_column_double(s.stmt, C.int(i)))
}

func (s *Stmt) ColumnInt(i int) int {
	return int(C.sqlite3_column_int64(s.stmt, C.int(i)))
}

func (s *Stmt) ColumnInt64(i int) int64 {
	return int64(C.sqlite3_column_int64(s.stmt, C.int(i)))
}

func (s *Stmt) ColumnText(i int) (string, error) {
	n := C.sqlite3_column_bytes(s.stmt, C.int(i))
	if n == 0 {
		return "", nil
	}

	p := (*C.char)(unsafe.Pointer(C.sqlite3_column_text(s.stmt, C.int(i))))
	if p == nil {
		db := s.db
		rc := C.sqlite3_errcode(db)
		return "", errorFromCode(db, rc)
	}

	return C.GoStringN(p, n), nil
}
