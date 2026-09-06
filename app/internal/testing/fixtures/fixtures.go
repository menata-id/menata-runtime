// Package fixtures provides canned model.* sample data built from
// internal/testing/builders, for tests that need a valid starting point
// without constructing one field-by-field.
package fixtures

import (
	"menata.id/app/internal/model"
	"menata.id/app/internal/testing/builders"
)

// MinimalMachine returns the smallest valid Machine a test can start from:
// one required text field, no events/constraints/views.
func MinimalMachine() *model.Machine {
	return builders.Machine("mch_test").
		WithField(builders.Field("fld_name", model.FieldTypeText).Required().Build()).
		Build()
}
