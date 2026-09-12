package handler

import (
	"log/slog"
	"net/http"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/model"
	"menata.id/app/internal/ui"
)

// renderPageComponentChild (CR-20/CR-21, composable-runtime-roadmap.md
// 17l) builds an embedded component section from a component+dataset_id
// Children entry (model.ChildViewRef, real since 17k) -- the first real
// physical execution internal/composable has ever driven, not just a
// diagnostic (explainComposablePlan, composable_preview.go, only ever
// runs Explain()/MeasureComposition, never a real store call). Only
// "Metric" is wired to real execution today, named explicitly: filtered
// or grouped live execution remains open (CR-05). A non-Metric Component
// (or any resolution failure) degrades gracefully -- log + skip the
// section, the same posture page.go's own renderPageChild unknown-view
// case already takes, never a broken page. Split into its own file
// rather than page.go itself (Gate 2's own handler LOC ratchet).
func (h *Handler) renderPageComponentChild(r *http.Request, child model.ChildViewRef) *ui.PageSection {
	if child.Component != string(composable.ComponentMetric) {
		slog.Warn("page: declared children component is not yet wired to live execution", "component", child.Component, "dataset_id", child.DatasetID)
		return nil
	}
	declared, ok := h.interp.Get().GetDataset(child.DatasetID)
	if !ok {
		slog.Warn("page: declared children component names unknown dataset", "dataset_id", child.DatasetID)
		return nil
	}
	base, ok := h.interp.Get().GetMachine(declared.BaseMachineID)
	if !ok {
		slog.Warn("page: dataset names unknown base machine", "dataset_id", declared.ID, "base_machine_id", declared.BaseMachineID)
		return nil
	}
	// Permission-checked against the Dataset's own base Machine, mirroring
	// renderPageListChild's existing cross-machine CanRead guard -- a page
	// composing a Dataset from a different Machine must still respect
	// that Machine's own CAP-P05 read gate.
	_, appID := h.interp.Get().ScopeFor(base.ID)
	role := h.roleForApp(r, appID)
	if !h.guard.CanRead(base, role) {
		return nil
	}
	ds, err := composable.BuildDatasetFromDeclaredDataset(composable.MachineIndex{base.ID: base}, declared)
	if err != nil {
		slog.Warn("page: building dataset from declared config", "dataset_id", declared.ID, "error", err)
		return nil
	}
	node, err := composable.LowerMetric(ds)
	if err != nil {
		slog.Warn("page: children component metric rejected", "dataset_id", declared.ID, "error", err)
		return nil
	}
	// The real physical execution: an ordinary, ungrouped, unfiltered
	// count over base's own real records -- the same store method every
	// other page/list render already uses (h.records.List), no new
	// aggregate-pushdown-aware method added ("Infer Before Configure": a
	// plain count needs nothing more than this).
	records, err := h.records.List(r.Context(), base.ID, "", "")
	if err != nil {
		slog.Warn("page: listing records for metric", "machine_id", base.ID, "error", err)
		return nil
	}
	title := child.Title
	if title == "" {
		title = declared.Name
	}
	metric := composable.ResolveMetricValue(node, title, float64(len(records)))
	return &ui.PageSection{
		Title:   title,
		Layout:  child.Layout,
		Content: ui.MetricContent(metric),
	}
}
