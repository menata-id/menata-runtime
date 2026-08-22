package handler

import (
	"context"

	"menata.id/runtime/internal/model"
	"menata.id/runtime/internal/store"
)

// stampRequirementCounters (CAP-W01, Process Overlay B3) is the write-time
// half of "write-time fan-in, read-time O(1)" (Study 20 §6.3): called right
// after a record of `machine` is successfully created, it scans machine's
// own `reference` Fields, resolves each target record's own Machine, and --
// only when that Machine's Process declares a Requirement naming `machine`
// as its Target (model.ProcessRequirement, compiled by
// metadata.compileProcess into a generated counter Field + gating
// Constraint) -- atomically increments that counter on the referenced
// parent record. The transition-time cardinality check later reads the
// counter directly (constraint.Eval against already-fetched data); it never
// re-derives it with a query.
//
// Scope, named not silently assumed complete: wired only into the plain
// HTTP Create path this pass (the dominant case -- a user attaches evidence
// via an ordinary form). CAP-A06 (create_record), CSV import, and the JSON
// API's create route do not call this yet -- a deliberate, later-pass gap,
// the same class of scope boundary CAP-F16/CAP-A06's own notes already
// name elsewhere in this codebase.
func (h *Handler) stampRequirementCounters(ctx context.Context, machine *model.Machine, rec *store.Record) error {
	for _, f := range machine.Fields {
		if f.Type != model.FieldTypeReference {
			continue
		}
		parentID, _ := rec.Data[f.ID].(string)
		if parentID == "" {
			continue
		}
		parentMachine, ok := h.interp.Get().GetMachine(f.Options.TargetMachine)
		if !ok || parentMachine.Process == nil {
			continue
		}
		// Dedup by counter id -- the compiler already collapses multiple
		// transitions naming the same (type, target) into ONE counter
		// Field (compile.go's compileRequirements); incrementing once per
		// matching transition here would over-count the same requirement.
		counters := map[string]bool{}
		for _, t := range parentMachine.Process.Transitions {
			for _, r := range t.Requirements {
				if r.Target == machine.ID {
					counters[model.RequirementCounterFieldID(parentMachine.ID, r.Target)] = true
				}
			}
		}
		for counterID := range counters {
			if err := h.records.IncrementField(ctx, parentID, counterID); err != nil {
				return err
			}
		}
	}
	return nil
}
