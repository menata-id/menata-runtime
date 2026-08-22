package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"menata.id/runtime/internal/model"
)

// referenceViolations enforces CAP-F13 referential integrity: a `reference`
// field's value must resolve to a real record on the target Machine. This is
// intrinsic to the field type, the same way `required` is intrinsic to a
// Field's Required flag — not a declared Constraint row.
func (h *Handler) referenceViolations(ctx context.Context, machine *model.Machine, data map[string]any) ([]string, error) {
	var out []string
	for _, f := range machine.Fields {
		if f.Type != model.FieldTypeReference {
			continue
		}
		v, ok := data[f.ID]
		if !ok {
			continue
		}
		recordID, _ := v.(string)
		if recordID == "" {
			continue
		}
		exists, err := h.records.Exists(ctx, f.Options.TargetMachine, recordID)
		if err != nil {
			return nil, err
		}
		if !exists {
			target, _ := h.interp.Get().GetMachine(f.Options.TargetMachine)
			targetName := f.Options.TargetMachine
			if target != nil {
				targetName = target.Name
			}
			out = append(out, fmt.Sprintf("%s does not reference an existing %s record.", f.Name, targetName))
		}
	}
	return out, nil
}

// userReferenceViolations (CAP-F05) enforces referential integrity for
// `user`-typed fields, the exact same tier and shape as referenceViolations
// (CAP-F13) -- a required-field-style violation, not a 500, when a value
// doesn't resolve to a real account.
func (h *Handler) userReferenceViolations(ctx context.Context, machine *model.Machine, data map[string]any) ([]string, error) {
	var out []string
	for _, f := range machine.Fields {
		if f.Type != model.FieldTypeUser {
			continue
		}
		v, ok := data[f.ID]
		if !ok {
			continue
		}
		userID, _ := v.(string)
		if userID == "" {
			continue
		}
		exists, err := h.users.Exists(ctx, userID)
		if err != nil {
			return nil, err
		}
		if !exists {
			out = append(out, fmt.Sprintf("%s does not reference an existing user.", f.Name))
		}
	}
	return out, nil
}

// immutabilityViolation (CAP-R07) returns a non-empty rejection reason when
// machine.Config declares `immutable_field`/`immutable_values` (a
// comma-separated list, CAP-X03's generic Machine-level settings, not a new
// migration column) and data's current value for that field is one of
// them -- "record is frozen once Posted," stronger than CAP-E06 (which only
// guards Events): this guards direct field edits (Update) and archival
// (Archive) too, every mutation path, not just the workflow transitions
// CAP-E06 already covers.
func (h *Handler) immutabilityViolation(machine *model.Machine, data map[string]any) string {
	field := machine.Config["immutable_field"]
	if field == "" {
		return ""
	}
	cur := fmt.Sprintf("%v", data[field])
	for _, v := range strings.Split(machine.Config["immutable_values"], ",") {
		if strings.TrimSpace(v) == cur {
			return fmt.Sprintf("record is immutable while %s is %q", field, cur)
		}
	}
	return ""
}

// inScratchState (CAP-R08) reports whether data's current value for
// machine.Config's declared `scratch_field` is one of `scratch_values`
// (comma-separated) -- a record in this state (e.g. a Cart before
// Checkout) has none of its eventual business-rule Constraints enforced
// yet, the opposite end of CAP-R07's spectrum. Referential integrity
// (referenceViolations/userReferenceViolations/uniquenessViolations) is
// NOT exempted -- a scratch record can be incomplete, not corrupt. The
// commit point back into full enforcement needs no new mechanism: CAP-C09's
// existing trigger-time Violations re-check already applies the moment an
// event moves the record out of scratch_values.
func (h *Handler) inScratchState(machine *model.Machine, data map[string]any) bool {
	field := machine.Config["scratch_field"]
	if field == "" {
		return false
	}
	cur := fmt.Sprintf("%v", data[field])
	for _, v := range strings.Split(machine.Config["scratch_values"], ",") {
		if strings.TrimSpace(v) == cur {
			return true
		}
	}
	return false
}

// withChangePolicyCreatedAt (CAP-W07) exposes a record's creation time to
// constraint.Eval as an ordinary comparable field (model.ChangePolicyCreatedAtField)
// without ever persisting it. Always returns a COPY -- data is the exact map
// that gets written to the record's JSONB column right after the Violations
// check runs, so mutating it in place would leak the synthetic key into
// storage. Skips the copy entirely for the vast majority of Machines that
// declare no `new_records` change_policy (machine.NeedsCreatedAtGuard).
func withChangePolicyCreatedAt(machine *model.Machine, data map[string]any, t time.Time) map[string]any {
	if !machine.NeedsCreatedAtGuard {
		return data
	}
	out := make(map[string]any, len(data)+1)
	for k, v := range data {
		out[k] = v
	}
	out[model.ChangePolicyCreatedAtField] = t.Format("2006-01-02")
	return out
}

// uniquenessViolations (CAP-C12) enforces `unique` constraints -- single or
// composite/multi-field -- against every OTHER record on the same machine.
// Unlike engine.Violations, this needs the database (RecordStore), so it's a
// separate check at the same tier as referenceViolations/
// userReferenceViolations, not folded into constraint.Engine (which
// deliberately never touches storage). excludeRecordID is the record being
// updated -- empty on Create, where nothing to exclude exists yet.
func (h *Handler) uniquenessViolations(ctx context.Context, machine *model.Machine, data map[string]any, excludeRecordID string) ([]string, error) {
	var out []string
	for _, c := range machine.Constraints {
		if c.Expression.Operator != "unique" {
			continue
		}
		fields := c.Expression.Fields
		if len(fields) == 0 && c.Expression.Field != "" {
			fields = []string{c.Expression.Field}
		}
		if len(fields) == 0 {
			continue
		}
		fieldValues := make(map[string]string, len(fields))
		allSet := true
		for _, fid := range fields {
			v, ok := data[fid]
			if !ok {
				allSet = false
				break
			}
			fieldValues[fid] = fmt.Sprintf("%v", v)
		}
		if !allSet {
			// Nothing to collide with yet -- CAP-C01 `required` already
			// covers "this field must have a value" separately.
			continue
		}
		exists, err := h.records.ExistsWithFieldValues(ctx, machine.ID, fieldValues, excludeRecordID)
		if err != nil {
			return nil, err
		}
		if exists {
			out = append(out, c.Rule)
		}
	}
	return out, nil
}

// --- CAP-A07 / CAP-A08 workflow orchestration --------------------------------
//
// Both capabilities need the same handful of lookups on the child Machine
// (Approval Step, in Case 3's proof): which Field references the parent
// record, which Field holds the step's own Sequence, which Field holds its
// Decision. None of these are named in the action's own params — Menata
// Language doesn't have a way for a business author to name "the field that
// scopes this" beyond writing an ordinary `reference` Field — so, like
// displayLabel, this resolves them by name/type heuristic (a `reference`
// Field pointing at the parent Machine; a Field literally named "Sequence" or
// "Decision", case-insensitive). Prototype-honest, not a final design.
