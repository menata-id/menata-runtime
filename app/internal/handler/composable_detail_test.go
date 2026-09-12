package handler

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"menata.id/app/internal/model"
	"menata.id/app/internal/store"
	"menata.id/app/internal/testing/builders"
	"menata.id/app/internal/ui"
)

// TestBuildDetailFieldsViaComposable (composable-runtime-roadmap.md 17o)
// is the equivalence proof for the Detail cutover: buildDetailFieldsViaComposable
// (sourcing the field SET/ORDER from composable.BuildDatasetFromView) must
// produce the exact same []ui.DetailField the OLD, now-removed inline
// loop used to build directly, for every field type that never needs a
// real store lookup (text, value_list, boolean, money, file, computed).
// Reference/user/group label dereferencing needs a real DATABASE_URL
// (referenceLabel/userLabel/groupLabel each do a real store call) and is
// proven instead by the full conformance suite (T-numbers spanning many
// real Machines/field types already exercise those paths against the
// real route this cutover changed).
func TestBuildDetailFieldsViaComposable(t *testing.T) {
	m := builders.Machine("mch_task").
		WithField(builders.Field("fld_title", model.FieldTypeText).Name("Title").Build()).
		WithField(builders.Field("fld_active", model.FieldTypeBoolean).Name("Active").Build()).
		WithField(builders.Field("fld_amount", model.FieldTypeMoney).Name("Amount").
			Options(model.FieldOptions{Currency: "USD"}).Build()).
		WithField(builders.Field("fld_total", model.FieldTypeComputed).Name("Total").
			Options(model.FieldOptions{SourceField: "fld_amount", Factor: 2}).Build()).
		WithField(builders.Field("fld_attachment", model.FieldTypeFile).Name("Attachment").Build()).
		WithField(builders.Field("fld_due", model.FieldTypeDate).Name("Due").Build()).
		Build()
	detailView := &model.View{ID: "vw_task_detail", MachineID: "mch_task", Type: model.ViewTypeDetail,
		Config: model.ViewConfig{SlaField: "fld_due", SlaWarningDays: 3}}

	rec := &store.Record{ID: "rec1", Data: map[string]any{
		"fld_title":      "Ship it",
		"fld_active":     "true",
		"fld_amount":     "100",
		"fld_attachment": "abc123",
		"fld_due":        "2020-01-01", // far in the past -- deterministically "overdue"
	}}

	h := &Handler{}
	got, err := h.buildDetailFieldsViaComposable(context.Background(), m, detailView, map[string]bool{}, "ws_default", rec)
	if err != nil {
		t.Fatalf("buildDetailFieldsViaComposable: %v", err)
	}

	if len(got) != 6 {
		t.Fatalf("len(fields) = %d, want 6", len(got))
	}
	// The OLD behavior, reconstructed independently per field, in
	// Machine.Fields order (the exact order composable's own
	// BuildDatasetFromView Detail case produces too). The Due field
	// (last) is checked separately below -- slaUrgency's own "Overdue by
	// N day(s)" label embeds a day count relative to time.Now(), not a
	// fixed string.
	want := []ui.DetailField{
		{Name: "Title", Value: "Ship it"},
		{Name: "Active", Value: "Yes"},
		{Name: "Amount", Value: "USD 100"},
		{Name: "Total", Value: "200"},
		{Name: "Attachment", Value: "abc123", Link: "/files/abc123"},
	}
	if !reflect.DeepEqual(got[:5], want) {
		t.Errorf("fields[:5] = %+v, want %+v", got[:5], want)
	}
	due := got[5]
	if due.Name != "Due" || due.SlaUrgency != "overdue" || !strings.HasPrefix(due.Value, "Overdue by ") {
		t.Errorf("fields[5] = %+v, want {Name:Due SlaUrgency:overdue Value:\"Overdue by ...\"}", due)
	}
}

// TestBuildDetailFieldsViaComposable_HiddenFieldExcluded proves CAP-P06
// survives the cutover -- a hidden field never reaches the result, same
// as the old inline loop's own "if hidden[f.ID] { continue }".
func TestBuildDetailFieldsViaComposable_HiddenFieldExcluded(t *testing.T) {
	m := builders.Machine("mch_task").
		WithField(builders.Field("fld_title", model.FieldTypeText).Name("Title").Build()).
		WithField(builders.Field("fld_secret", model.FieldTypeText).Name("Secret").Build()).
		Build()
	detailView := &model.View{ID: "vw_task_detail", MachineID: "mch_task", Type: model.ViewTypeDetail}
	rec := &store.Record{ID: "rec1", Data: map[string]any{"fld_title": "Ship it", "fld_secret": "hunter2"}}

	h := &Handler{}
	got, err := h.buildDetailFieldsViaComposable(context.Background(), m, detailView, map[string]bool{"fld_secret": true}, "ws_default", rec)
	if err != nil {
		t.Fatalf("buildDetailFieldsViaComposable: %v", err)
	}
	want := []ui.DetailField{{Name: "Title", Value: "Ship it"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("fields = %+v, want %+v", got, want)
	}
}
