package sqlite

type Row struct {
	Stmt *Stmt
	err  error
}

func (r Row) Map() (map[string]any, error) {
	m := make(map[string]any, r.Stmt.columnCount)
	err := r.MapInto(m)
	return m, err
}

func (r Row) MapInto(m map[string]any) error {
	stmt := r.Stmt
	defer stmt.Close()

	hasRow, err := stmt.Step()
	if err != nil {
		return err
	}

	if !hasRow {
		return ErrNoRows
	}

	return stmt.MapInto(m)
}

func (r Row) Scan(dst ...interface{}) error {
	if err := r.err; err != nil {
		return err
	}
	stmt := r.Stmt
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
