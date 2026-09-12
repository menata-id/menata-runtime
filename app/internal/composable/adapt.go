package composable

import (
	"fmt"

	"menata.id/app/internal/model"
)

// LowerApplication lowers every Machine in app into a page-kind UINode, in
// declaration order. Deterministic: the same *model.Application always
// produces an identical (reflect.DeepEqual) result -- see adapt_test.go.
func LowerApplication(app *model.Application) ([]UINode, error) {
	viewIdx := IndexViews(app)
	machineIdx := IndexMachines(app)
	pages := make([]UINode, 0, len(app.Machines))
	for _, m := range app.Machines {
		page, err := LowerPage(m, viewIdx, machineIdx)
		if err != nil {
			return nil, fmt.Errorf("composable: lower page for machine %s: %w", m.ID, err)
		}
		pages = append(pages, page)
	}
	return pages, nil
}
