package orm

// QB represents a query builder.
// Consumers hold a *QB reference in variables for incremental building.
type QB struct {
	db      *DB
	model   Model
	conds   []Condition
	orderBy []Order
	groupBy []string
	limit   int
	offset  int
}

// Where adds conditions to the query.
func (qb *QB) Where(conds ...Condition) *QB {
	qb.conds = append(qb.conds, conds...)
	return qb
}

// Limit sets the limit for the query.
func (qb *QB) Limit(limit int) *QB {
	qb.limit = limit
	return qb
}

// Offset sets the offset for the query.
func (qb *QB) Offset(offset int) *QB {
	qb.offset = offset
	return qb
}

// OrderBy adds an order clause to the query.
func (qb *QB) OrderBy(column, dir string) *QB {
	qb.orderBy = append(qb.orderBy, Order{column: column, dir: dir})
	return qb
}

// GroupBy adds a group by clause to the query.
func (qb *QB) GroupBy(columns ...string) *QB {
	qb.groupBy = append(qb.groupBy, columns...)
	return qb
}

// ReadOne executes the query and returns a single result.
func (qb *QB) ReadOne() error {
	if err := validate(ActionReadOne, qb.model); err != nil {
		return err
	}
	q := Query{
		Action:     ActionReadOne,
		Table:      qb.model.TableName(),
		Conditions: qb.conds,
		OrderBy:    qb.orderBy,
		GroupBy:    qb.groupBy,
		Limit:      1, // Force limit 1
		Offset:     qb.offset,
	}
	plan, err := qb.db.planner.Plan(q, qb.model)
	if err != nil {
		return err
	}

	row := qb.db.exec.QueryRow(plan.Query, plan.Args...)
	if err := row.Scan(qb.model.Pointers()...); err != nil {
		return err
	}
	return nil
}

// ReadAll executes the query and returns all results.
func (qb *QB) ReadAll(factory func() Model, each func(Model)) error {
	if err := validate(ActionReadAll, qb.model); err != nil {
		return err
	}
	q := Query{
		Action:     ActionReadAll,
		Table:      qb.model.TableName(),
		Conditions: qb.conds,
		OrderBy:    qb.orderBy,
		GroupBy:    qb.groupBy,
		Limit:      qb.limit,
		Offset:     qb.offset,
	}
	plan, err := qb.db.planner.Plan(q, qb.model)
	if err != nil {
		return err
	}

	rows, err := qb.db.exec.Query(plan.Query, plan.Args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		m := factory()
		if err := rows.Scan(m.Pointers()...); err != nil {
			return err
		}
		each(m)
	}
	return rows.Err()
}
