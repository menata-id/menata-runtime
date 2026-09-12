package handler

import (
	"fmt"
	"strconv"
	"strings"

	"menata.id/app/internal/model"
	"menata.id/app/internal/store"
)

// sortFieldFor (CAP-V14/DefaultSort) resolves which sort field/direction a
// list View's own live (non-archived) records should be fetched in --
// CAP-V14's ManualOrder wins over DefaultSort when both are declared on
// the same View, since manual order only means anything if it's what's
// actually shown. Extracted from record_crud.go's own List (composable-
// runtime-roadmap.md 17f) into this file rather than growing that one
// further (already over Gate 2's own LOC budget) -- so ComposablePreview
// can reuse the identical decision instead of reimplementing it --
// previously ComposablePreview passed "","" unconditionally, correct for
// vw_ad_all's own DefaultSort only by coincidence.
func sortFieldFor(view *model.View) (field, direction string) {
	if view == nil {
		return "", ""
	}
	if view.Config.ManualOrder {
		return store.SortOrderField, ""
	}
	if view.Config.DefaultSort != nil {
		return view.Config.DefaultSort.Field, view.Config.DefaultSort.Direction
	}
	return "", ""
}

// searchListRecords (CAP-V08) filters records to those matching query
// (substring, case-insensitive) across colIDs -- a plain GET query
// param, no JS, matching this prototype's no-SPA-framework posture.
// Extracted from record_crud.go's own List (composable-runtime-roadmap.md
// 17f) so ComposablePreview can reuse the identical behavior instead of
// duplicating it -- exactly the kind of duplicated business logic
// composable-apps-trial.md's own anti-pattern section (§14.1) warns
// against.
func searchListRecords(query string, colIDs []string, records []*store.Record) []*store.Record {
	if query == "" {
		return records
	}
	q := strings.ToLower(query)
	kept := records[:0]
	for _, rec := range records {
		match := false
		for _, id := range colIDs {
			if v, ok := rec.Data[id]; ok && strings.Contains(strings.ToLower(fmt.Sprintf("%v", v)), q) {
				match = true
				break
			}
		}
		if match {
			kept = append(kept, rec)
		}
	}
	return kept
}

// paginateListRecords (CAP-R05) slices records into the requested page --
// applied AFTER filter/search, on the final matching set, not as a SQL
// LIMIT/OFFSET before them, since otherwise a filter could discard most
// of one SQL page and never see matching rows sitting on the next one.
// pageParam is the raw ?page= query value; an invalid/missing value
// clamps to page 1, a value beyond the real last page clamps to it
// (never an out-of-range slice or an empty response by surprise).
// Extracted from record_crud.go's own List (composable-runtime-
// roadmap.md 17f) for the same reuse reason as searchListRecords.
func paginateListRecords(records []*store.Record, pageParam string) (paged []*store.Record, page, totalPages int) {
	const pageSize = 25
	pageNum, _ := strconv.Atoi(pageParam)
	if pageNum < 1 {
		pageNum = 1
	}
	totalRecords := len(records)
	totalPages = (totalRecords + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if pageNum > totalPages {
		pageNum = totalPages
	}
	start := (pageNum - 1) * pageSize
	end := start + pageSize
	if start > totalRecords {
		start = totalRecords
	}
	if end > totalRecords {
		end = totalRecords
	}
	return records[start:end], pageNum, totalPages
}
