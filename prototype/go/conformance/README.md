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

**Prerequisites:** server running, Cases 1 & 2 seeded (`seeds/001`, `seeds/002`).

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

---

## Design Notes

- **HTTP black-box** — tests exercise the runtime exactly as a user would; no DB inspection. Capabilities that are DB-only (e.g. CAP-R04 audit log) stay on manual evidence until a UI exposes them.
- **Data pollution accepted** — each run creates a few `ConformanceBot` records. Acceptable for the prototype; a future version should use a disposable workspace.
- **Adding a test:** new ✅ capability → add a `T##` here and in `run.sh`, then set the registry's Proof column to `conformance T##`.
- **State-guard caveat** — T12 approves a record that was Submitted. Once CAP-E06 (state-conditional events) lands, add tests asserting *rejection* of out-of-state transitions.
