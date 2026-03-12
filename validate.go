package orm

import "github.com/tinywasm/fmt"

func validate(action Action, m Model) error {
	if action != ActionCreateDatabase && m.TableName() == "" {
		return ErrEmptyTable
	}

	if action == ActionCreate || action == ActionUpdate {
		if len(m.Schema()) != len(m.Pointers()) {
			return fmt.Err(ErrValidation, "schema and pointers length mismatch")
		}
	}

	return nil
}
