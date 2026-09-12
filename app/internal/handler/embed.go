package handler

import (
	"log/slog"
	"net/http"

	"menata.id/app/internal/model"
	"menata.id/app/internal/store"
	"menata.id/app/internal/ui"
)

// renderEmbeddedViews (CAP-V20 Tier 2) resolves a host View's own declared
// Config.Children (model.go's own doc comment on that field has the full
// reasoning) into ready-to-render sections. hostRec is whatever record the
// page actually being served is showing (e.g. one Approval Step) -- what
// each Child needs from it depends entirely on that Child's own Type,
// decided inside renderChildView below, never here.
//
// This is the one and only place a View gets composed from other Views in
// this codebase today -- CAP-V10 Tier 2 (a dedicated `page` View type
// composing several independently-sourced Views as its OWN entire body,
// capability-registry.md's own still-❌-not-admitted proposal) is a
// different, larger shape: a NEW View type whose whole job is composition.
// This is narrower and already real: ANY existing View type may
// additionally embed others inline, alongside its own primary content.
// Both would share this exact dispatch-by-Type mechanism if CAP-V10 Tier 2
// is ever admitted -- Children/ChildViewRef were named to make that
// convergence cheap, not to preempt that admission decision.
func (h *Handler) renderEmbeddedViews(r *http.Request, hostRec *store.Record, children []model.ChildViewRef) []ui.EmbeddedSection {
	var sections []ui.EmbeddedSection
	for _, ref := range children {
		if sec := h.renderChildView(r, hostRec, ref.View); sec != nil {
			sections = append(sections, *sec)
		}
	}
	return sections
}

// renderChildView (CAP-V20 Tier 2) is the dispatch this whole mechanism
// exists for: given a View id a host page declared as a Child, resolve
// that View and look at its own Type -- never at what the HOST expected or
// asked for by name. Everything downstream (does this Type need a parent
// record? a different Machine's own data? nothing but its own config?) is
// decided inside that Type's own case, in that Type's own file (today,
// decisionstepper.go's renderDecisionStepperChild) -- this function itself
// carries no capability-specific knowledge at all, on purpose: adding a
// second embeddable Type means adding one `case` here and one new
// render*Child function elsewhere, never touching this dispatch's own
// shape -- proven by CoordPlacement (2026-09-07): it was the SECOND Type
// added, and this function's own body only grew by one `case` line.
// model.EmbeddableChildViewTypes is the load-time half of the same
// contract (metadata/validate.go) -- keep both in sync, per that var's own
// doc comment.
// childEmbeds reports whether view already declares childViewID among its
// own Config.Children -- used to suppress a redundant top-of-page
// ui.DetailLink (record_crud.go's own "Set Position"/"View Progress" slots)
// pointing at the exact same content a Children entry already renders
// inline on this same page. view may be nil (no detail View declared at
// all); childViewID empty never matches a real Children entry.
func childEmbeds(view *model.View, childViewID string) bool {
	if view == nil {
		return false
	}
	for _, c := range view.Config.Children {
		if c.View == childViewID {
			return true
		}
	}
	return false
}

func (h *Handler) renderChildView(r *http.Request, hostRec *store.Record, childViewID string) *ui.EmbeddedSection {
	view, ok := h.interp.Get().GetView(childViewID)
	if !ok {
		// Load-time validation (metadata/validate.go) guarantees this can't
		// happen against the metadata that was actually loaded -- it CAN
		// still happen against a live database that has since drifted from
		// what's in memory (edited directly, or a reload skipped) -- the
		// exact class of problem this codebase has hit for real before
		// (capability-registry.md's CAP-V20 row, the mch_ca_lifted
		// incident) with zero log trace to explain the resulting blank
		// section. Warn, not silent, so that specific case leaves one.
		slog.Warn("children: declared view id does not resolve against loaded metadata",
			"host_record", hostRec.ID, "host_machine", hostRec.MachineID, "children_view", childViewID)
		return nil
	}
	switch view.Type {
	case model.ViewTypeDecisionStepper:
		return h.renderDecisionStepperChild(r, hostRec, view)
	case model.ViewTypeCoordPlacement:
		return h.renderCoordPlacementChild(r, hostRec, view)
	case model.ViewTypeActivityLog:
		return h.renderActivityLogChild(r, hostRec, view)
	default:
		// Same drift class as above: metadata/validate.go's own
		// EmbeddableChildViewTypes check should have already rejected this
		// at load time -- reaching here live means the loaded metadata and
		// this dispatch's own case list have gone out of sync somehow.
		slog.Warn("children: declared view is not an embeddable Type",
			"host_record", hostRec.ID, "host_machine", hostRec.MachineID, "children_view", childViewID, "type", view.Type)
		return nil
	}
}
