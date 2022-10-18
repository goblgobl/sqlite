package sqlite

type Rows struct {
	Stmt *Stmt
	err  error
}

func (r *Rows) Next() bool {
	stmt := r.Stmt
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

func (r *Rows) Scan(dst ...interface{}) error {
	stmt := r.Stmt
	for i, v := range dst {
		if err := stmt.scan(i, v); err != nil {
			r.err = err
			return err
		}
	}

	return nil
}

func (r Rows) Error() error {
	return r.err
}

func (r Rows) Close() {
	// will be nil if the query was never valid
	if stmt := r.Stmt; stmt != nil {
		stmt.Close()
	}
}
