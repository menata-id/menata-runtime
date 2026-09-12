package handler

import (
	"context"
	"fmt"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/model"
	"menata.id/app/internal/store"
	"menata.id/app/internal/ui"
)

// buildDetailFieldsViaComposable (composable-runtime-roadmap.md 17o,
// closing CR-29's own deferred item) sources the Detail page's own field
// SET/ORDER from composable.BuildDatasetFromView's ViewTypeDetail case
// (17m) instead of iterating machine.Fields directly -- the one part of
// Detail's own real behavior internal/composable can legitimately drive,
// since it's a pure Data-plane concept (which fields, in what order).
//
// Every per-field VALUE formatting rule below is moved VERBATIM from
// Detail's own prior inline loop, not reimplemented -- reference/user/
// group label dereferencing needs a real second store lookup
// (referenceLabel/userLabel/groupLabel), money/computed/SLA-urgency need
// real business logic (formatMoney/computedValue/slaUrgency). None of
// that can move INTO internal/composable itself: that package is zero-
// I/O by construction (Gate 5), and ResolveFieldValue's own doc comment
// already names this exact boundary ("full label resolution... needs a
// SECOND store lookup this package doesn't perform"). This function
// stays in internal/handler for exactly that reason -- not scope
// avoidance, a real architectural constraint.
func (h *Handler) buildDetailFieldsViaComposable(ctx context.Context, machine *model.Machine, detailView *model.View, hidden map[string]bool, wsSlug string, rec *store.Record) ([]ui.DetailField, error) {
	ds, err := composable.BuildDatasetFromView(machine, detailView)
	if err != nil {
		return nil, err
	}
	fieldByID := fieldIndex(machine)
	fields := make([]ui.DetailField, 0, len(ds.Projection.Fields))
	for _, fieldID := range ds.Projection.Fields {
		if hidden[fieldID] {
			continue
		}
		f, ok := fieldByID[fieldID]
		if !ok {
			continue
		}
		fields = append(fields, h.detailFieldValue(ctx, f, detailView, wsSlug, rec))
	}
	return fields, nil
}

// detailFieldValue formats one Field's own value exactly as Detail's
// prior inline switch did -- split into resolveDetailFieldValue (the
// per-type dispatch) plus this SLA-urgency wrapper, purely for Gate 3
// (keeps each function's own complexity small), not a behavior change.
func (h *Handler) detailFieldValue(ctx context.Context, f *model.Field, detailView *model.View, wsSlug string, rec *store.Record) ui.DetailField {
	val, link := h.resolveDetailFieldValue(ctx, f, wsSlug, rec)
	urgency := ""
	if detailView != nil && f.ID == detailView.Config.SlaField {
		if label, u, ok := slaUrgency(val, detailView.Config.SlaWarningDays); ok { // CAP-V17
			val, urgency = label, u
		}
	}
	return ui.DetailField{Name: f.Name, Value: val, Link: link, SlaUrgency: urgency}
}

// resolveDetailFieldValue is the per-Field-Type dispatch, moved verbatim
// from Detail's own prior inline switch (only the guard shape changed,
// from per-case "&& val != \"\"" to one early return covering every type
// that needs a non-empty value -- Boolean/Computed still run regardless,
// exactly as before).
func (h *Handler) resolveDetailFieldValue(ctx context.Context, f *model.Field, wsSlug string, rec *store.Record) (val, link string) {
	if v, ok := rec.Data[f.ID]; ok {
		val = fmt.Sprintf("%v", v)
	}
	if val == "" && f.Type != model.FieldTypeBoolean && f.Type != model.FieldTypeComputed {
		return val, link
	}
	switch f.Type {
	case model.FieldTypeReference:
		return h.dereferenceDetailLink(ctx, f, wsSlug, val)
	case model.FieldTypeUser:
		if label, err := h.userLabel(ctx, val); err == nil && label != "" {
			val = label
		}
	case model.FieldTypeGroup:
		if label, err := h.groupLabel(ctx, val); err == nil && label != "" {
			val = label
		}
	case model.FieldTypeBoolean:
		val = boolLabel(val) // CAP-F09
	case model.FieldTypeMoney:
		val = formatMoney(val, f, rec.Data) // CAP-F08
	case model.FieldTypeFile:
		link = "/files/" + val // CAP-F06
	case model.FieldTypeComputed:
		val = computedValue(f, rec.Data) // CAP-F14
	}
	return val, link
}

// dereferenceDetailLink resolves a reference Field's own real display
// label + link, moved verbatim from Detail's own prior inline switch.
func (h *Handler) dereferenceDetailLink(ctx context.Context, f *model.Field, wsSlug, refID string) (val, link string) {
	val = refID
	target := f.Options.TargetMachine
	if label, err := h.referenceLabel(ctx, target, refID); err == nil && label != "" {
		val = label
		link = "/" + wsSlug + "/" + target + "/" + refID
	}
	return val, link
}
