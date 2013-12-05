package driver

import (
	"gnd.la/util/structs"
)

type Reference struct {
	Model Model
	Field string
}

type Fields struct {
	*structs.Struct
	// Quoted mangled names of the fields, including the table
	// name (e.g. "table"."field").
	QuotedNames []string
	// Fields which should be omitted when they are empty
	OmitEmpty []bool
	// Fields which should become null when they are empty
	NullEmpty []bool
	// The index of the primary (-1 if there's no pk)
	PrimaryKey int
	// True if the primary key is an integer type with auto_increment
	IntegerAutoincrementPk bool
	// The fields which make the composite primary key, if any
	CompositePrimaryKey []int
	// Model methods called by the ORM
	Methods *Methods
	// Other models referenced by this model. The key
	// is the field name in this model.
	References map[string]*Reference
}

func (f *Fields) IsSubfield(field, parent []int) bool {
	if len(field) <= len(parent) {
		return false
	}
	for ii, v := range parent {
		if field[ii] != v {
			return false
		}
	}
	return true
}
