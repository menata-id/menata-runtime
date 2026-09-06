// Package testing collects the shared test infrastructure a real coverage
// push needs before it means anything: builders (fluent model.* test
// constructors), fixtures (canned sample data built from those), httptest
// (wires a full *handler.Handler against a migrated+seeded test pool,
// mirroring cmd/server/main.go's own construction), mocks, and testdb
// (skip-if-no-DATABASE_URL + connect + cleanup, extracted from
// internal/metadata/loader_verify_test.go's own original inline version --
// the first place this pattern was needed).
//
// Scope note on mocks: this codebase's store/session/notification types are
// concrete structs backed directly by *pgxpool.Pool, not interfaces --
// there is exactly one real interface seam to mock, internal/storage.Store
// (see mocks.Storage). Introducing new interfaces elsewhere purely to make
// something mockable would be a bigger refactor than this package's own
// scope; a test needing real Store behavior uses testdb + httptest.NewHandler
// against a real (throwaway-schema) Postgres instead.
//
// New, not a port -- prototype/go never had this. Modeled on portal-ga3's
// own internal/testing/ layout (builders, fixtures, httptest, mocks,
// testdb); see app/docs/portal-ga3-code-quality-benchmark.md for the full
// rationale and what didn't transfer.
package testing
