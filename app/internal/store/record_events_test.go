package store_test

import (
	"context"
	"testing"
	"time"

	"menata.id/app/internal/store"
	"menata.id/app/internal/testing/testdb"
)

// activityLogFixture (CAP-R04 "R28", composable-runtime-roadmap.md 17p)
// creates two real Documents and three real events across them --
// shared setup for the three TestListEventsFor* tests below. Each gets
// its own top-level function (Gate 3: gocyclo attributes a t.Run
// closure's complexity to its enclosing function, so separate functions,
// not subtests, are what actually keeps each one's own complexity
// small).
//
// No shared transaction, deliberately: migrations/manual/009's own RLS
// cutover is never applied by plain `make migrate-up` (this test's own
// DATABASE_URL only ever ran that), so no app.workspace_id session
// variable is needed for correctness here. Each call below goes straight
// to the pool, its own real transaction -- giving each LogEvent call its
// own real, distinct performed_at (Postgres's NOW() is transaction-time:
// sharing one transaction across multiple LogEvent calls, as an earlier
// version of this test did, made every row's own performed_at IDENTICAL,
// caught live when the very next assertion on stable ordering failed).
// This throwaway isolated schema is dropped by the caller after the
// whole suite runs, same as every other real-write test in this codebase
// -- no rollback needed.
func activityLogFixture(t *testing.T) (rs *store.RecordStore, doc1ID, doc2ID string) {
	t.Helper()
	pool := testdb.Connect(t)
	ctx := context.Background()
	rs = store.NewRecordStore(pool)

	doc1, err := rs.Create(ctx, "mch_approval_document", "ws_default", map[string]any{"fld_ad_title": "Doc 1"})
	if err != nil {
		t.Fatalf("create doc1: %v", err)
	}
	doc2, err := rs.Create(ctx, "mch_approval_document", "ws_default", map[string]any{"fld_ad_title": "Doc 2"})
	if err != nil {
		t.Fatalf("create doc2: %v", err)
	}

	// Two real events on doc1, one on doc2 -- LogEvent is the only
	// existing write path, called directly here the same way
	// executor.go's own Persist already does.
	if err := rs.LogEvent(ctx, doc1.ID, "evt_ad_submit", "Alice", "corr-1", "ws_default", map[string]any{"fld_ad_title": "Doc 1"}); err != nil {
		t.Fatalf("log event 1: %v", err)
	}
	time.Sleep(20 * time.Millisecond) // each call is its own transaction now -- this guarantees a real, distinct performed_at
	if err := rs.LogEvent(ctx, doc1.ID, "evt_ad_approve", "Bob", "corr-2", "ws_default", map[string]any{"fld_ad_title": "Doc 1"}); err != nil {
		t.Fatalf("log event 2: %v", err)
	}
	if err := rs.LogEvent(ctx, doc2.ID, "evt_ad_submit", "Alice", "corr-3", "ws_default", map[string]any{"fld_ad_title": "Doc 2"}); err != nil {
		t.Fatalf("log event 3: %v", err)
	}
	return rs, doc1.ID, doc2.ID
}

// TestListEventsForRecord proves the record-scoped read side, most-
// recent-first.
func TestListEventsForRecord(t *testing.T) {
	rs, doc1ID, _ := activityLogFixture(t)
	events, err := rs.ListEventsForRecord(context.Background(), doc1ID)
	if err != nil {
		t.Fatalf("ListEventsForRecord: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	// Most-recent-first: the Approve event (logged second) comes first.
	if events[0].EventID != "evt_ad_approve" || events[0].PerformedBy != "Bob" {
		t.Errorf("events[0] = %+v, want evt_ad_approve by Bob", events[0])
	}
	if events[1].EventID != "evt_ad_submit" || events[1].PerformedBy != "Alice" {
		t.Errorf("events[1] = %+v, want evt_ad_submit by Alice", events[1])
	}
}

// TestListEventsForMachine proves the cross-record read side -- doc2's
// own event appears even though it's a different record than doc1's.
func TestListEventsForMachine(t *testing.T) {
	rs, _, doc2ID := activityLogFixture(t)
	events, err := rs.ListEventsForMachine(context.Background(), "mch_approval_document", 10)
	if err != nil {
		t.Fatalf("ListEventsForMachine: %v", err)
	}
	if len(events) < 3 {
		t.Fatalf("len(events) = %d, want at least 3 (this test's own three)", len(events))
	}
	var sawDoc2 bool
	for _, e := range events {
		if e.RecordID == doc2ID {
			sawDoc2 = true
		}
	}
	if !sawDoc2 {
		t.Error("ListEventsForMachine did not include doc2's own event -- not genuinely cross-record")
	}
}

// TestListEventsForMachine_ScopedToOneMachine proves a Machine-scoped
// query never returns another Machine's rows -- mch_approval_step never
// had any event logged in this fixture.
func TestListEventsForMachine_ScopedToOneMachine(t *testing.T) {
	rs, doc1ID, doc2ID := activityLogFixture(t)
	events, err := rs.ListEventsForMachine(context.Background(), "mch_approval_step", 10)
	if err != nil {
		t.Fatalf("ListEventsForMachine: %v", err)
	}
	for _, e := range events {
		if e.RecordID == doc1ID || e.RecordID == doc2ID {
			t.Errorf("ListEventsForMachine(mch_approval_step) leaked a mch_approval_document event: %+v", e)
		}
	}
}
