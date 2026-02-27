package orm

// Get executes the query and returns a single result of type P (pointer to T).
// It expects T to be the struct type and P to be the pointer type that implements Model.
func Get[T any, P interface{*T; Model}](db *DB, qb *QB) (P, error) {
	if err := validate(ActionReadOne, qb.model); err != nil {
		return nil, err
	}

	q := qb.ToQuery()
	q.Action = ActionReadOne
	q.Limit = 1

	// Instantiate T. new(T) returns *T.
	// Cast to P to satisfy the return type and access Model methods.
	dest := P(new(T))

	scanner := db.adapter.QueryRow(q)
	if err := scanner.Scan(dest.Pointers()...); err != nil {
		return nil, err
	}

	return dest, nil
}

// FindAll executes the query and returns a slice of results of type P (pointer to T).
// It expects T to be the struct type and P to be the pointer type that implements Model.
func FindAll[T any, P interface{*T; Model}](db *DB, qb *QB) ([]P, error) {
	if err := validate(ActionReadAll, qb.model); err != nil {
		return nil, err
	}

	q := qb.ToQuery()
	q.Action = ActionReadAll

	rows, err := db.adapter.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []P
	for rows.Next() {
		dest := P(new(T))
		if err := rows.Scan(dest.Pointers()...); err != nil {
			return nil, err
		}
		results = append(results, dest)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
