# ADR-007: Conformance Suite Split — `run.sh` into `lib.sh` + `tests/NNN_*.sh`

**Status:** Accepted — implemented in this change
**Date:** 2026-08-23

## Context

The user asked directly whether `conformance/run.sh` had grown too large for one file, and what
the best practice for a suite like this should be. It had: **2,128 lines, 164 `check()` call
sites (178 actual invocations across a run, some inside loops/negative-positive pairs), 43
`# --- ...` section banners**, one appended per capability batch since Case 1. Every batch this
codebase implements appends more — the file was never going to shrink (same ratchet dynamic
`capability-registry.md` itself follows).

For comparison, this exact repo already treated a 3,244-line `internal/handler/handler.go` as
needing a split at that size (ADR-006) — `run.sh` had grown to roughly two-thirds that size, past
every general file-length guideline ADR-006 itself surveyed (ESLint's 300-line recommendation,
SonarQube/CodeClimate's ~500-line "large file" flag), for the identical underlying reason: many
unrelated concerns (Design Request through the full Track D UI cluster) accumulating in one file
because nothing ever forced them apart.

**Why not a bash test framework (bats-core etc.) instead of a plain split:** this suite's tests
are **stateful and sequential**, not independent — session cookie jars (`session_for`, cached in
one `$SESSION_DIR` for the whole run), resolved user ids (`ALICE_ID` etc.), and even record state
created by one test consumed by a later one (e.g. `overlay_lifecycle`'s record reused across
several `check` calls) are all shared via plain shell variables across the entire run, by design.
A framework built around independent, isolated test cases would fight that shape rather than fit
it. A structural split that preserves single-process, single-`source`-chain execution — where
everything currently works exactly as `source`-based inclusion already behaves — carries none of
that risk.

## Decision

Split into:

- **`conformance/lib.sh`** — everything every test file needs regardless of which batch it
  covers: `set -u`/`BASE_URL`/`DATABASE_URL`/`PASS`/`FAIL`/`SESSION_DIR`/trap, all 11 original
  HTTP/assertion helpers (`check`, `session_for`, `csrf_for`, `body_contains`,
  `post_body_contains`, `post_status`, `post_status_no_csrf`, `post_redirect`, `get_body`,
  `count_all_pages`, `user_option_id`), the `T00` reachability preflight, and the seeded-account
  session cache (`ALICE`, `BOB`, ... plus the four resolved `*_ID` values).
- **Four helper functions relocated into `lib.sh`**, even though each originally lived inline in
  the one batch that first needed it: `add_business_days`, `overlay_lifecycle`,
  `overlay_negatives`, `process_map_has_shape`. Centralizing them here — rather than leaving them
  in whichever `tests/*.sh` file happened to define them first — matters concretely:
  `overlay_lifecycle` (defined originally next to Process Overlay B1's tests) is called again much
  later by the decompile-lift tests (B6). Leaving it where it was defined would make one
  `tests/*.sh` file silently depend on another one having been `source`d first, in a specific
  order, for a reason invisible from either file alone. In `lib.sh`, any single `tests/*.sh` file
  can be `source`d on its own (after `lib.sh`) without that hidden coupling.
- **`conformance/tests/NNN_*.sh`** — six files, ~200–430 lines each (comparable to ADR-006's own
  resulting file sizes, its largest being 918), holding the actual test content in original order,
  split at the file's own pre-existing `# --- ...` section banners:

  | File | Covers |
  |---|---|
  | `010_case1_3_core.sh` | Design Request / Leave Request / Approval Document+Step / Complaint — Cases 1–3, before the "Batch N" naming convention started |
  | `020_notify_permissions_actionlab.sh` | Notifications, CRUD permissions, audit log, navigation, workspace isolation, real auth, comparison operators, line items, Action Lab, aggregate-conditioned actions — Cases 3–12 |
  | `030_batches4_9.sh` | Batch 4 (Views) through Batch 9 (Workspace Services), 2026-07-12 |
  | `040_batches10_11_subnav.sh` | Batch 10 (Infra), Batch 11 (remaining Field Types), CAP-O03 Tier 2 |
  | `050_process_overlay.sh` | Process Overlay B1–B4 (parity, process map, requirement cardinality, SLA, quorum-of-N), CAP-X04 (metadata live reload) |
  | `060_ui_cluster.sh` | CAP-W03 declarative quorum, Case 9 completion batch (CAP-C08/C10/C11), decompile-lift (B6), the full Track D UI cluster (CAP-V17/V18/V16/V15/V19/V14 Tier 2) |

  Numbered in increments of 10 (`010`, `020`, ...), the same convention `runtime-metadata-
  schema.md`'s own migration numbering uses reasoning for — inserting an unplanned new batch
  between two existing ones never forces renumbering everything after it.
- **`conformance/run.sh`** — trimmed to a ~30-line orchestrator: `source lib.sh`, then `source`
  every `tests/*.sh` file in numeric order, then print the final `$PASS`/`$FAIL` tally exactly as
  before. Its invocation contract is unchanged (`./conformance/run.sh`, `BASE_URL=... ./conformance/
  run.sh`, `make conformance`) — nothing outside `conformance/` needed to change.
- **Adding a test going forward:** append to the last file if it's the same theme/batch, or create
  the next `tests/NNN_*.sh` (increment by 10) if it's a new one — stated directly in `conformance/
  README.md`'s own "Adding a test" bullet now, so this doesn't need rediscovering.

## Implementation Strategy

Pure move, no assertion logic changed — same discipline ADR-006 established for `handler.go`:

1. **Mechanical extraction**, not a rewrite — every line of actual test/assertion content moved
   verbatim into its target file; only file-level header comments are new.
2. **Correctness gates, in order:**
   - `bash -n` on every output file — must be syntactically valid.
   - **`check()`-call-site inventory, in exact order**: every `check T##` line in the original
     file, compared against the concatenation of `lib.sh` + all six `tests/*.sh` files in run
     order — 164/164, same sequence, none dropped or duplicated.
   - **Function-definition inventory**: the sorted set of all 15 top-level function names in the
     original must equal the sorted set across `lib.sh` + all `tests/*.sh` files — exact match.
   - **Line-accounting**: every content line of the original (1–2125; the trailing summary block,
     2126–2128, intentionally moves into the new `run.sh` instead) accounted for exactly once
     across the output files — no line lost, none duplicated.
   - **Behavioral equivalence on two independent fresh schemas** (not the shared, repeatedly-run
     dev database, which was already carrying known non-idempotent-reseed artifacts unrelated to
     this change — see `benchmarks/020-ui-interaction-cluster-proof.md`'s own note on this): the
     unmodified original `run.sh` run against a freshly migrated+seeded schema (`test_split1`) and
     the new split version run against a second, independently freshly migrated+seeded schema
     (`test_split2`) both produced **177 passed, 0 failed**, and the per-test `PASS`/`FAIL`
     sequence diffed byte-for-byte identical between the two runs.
3. **No schema, seed, migration, or Go source changes** — this ADR's scope is strictly the
   conformance suite's own file layout, mirroring ADR-006's own stated scope limit.

## Consequences

**Positive:**
- Largest file drops from 2,128 lines to 430 (`030_batches4_9.sh`), each with one coherent theme.
- A change scoped to one capability area no longer touches a 2,128-line file's diff; lower
  merge-conflict surface across concurrent work on different capability areas (directly relevant
  here — this suite is under active, frequent, sometimes concurrent editing).
- Centralizing `overlay_lifecycle` and its three siblings into `lib.sh` removes a real, previously
  invisible cross-file ordering dependency, and makes "run just this one batch's tests" (`source
  lib.sh && source tests/060_ui_cluster.sh`) a real, supported thing to do while debugging.
- `run.sh`'s own invocation contract, and everything outside `conformance/` that calls it
  (`Makefile`'s `conformance:` target, `DEVELOPMENT.md`, `CLAUDE.md`), needed zero changes.

**Negative:**
- One more level of indirection to trace a call chain that legitimately crosses files (a helper
  defined in `lib.sh`, called from a `tests/*.sh` file) — mitigated the same way ADR-006 accepted
  for its own split: function names are unique and `grep -rn` works regardless of which file.
- Seven files to open instead of one when reading the whole suite start-to-finish — accepted as
  the direct trade for the file-size and merge-conflict benefits above, the same trade ADR-006
  already made for `handler.go`.

## Compliance

- Registered in `prototype/go/docs/README.md`'s ADR table.
- `conformance/README.md`'s "Adding a test" bullet and its ACCOUNTS-comment-block pointer updated
  to name `lib.sh`/`tests/*.sh` instead of the now-retired monolithic `run.sh`.
- No entry needed in `capability-registry.md` or `roadmap.md` — like ADR-006, this is conformance
  suite file layout, not a capability, and has zero behavioral surface (proven above).
