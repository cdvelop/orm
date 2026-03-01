package orm

// Condition represents a filter for a query.
// It is a sealed value type constructed via helper functions.
type Condition struct {
	field    string
	operator string
	value    any
	logic    string
}

func (c Condition) Field() string    { return c.field }
func (c Condition) Operator() string { return c.operator }
func (c Condition) Value() any       { return c.value }
func (c Condition) Logic() string    { return c.logic }

// Eq creates a condition for checking equality.
func Eq(field string, value any) Condition {
	return Condition{
		field:    field,
		operator: "=",
		value:    value,
		logic:    "AND",
	}
}

// Neq creates a condition for checking inequality.
func Neq(field string, value any) Condition {
	return Condition{
		field:    field,
		operator: "!=",
		value:    value,
		logic:    "AND",
	}
}

// Gt creates a condition for checking if a value is greater than another.
func Gt(field string, value any) Condition {
	return Condition{
		field:    field,
		operator: ">",
		value:    value,
		logic:    "AND",
	}
}

// Gte creates a condition for checking if a value is greater than or equal to another.
func Gte(field string, value any) Condition {
	return Condition{
		field:    field,
		operator: ">=",
		value:    value,
		logic:    "AND",
	}
}

// Lt creates a condition for checking if a value is less than another.
func Lt(field string, value any) Condition {
	return Condition{
		field:    field,
		operator: "<",
		value:    value,
		logic:    "AND",
	}
}

// Lte creates a condition for checking if a value is less than or equal to another.
func Lte(field string, value any) Condition {
	return Condition{
		field:    field,
		operator: "<=",
		value:    value,
		logic:    "AND",
	}
}

// Like creates a condition for checking if a value matches a pattern.
func Like(field string, value any) Condition {
	return Condition{
		field:    field,
		operator: "LIKE",
		value:    value,
		logic:    "AND",
	}
}

// In creates a condition for checking if a value is in a list of values.
func In(field string, value any) Condition {
	return Condition{
		field:    field,
		operator: "IN",
		value:    value,
		logic:    "AND",
	}
}

// Or creates a condition with OR logic.
func Or(c Condition) Condition {
	c.logic = "OR"
	return c
}
