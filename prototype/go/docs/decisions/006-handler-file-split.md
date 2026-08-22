# ADR-006: Line-Length Audit and `handler.go` Domain Split

**Status:** Accepted — implemented in this change
**Date:** 2026-08-22

## Context

An audit was requested against world-class practice for two related but distinct axes —
character width per line, and line count per file — plus a general check of this repo's file/
folder structure. The audit covered the whole `menata-runtime` tree, not just this prototype.

**Line width (characters per line).** Not a systemic problem. Go has no official column limit —
`gofmt` deliberately does not wrap lines, treating width as an author judgment call, not a tooling
one. The closest community convention is the `lll` linter's default of 120 characters. A scan of
this codebase found a small number of lines over 120 chars (25 in `internal/metadata/compile.go`,
40 in `internal/handler/handler.go`, 11 in `internal/executor/executor.go`), concentrated in long
struct literals and string constants rather than a structural pattern. No action taken — not worth
churning diffs over.

**File length (lines per file).** No ISO standard exists, but common tooling converges on similar
thresholds: ESLint's `max-lines` default recommendation is 300/file and `max-lines-per-function`
50/function; golangci-lint's `funlen` defaults to ~60 lines/40 statements per function; SonarQube
and CodeClimate both flag "large file" by default around 500 lines. Clean Code's underlying
principle — a file should hold one reason to change — is the more durable rule than any specific
number.

Measured against that: `internal/handler/handler.go` was **3,244 lines and 86 functions**,
covering session/auth, admin user management, record CRUD, CSV import/export, calendar/
timeline/dashboard rendering, the entire event-trigger and webhook engine, form-field
construction, record-level validation, display formatting, and search/notifications — all in one
file. Every other Go file in the tree was under 950 lines, and the package already had five
sibling files (`api.go`, `processmap.go`, `requirement.go`, `scheduler.go`, `upload.go`) — proof
the team already uses one-file-per-domain within a package, just hadn't extended that pattern to
the original, longest-lived file. This was the one clear, actionable finding.

`.templ` source files (36–178 lines each, generating the larger `_templ.go` files) were already
well within any reasonable bound — no action needed there, and generated `_templ.go` files are
never hand-edited regardless of size (`prototype/go/CLAUDE.md`).

**Documentation files** (`roadmap.md` at 1,743 lines, `capability-registry.md`, `runtime-metadata-
schema.md` at 1,234, etc.) are not held to the same bar. Reference/spec prose is expected to be
long; what matters is heading structure for navigation, not line count. `markdownlint`'s MD013
(line length) exists but is routinely disabled for prose files project-wide for this reason — no
action taken.

**Folder structure.** `cmd/` + `internal/` matches the de-facto `golang-standards/project-layout`
convention and needed no change. The repo-root documents (numbered `001`–`006` plus `README.md`,
`brd-*.md`, `capability-registry.md`, `roadmap.md`, `case-portfolio.md`, `nfr-standards.md`) sit
flat at root rather than under a `docs/` folder the way `guides/` and `benchmarks/` already do —
this is a legitimate style choice at the current document count (the numbering already encodes
reading order), not a defect. No action taken.

Separately, while registering this ADR, `docs/README.md`'s own ADR table was found to only list
`001`–`002` despite `003`–`005` already existing on disk — the same class of "index went stale"
gap ADR-005 named for `nfr-standards.md`. Fixed as part of this change (see Compliance).

## Decision

Split `internal/handler/handler.go` into domain-scoped files, all remaining in package `handler`
(same pattern already used for `api.go`/`processmap.go`/`requirement.go`/`scheduler.go`/
`upload.go` — Go's answer to "split a large package" is more files in the same package, not more
packages, as long as everything still shares the same core type, here `*Handler`). No behavior
changes: every function moves verbatim, nothing is rewritten.

| New file | Responsibility | Moved from `handler.go` |
|---|---|---|
| `handler.go` (trimmed) | Construction, session/identity/role resolution, sub-nav, small shared lookups | `New`, `auth`, `identity`, `identityID`, `workspace`, `roleForApp`, `isWorkspaceAdmin`, `Apps`, `AppMachines`, `uiRoleGroups`, `subNavFor`, `fieldIndex`, `findFieldByID`, `findMachineContainingField`, `toFloat` |
| `auth.go` | Login/logout | `LoginForm`, `Login`, `Logout` |
| `admin.go` | Workspace admin + metadata reload (CAP-X04) | `AdminUsers`, `AdminUpdateUser`, `adminUserRows`, `Reload` |
| `record_crud.go` | Record List/Create/Update/Detail/Archive lifecycle | `List`, `Archive`, `Restore`, `setDeleted`, `MoveRecord`, `Document`, `NewForm`, `Create`, `EditForm`, `Update`, `Detail` |
| `csv.go` | CSV import/export | `csvFieldIDs`, `ExportCSV`, `ImportCSVForm`, `ImportCSV` |
| `views.go` | Calendar/timeline/dashboard/report rendering | `Report`, `calendarTimeline`, `Calendar`, `Timeline`, `Dashboard` |
| `events.go` | Event trigger path, webhooks, workflow actions | `TriggerEvent`, `Webhook`, `triggerEvent`, `processSubscriptions`, `ruleViolation` (+`Error`), `logPermissionDenied`, `logRuleViolation`, `runWorkflowActions`, `doActivateNext`, `doTriggerEvent`, `doAggregateStatus`, `aggregateConditionViolation`, `sequentialGuardViolation`, `separationOfDutiesViolation` |
| `formfields.go` | Form-field construction, child rows, reference lookups | `hiddenFields`, `buildFormFields`, `buildFormFieldsFor`, `childRowName`, `buildChildLinesData`, `validateChildRows`, `insertChildRows`, `childLists`, `referenceOptions`, `referenceLabel`, `userFieldOptions`, `userLabel` |
| `validation.go` | Record-level business-rule checks | `referenceViolations`, `userReferenceViolations`, `immutabilityViolation`, `inScratchState`, `withChangePolicyCreatedAt`, `uniquenessViolations` |
| `format.go` | Display/formatting helpers | `formatAutoNumber`, `boolLabel`, `formatMoney`, `computedValue`, `displayLabel` |
| `search.go` | Search + notifications | `unreadCount`, `Search`, `Notifications`, `SetNotificationPreference`, `MarkNotificationRead` |

This brings every file in the package under 1,000 lines — down from one 3,244-line file to a
largest of 918 (`record_crud.go`, still the single biggest domain: List/Create/Update/Detail/
Archive together) and 696 (`events.go`), with the rest under 300 — while keeping the package
boundary, and therefore every existing import path, unchanged.

## Implementation Strategy

1. **Pure move, not a rewrite.** Each function (with its doc comment, if any) is relocated
   byte-for-byte into its target file. No logic, signature, or behavior changes — this is a
   structural refactor only.
2. **Per-file imports** are resolved from scratch per file (Go requires each file to declare only
   the imports it uses) — `goimports` where available, otherwise iterating on `go build` errors.
3. **Correctness gates, in order:**
   - `gofmt -l .` — must return nothing.
   - `go vet ./...` — must be clean.
   - `go build ./...` — must succeed.
   - Function-set equality check: the sorted list of `^func ` signatures across the new files must
     exactly match the original `handler.go`'s list — nothing dropped, nothing duplicated.
   - `make conformance` against a running local server (`prototype/go/CLAUDE.md`'s documented dev
     loop) — this codebase has no Go unit tests by design; the conformance suite (T1–T158 as of
     this change) is the real proof that request-handling behavior is unchanged.
4. **No schema, seed, migration, or `.templ` changes** — this ADR's scope is strictly the one Go
   package's internal file layout.

## Consequences

**Positive:**
- Largest file in the package drops from 3,244 lines to 918, each new file with one coherent
  responsibility — matches the file-length guidance surveyed in Context and the pattern the
  package's other five files already established. `record_crud.go` (918) remains the largest
  because CRUD lifecycle genuinely is the biggest single domain; a further split was considered
  and rejected — List/Create/Update/Detail/Archive share enough local helpers and validation
  call-sites that separating them would trade one large, coherent file for several small,
  cross-referencing ones.
- Easier to `grep`/navigate/review a change scoped to, say, CSV import without the diff noise of
  an unrelated 3,000-line file, and lower merge-conflict surface when two changes land in
  different domains of what used to be one file.
- No import-path or behavior change for any caller — this is invisible outside the `handler`
  package.

**Negative:**
- More files to open when tracing a call chain that legitimately crosses domains (e.g.
  `triggerEvent` in `events.go` calling into `buildFormFields` in `formfields.go`) — mitigated by
  package-level function names being unique and easy to `grep -rn "^func.*Name"` regardless of
  which file they live in.
- One-time review cost for this change itself, since a pure move still touches every line of a
  3,244-line file — mitigated by the function-set equality check above making the "nothing else
  changed" claim mechanically verifiable rather than only reviewer-asserted.

## Compliance

- Registered in `prototype/go/docs/README.md`'s ADR table (also backfilled that table's missing
  `003`–`005` rows in the same change, per the stale-index gap noted in Context).
- No entry needed in `capability-registry.md` or `roadmap.md` — this is not a capability, it is
  internal code structure with zero behavioral surface.
