package sqlite

/*
#include "sqlite3.h"
*/
import "C"

import (
	"fmt"
)

type Error struct {
	Code    int
	Message string
}

func errorFromCode(db *C.sqlite3, rc C.int) error {
	var message string
	if db == nil {
		message = C.GoString(C.sqlite3_errstr(rc))
	} else {
		message = C.GoString(C.sqlite3_errmsg(db))
	}
	return Error{
		Code:    int(rc),
		Message: message,
	}
}

func (err Error) Error() string {
	return fmt.Sprintf("sqlite: %s (code: %d)", err.Message, err.Code)
}

type PrepareError struct {
	sql   string
	args  []any
	error error
}

func prepareError(db *C.sqlite3, rc C.int, sql string, args []any) PrepareError {
	return PrepareError{
		sql:   sql,
		args:  args,
		error: errorFromCode(db, rc),
	}
}

func (e PrepareError) Unwrap() error {
	return e.error
}

func (err PrepareError) Error() string {
	// not using the args for now
	// not sure we should...performance, privacy, ...
	return fmt.Sprintf("%s - %s", err.error.Error(), err.sql)
}
