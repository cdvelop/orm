package orm

import "github.com/tinywasm/fmt"

// Get executes the query and returns a single result of type *T.
// T must be a struct type such that *T implements the Model interface.
func Get[T any](db *DB, qb *QB) (*T, error) {
	if err := validate(ActionReadOne, qb.model); err != nil {
		return nil, err
	}

	q := qb.ToQuery()
	q.Action = ActionReadOne
	q.Limit = 1

	dest := new(T)
	m, ok := any(dest).(Model)
	if !ok {
		return nil, fmt.Err("model", "interface", "not", "implemented")
	}

	scanner := db.adapter.QueryRow(q)
	if err := scanner.Scan(m.Pointers()...); err != nil {
		return nil, err
	}

	return dest, nil
}

// FindAll executes the query and returns a slice of results of type *T.
// T must be a struct type such that *T implements the Model interface.
func FindAll[T any](db *DB, qb *QB) ([]*T, error) {
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

	// Fast-fail check before iterating
	if _, ok := any(new(T)).(Model); !ok {
		return nil, fmt.Err("model", "interface", "not", "implemented")
	}

	var results []*T
	for rows.Next() {
		dest := new(T)
		m := any(dest).(Model) // Safe because we checked above
		if err := rows.Scan(m.Pointers()...); err != nil {
			return nil, err
		}
		results = append(results, dest)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
