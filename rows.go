package sqlite

type Rows struct {
	stmt *Stmt
	err  error
}

func (r *Rows) Next() bool {
	stmt := r.stmt
	if r.err != nil {
		return false
	}

	hasMore, err := stmt.Step()
	if err != nil {
		r.err = err
		return false
	}

	return hasMore
}

func (r *Rows) Scan(dst ...interface{}) bool {
	stmt := r.stmt
	for i, v := range dst {
		if err := stmt.scan(i, v); err != nil {
			r.err = err
			return false
		}
	}

	return true
}

func (r Rows) Error() error {
	return r.err
}

func (r Rows) Close() {
	// will be nil if the query was never valid
	if stmt := r.stmt; stmt != nil {
		stmt.Close()
	}
}
