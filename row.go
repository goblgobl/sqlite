package sqlite

type Row struct {
	stmt *Stmt
	err  error
}

func (r Row) Scan(dst ...interface{}) (bool, error) {
	if err := r.err; err != nil {
		return false, err
	}
	stmt := r.stmt
	defer stmt.Close()

	hasRow, err := stmt.Step()
	if err != nil {
		return false, err
	}

	if !hasRow {
		return false, nil
	}

	for i, v := range dst {
		if err := stmt.scan(i, v); err != nil {
			return false, err
		}
	}
	return true, nil
}
