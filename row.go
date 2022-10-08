package sqlite

type Row struct {
	stmt *Stmt
	err  error
}

func (r Row) Scan(dst ...interface{}) error {
	if err := r.err; err != nil {
		return err
	}
	stmt := r.stmt
	defer stmt.Close()

	hasRow, err := stmt.Step()
	if err != nil {
		return err
	}

	if !hasRow {
		return ErrNoRows
	}

	for i, v := range dst {
		if err := stmt.scan(i, v); err != nil {
			return err
		}
	}
	return nil
}
