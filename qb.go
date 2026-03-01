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
	nextOr  bool
}

// Clause represents an intermediate state for building a query condition.
type Clause struct {
	qb    *QB
	field string
}

// Where starts a new condition clause for the given column.
func (qb *QB) Where(column string) *Clause {
	return &Clause{qb: qb, field: column}
}

// Or sets the next condition to use OR logic instead of AND.
func (qb *QB) Or() *QB {
	qb.nextOr = true
	return qb
}

func (qb *QB) addCondition(c Condition) *QB {
	if qb.nextOr {
		c.logic = "OR"
		qb.nextOr = false
	} else {
		c.logic = "AND"
	}
	qb.conds = append(qb.conds, c)
	return qb
}

// Eq creates an equality condition.
func (c *Clause) Eq(value any) *QB {
	return c.qb.addCondition(Eq(c.field, value))
}

// Neq creates an inequality condition.
func (c *Clause) Neq(value any) *QB {
	return c.qb.addCondition(Neq(c.field, value))
}

// Gt creates a greater-than condition.
func (c *Clause) Gt(value any) *QB {
	return c.qb.addCondition(Gt(c.field, value))
}

// Gte creates a greater-than-or-equal condition.
func (c *Clause) Gte(value any) *QB {
	return c.qb.addCondition(Gte(c.field, value))
}

// Lt creates a less-than condition.
func (c *Clause) Lt(value any) *QB {
	return c.qb.addCondition(Lt(c.field, value))
}

// Lte creates a less-than-or-equal condition.
func (c *Clause) Lte(value any) *QB {
	return c.qb.addCondition(Lte(c.field, value))
}

// Like creates a LIKE condition.
func (c *Clause) Like(value any) *QB {
	return c.qb.addCondition(Like(c.field, value))
}

// In creates an IN condition.
func (c *Clause) In(value any) *QB {
	return c.qb.addCondition(In(c.field, value))
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

// OrderClause represents an intermediate state for building an order by clause.
type OrderClause struct {
	qb    *QB
	field string
}

// OrderBy starts a new order clause for the given column.
func (qb *QB) OrderBy(column string) *OrderClause {
	return &OrderClause{qb: qb, field: column}
}

// Asc sets the order direction to ascending.
func (o *OrderClause) Asc() *QB {
	o.qb.orderBy = append(o.qb.orderBy, Order{column: o.field, dir: "ASC"})
	return o.qb
}

// Desc sets the order direction to descending.
func (o *OrderClause) Desc() *QB {
	o.qb.orderBy = append(o.qb.orderBy, Order{column: o.field, dir: "DESC"})
	return o.qb
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
	plan, err := qb.db.compiler.Compile(q, qb.model)
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
func (qb *QB) ReadAll(new func() Model, onRow func(Model)) error {
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
	plan, err := qb.db.compiler.Compile(q, qb.model)
	if err != nil {
		return err
	}

	rows, err := qb.db.exec.Query(plan.Query, plan.Args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		m := new()
		if err := rows.Scan(m.Pointers()...); err != nil {
			return err
		}
		onRow(m)
	}
	return rows.Err()
}
