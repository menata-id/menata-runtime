package handler

import (
	"reflect"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/model"
	"menata.id/app/internal/store"
	"menata.id/app/internal/testing/builders"
	"menata.id/app/internal/ui"
)

// TestBuildComposableBoardLanes (composable-runtime-roadmap.md 17n) is
// the equivalence proof for the Board cutover: buildComposableBoardLanes
// (the new, composable-driven path) must produce the exact same
// []ui.BoardLane shape the OLD, now-removed views.go Reference-branch
// logic used to build directly -- same lane order, same real display
// names (via displayLabel, unchanged), same per-card cell values, in
// colIDs order.
func TestBuildComposableBoardLanes(t *testing.T) {
	cardMachine := builders.Machine("mch_card").
		WithField(builders.Field("fld_card_title", model.FieldTypeText).Name("Title").Build()).
		Build()
	listMachine := builders.Machine("mch_list").
		WithField(builders.Field("fld_list_name", model.FieldTypeText).Name("Name").Build()).
		Build()

	list1 := &store.Record{ID: "list1", Data: map[string]any{"fld_list_name": "To Do"}}
	list2 := &store.Record{ID: "list2", Data: map[string]any{"fld_list_name": "Doing"}}
	card1 := &store.Record{ID: "card1", Data: map[string]any{"fld_card_title": "Card 1"}}
	card2 := &store.Record{ID: "card2", Data: map[string]any{"fld_card_title": "Card 2"}}
	card3 := &store.Record{ID: "card3", Data: map[string]any{"fld_card_title": "Card 3"}}

	composableLanes := []composable.BoardLane{
		{LaneRecordID: "list1", Cards: []composable.CollectionItem{{RecordID: "card1"}, {RecordID: "card2"}}},
		{LaneRecordID: "list2", Cards: []composable.CollectionItem{{RecordID: "card3"}}},
	}

	lanes, err := buildComposableBoardLanes(cardMachine, composableLanes,
		[]*store.Record{card1, card2, card3}, []*store.Record{list1, list2},
		listMachine, []string{"fld_card_title"}, nil)
	if err != nil {
		t.Fatalf("buildComposableBoardLanes: %v", err)
	}

	want := []ui.BoardLane{
		{ID: "list1", Name: "To Do", Rows: []ui.ListRow{
			{ID: "card1", Cells: []ui.ListCell{{Value: "Card 1"}}},
			{ID: "card2", Cells: []ui.ListCell{{Value: "Card 2"}}},
		}},
		{ID: "list2", Name: "Doing", Rows: []ui.ListRow{
			{ID: "card3", Cells: []ui.ListCell{{Value: "Card 3"}}},
		}},
	}
	if !reflect.DeepEqual(lanes, want) {
		t.Errorf("lanes = %+v, want %+v", lanes, want)
	}
}

// TestBuildComposableBoardLanes_EmptyLane proves CAP-V14's own "an unused
// lane still renders empty" rule survives the cutover -- a lane record
// with zero matching cards still produces a BoardLane with an empty
// (non-nil-required) Rows slice, not a missing lane.
func TestBuildComposableBoardLanes_EmptyLane(t *testing.T) {
	cardMachine := builders.Machine("mch_card").
		WithField(builders.Field("fld_card_title", model.FieldTypeText).Name("Title").Build()).
		Build()
	listMachine := builders.Machine("mch_list").
		WithField(builders.Field("fld_list_name", model.FieldTypeText).Name("Name").Build()).
		Build()
	backlog := &store.Record{ID: "backlog", Data: map[string]any{"fld_list_name": "Backlog"}}

	composableLanes := []composable.BoardLane{{LaneRecordID: "backlog", Cards: nil}}
	lanes, err := buildComposableBoardLanes(cardMachine, composableLanes, nil, []*store.Record{backlog}, listMachine, []string{"fld_card_title"}, nil)
	if err != nil {
		t.Fatalf("buildComposableBoardLanes: %v", err)
	}
	if len(lanes) != 1 || lanes[0].ID != "backlog" || lanes[0].Name != "Backlog" || len(lanes[0].Rows) != 0 {
		t.Errorf("lanes = %+v, want one empty Backlog lane", lanes)
	}
}

// TestBuildComposableBoardLanes_WithCardMeta (17q) proves CardMeta lines
// up with Rows by index, per-card, not per-lane -- card2's own metadata
// must never leak onto card1's row just because they share a lane.
func TestBuildComposableBoardLanes_WithCardMeta(t *testing.T) {
	cardMachine := builders.Machine("mch_card").
		WithField(builders.Field("fld_card_title", model.FieldTypeText).Name("Title").Build()).
		Build()
	listMachine := builders.Machine("mch_list").
		WithField(builders.Field("fld_list_name", model.FieldTypeText).Name("Name").Build()).
		Build()
	list1 := &store.Record{ID: "list1", Data: map[string]any{"fld_list_name": "To Do"}}
	card1 := &store.Record{ID: "card1", Data: map[string]any{"fld_card_title": "Card 1"}}
	card2 := &store.Record{ID: "card2", Data: map[string]any{"fld_card_title": "Card 2"}}

	composableLanes := []composable.BoardLane{
		{LaneRecordID: "list1", Cards: []composable.CollectionItem{{RecordID: "card1"}, {RecordID: "card2"}}},
	}
	cardMeta := map[string]ui.BoardCardMeta{
		"card1": {Progress: "1/2"},
		"card2": {DueDate: "Sep 14"},
	}
	lanes, err := buildComposableBoardLanes(cardMachine, composableLanes,
		[]*store.Record{card1, card2}, []*store.Record{list1}, listMachine, []string{"fld_card_title"}, cardMeta)
	if err != nil {
		t.Fatalf("buildComposableBoardLanes: %v", err)
	}
	if len(lanes) != 1 || len(lanes[0].CardMeta) != 2 {
		t.Fatalf("lanes = %+v, want one lane with 2 CardMeta entries", lanes)
	}
	if lanes[0].CardMeta[0].Progress != "1/2" || lanes[0].CardMeta[0].DueDate != "" {
		t.Errorf("card1 meta = %+v, want Progress=1/2, DueDate empty", lanes[0].CardMeta[0])
	}
	if lanes[0].CardMeta[1].DueDate != "Sep 14" || lanes[0].CardMeta[1].Progress != "" {
		t.Errorf("card2 meta = %+v, want DueDate=Sep 14, Progress empty", lanes[0].CardMeta[1])
	}
}
