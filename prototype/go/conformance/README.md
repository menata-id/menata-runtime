# Conformance Suite

> Study 4 deliverable (`runtime/roadmap.md`).
>
> A capability exists only if an executable test proves it (TCK discipline).
> Every ✅ capability in `runtime/capability-registry.md` must keep its test
> passing — the ratchet rule.

---

## Run

```bash
# against local dev server
./conformance/run.sh

# against any deployment
BASE_URL=https://aksi.menata.id ./conformance/run.sh
```

Exit code 0 = all pass. Non-zero = at least one capability regressed.

**Prerequisites:** server running, seeds `001`, `002`, `003` applied (Cases 1, 2, 18). For T19, also
export `DATABASE_URL` (same value as the server's `.env`) — it's skipped otherwise.

---

## Test → Capability Map

| Test | Capabilities | Proves |
|------|--------------|--------|
| T00 | — | server reachable (gate) |
| T01 | CAP-X01 | multi-application, multi-machine in one workspace |
| T02 | CAP-V01 | form view: fields config drives inputs; status excluded |
| T03 | CAP-V02 | list view: columns config drives table |
| T04 | CAP-C01 | `required` constraint rejects empty submit |
| T05 | CAP-C02 | `after: today` constraint rejects empty submit |
| T06 | CAP-C03, CAP-C04 | conditional constraint fires when condition true (Banner without attachment) |
| T07 | CAP-C04 | conditional constraint silent when condition false (Poster) |
| T08 | CAP-R01 | create record → 303 → default status Draft |
| T09 | CAP-V03 | detail view renders machine fields |
| T10 | CAP-E01, CAP-A01 | permitted role triggers event; set_field applies |
| T11 | CAP-P01 | unpermitted role gets 403 |
| T12 | CAP-P01, CAP-E01 | cross-role transition (Manager approves) |
| T13 | CAP-F13 | reference field renders as a picker, not a bare text input |
| T14 | CAP-F13 | create with an empty (optional) reference succeeds |
| T15 | CAP-F13 | create with a valid reference succeeds; detail links to the target record |
| T16 | CAP-F13 | dangling reference value rejected (negative case — security NFR gate) |
| T17 | CAP-E06 | Approve rejected while record is still Draft |
| T18 | CAP-E06 | Reject rejected on an already-Approved record — the exact Study 1 gap |
| T19 | CAP-C09 | a Constraint satisfied at Create is re-checked at event trigger, not skipped |

---

## Design Notes

- **HTTP black-box** — tests exercise the runtime exactly as a user would; no DB inspection. Capabilities that are DB-only (e.g. CAP-R04 audit log) stay on manual evidence until a UI exposes them. **T19 is a deliberate, documented exception**: it uses `psql` to backdate a date field as test-fixture setup (simulating time passing without waiting years), not to inspect runtime behavior — the assertion itself is still a plain HTTP response check.
- **Data pollution accepted** — each run creates a few `ConformanceBot` records. Acceptable for the prototype; a future version should use a disposable workspace.
- **Adding a test:** new ✅ capability → add a `T##` here and in `run.sh`, then set the registry's Proof column to `conformance T##`.
- **State-guard caveat resolved (2026-07-11)** — CAP-E06 landed; T17/T18 assert rejection of out-of-state transitions, including the exact "Approved record still Rejectable" gap Study 1 found.
