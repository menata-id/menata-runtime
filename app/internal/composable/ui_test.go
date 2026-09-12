package composable_test

import (
	"reflect"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/model"
	"menata.id/app/internal/testing/builders"
)

// TestLowerPage_ViewRefCarriesOwnDataset is the regression test for the
// TODO Phases 1-2 left behind ("Phase 2 associates a Dataset per component
// node, not per page"): two list Views on the same Machine with DIFFERENT
// Columns must each get their OWN Dataset, not one page-level
// approximation.
func TestLowerPage_ViewRefCarriesOwnDataset(t *testing.T) {
	m := builders.Machine("mch_task").
		WithView(builders.View("vw_a", model.ViewTypeList).Columns("fld_a").Build()).
		WithView(builders.View("vw_b", model.ViewTypeList).Columns("fld_b").Build()).
		Build()

	page, err := composable.LowerPage(m, nil, nil)
	if err != nil {
		t.Fatalf("LowerPage: %v", err)
	}
	if len(page.Children) != 2 {
		t.Fatalf("len(Children) = %d, want 2", len(page.Children))
	}
	if want := []string{"fld_a"}; !reflect.DeepEqual(page.Children[0].Dataset.Projection.Fields, want) {
		t.Errorf("Children[0].Dataset.Projection.Fields = %v, want %v", page.Children[0].Dataset.Projection.Fields, want)
	}
	if want := []string{"fld_b"}; !reflect.DeepEqual(page.Children[1].Dataset.Projection.Fields, want) {
		t.Errorf("Children[1].Dataset.Projection.Fields = %v, want %v", page.Children[1].Dataset.Projection.Fields, want)
	}
}

func TestLowerPage_RelationProducesValidatedBinding(t *testing.T) {
	m := builders.Machine("mch_step").
		WithField(builders.Field("fld_document", model.FieldTypeReference).
			Options(model.FieldOptions{TargetMachine: "mch_document"}).Build()).
		WithView(builders.View("vw_list", model.ViewTypeList).Columns("fld_document").Build()).
		Build()

	page, err := composable.LowerPage(m, nil, nil)
	if err != nil {
		t.Fatalf("LowerPage: %v", err)
	}
	child := page.Children[0]
	if len(child.Bindings) != 1 {
		t.Fatalf("len(Bindings) = %d, want 1", len(child.Bindings))
	}
	b := child.Bindings[0]
	if b.Target.Domain != composable.ContextParentRecord || b.Target.Key != "fld_document" {
		t.Errorf("Binding.Target = %+v, want {parent_record fld_document}", b.Target)
	}

	// Verify independently, not just trust lowerViewRefChild's own internal
	// check -- ScopeForDataset(child.Dataset) must actually accept it.
	scope := composable.ScopeForDataset(*child.Dataset)
	if err := composable.ValidateBinding(scope, b); err != nil {
		t.Errorf("ValidateBinding: %v", err)
	}
}

func TestLowerChildren_SlotsAndStaticContent(t *testing.T) {
	summary := builders.Machine("mch_summary").
		WithView(builders.View("vw_summary", model.ViewTypeDashboard).Build()).
		Build()
	pending := builders.Machine("mch_pending").
		WithView(builders.View("vw_pending", model.ViewTypeList).Columns("fld_x").Build()).
		Build()

	viewIdx := map[string]*model.View{
		"vw_summary": summary.Views[0],
		"vw_pending": pending.Views[0],
	}
	machineIdx := composable.MachineIndex{
		"mch_summary": summary,
		"mch_pending": pending,
	}

	children := []model.ChildViewRef{
		{View: "vw_summary", Title: "Summary"},
		{View: "vw_pending", Title: "Pending", Layout: "main"},
		{Content: &model.PageContent{Type: "text", Text: "Recent Activity placeholder"}, Title: "Recent Activity", Layout: "aside"},
	}

	slots, err := composable.LowerChildren(children, viewIdx, machineIdx, nil)
	if err != nil {
		t.Fatalf("LowerChildren: %v", err)
	}

	if len(slots[""]) != 1 || slots[""][0].Properties["view_id"] != "vw_summary" {
		t.Errorf("Slots[\"\"] = %+v, want one node for vw_summary", slots[""])
	}
	if len(slots["main"]) != 1 || slots["main"][0].Properties["view_id"] != "vw_pending" {
		t.Errorf("Slots[\"main\"] = %+v, want one node for vw_pending", slots["main"])
	}
	if len(slots["aside"]) != 1 {
		t.Fatalf("Slots[\"aside\"] = %+v, want one node", slots["aside"])
	}
	aside := slots["aside"][0]
	// PageContent{Type:"text"} migrates through the Text component contract
	// (Phase 5) instead of a bare static_content node -- see ui.go's
	// lowerStaticContent.
	if aside.Kind != composable.UINodeComponent || aside.ComponentType != composable.ComponentText {
		t.Errorf("Slots[\"aside\"][0] = {Kind:%q ComponentType:%q}, want a Text component", aside.Kind, aside.ComponentType)
	}
	if aside.Properties["text"] != "Recent Activity placeholder" {
		t.Errorf("aside Properties[text] = %q, want %q", aside.Properties["text"], "Recent Activity placeholder")
	}
	if aside.Properties["title"] != "Recent Activity" {
		t.Errorf("aside Properties[title] = %q, want %q", aside.Properties["title"], "Recent Activity")
	}
}
