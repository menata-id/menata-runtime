# NFR Standards per Capability Area

> Study 10 deliverable (`roadmap.md` Phase 3).
>
> World-class **architecture, performance, and security** requirements for every
> capability area in the registry — as *kajian* (study/standard), enforced later
> as Definition-of-Done gates when each capability is implemented
> (`capability-lifecycle.md` §3b).
>
> Status: v0.1 — study only, no implementation | Created: 2026-07-04

**External standards used as yardsticks:**

| Dimension | Standard | Used for |
|-----------|----------|----------|
| Security | OWASP ASVS 4.x | Verification checklist per area (chapter refs below) |
| Security | STRIDE | Threat profile per area |
| Performance | Google SRE (SLO / error budget) | Performance budget classes |
| Architecture | ISO/IEC 25010 | Quality attributes vocabulary (same standard Portal GA BRD §14 uses) |
| Architecture | Fitness functions (Ford et al.) | Executable architecture checks (same mechanism as Portal GA FF catalog) |

---

# 0. The novel threat class: metadata is code

Traditional application security assumes *code is trusted, data is untrusted*. A metadata-driven runtime breaks that assumption: **metadata sits in between** — authored by workspace designers who are *less trusted than the runtime developers but more trusted than end users*.

Consequences that shape every area below:

1. **Metadata injection** — a field named `<script>…` or a value_list option carrying HTML is *stored XSS via metadata*. All metadata strings are untrusted output at render time.
2. **Logic bombs via metadata** — an event action chain that mass-updates or notify-spams is expressible declaratively. Budgets and limits must be *enforced by the runtime*, not assumed of authors.
3. **Confused deputy** — the runtime executes actions with system privileges on behalf of metadata authored by a workspace designer. Every executor must re-check the *triggering actor's* permission, never its own.
4. **Cross-tenant reach** — metadata must be constitutionally unable to name another workspace's machines, records, or users (RLS + loader validation, ADR-003).

**Baseline threat model (runtime core), STRIDE:**

| Threat | Runtime exposure | Countermeasure class |
|--------|-----------------|---------------------|
| Spoofing | Prototype cookie role (accepted for PoC); production needs real authn | ASVS V2/V3 session management (CAP-X02) |
| Tampering | Direct record/metadata mutation bypassing engines | Single write path through engines; DB-level immutability (CAP-R07); append-only event log |
| Repudiation | Actions without actor attribution | `record_events` must always carry actor + correlation_id (CAP-I04) |
| Information disclosure | Everything readable by everyone today (CAP-P05 gap); search/list leaks | Deny-by-default reads; RLS; permission-trimmed queries |
| Denial of service | Unbounded lists, unbounded action fan-out, report queries on OLTP pool | Pagination (CAP-R05), action budgets, pool separation (Study 8) |
| Elevation of privilege | `set_field` mass assignment; event trigger on wrong state | Field allow-lists per form view; state guards (CAP-E06); confused-deputy rule above |

---

# 1. Performance budget classes

Every capability, when implemented, declares one class. Budgets are p95, single modest server, at the Study 8 reference scale (100 ws × 50 machines × 1M records).

| Class | Budget (p95) | Applies to |
|-------|-------------|-----------|
| **P1 interactive read** | < 200 ms | list, detail, form render, search |
| **P2 interactive write** | < 500 ms | create, event trigger incl. constraint evaluation + synchronous actions |
| **P3 heavy read** | < 5 s, separate pool, `statement_timeout` | reports (V13), dashboards (V10), export |
| **P4 async** | no user-facing latency; completion SLO instead (e.g. 99% within 60 s) | notifications, webhooks out, batch integration, index reconciliation |
| **P5 boot/reload** | < 5 s cold workspace load; < 1 s reload swap | metadata loading (X11) |

Rule: a synchronous request path may not contain a P4 operation — slow work is dispatched, never awaited (mirrors ADR-0012 fire-and-forget).

---

# 2. NFR profiles per capability area

### 2.1 Field Types (CAP-F*)

- **Security** — ASVS V5 (input validation), V12 (files). All field values validated server-side by *type* (number parses, date parses, value_list value ∈ declared set — currently unchecked!). `rich_text` sanitized on output (allow-list, never raw HTML). File uploads (F06): content-type sniffing not extension, size limits, stored outside webroot with generated names, served with `Content-Disposition`. Reference fields (F13): target existence + *same-workspace* check at write time (IDOR via forged reference ID).
- **Performance** — P1/P2. Rendering O(fields) with zero per-field queries; reference display names batch-resolved (no N+1); hot fields indexable (CAP-X10).
- **Architecture** — each type = one registered triple (renderer, validator, storer) at the field-type seam; no type-switch sprawl in handlers. Fitness function: no field-type conditionals outside the registry.

### 2.2 Event Sources (CAP-E*)

- **Security** — ASVS V4 (access control). Business events: role + state guard (E06) + record-level check (P02) evaluated server-side per trigger. External events (E04): signature verification (HMAC), replay protection (timestamp + nonce), idempotency keys. Time events (E02): run as *system actor* with explicit, logged identity — never inherit a random user's context.
- **Performance** — P2 for user triggers; scheduler (E02/E03) must avoid thundering herd (jitter, per-workspace batching), drift bound documented.
- **Architecture** — event-source registry seam; every firing produces exactly one audit entry with actor + correlation_id; deterministic dispatch order within a record.

### 2.3 Actions (CAP-A*)

- **Security** — confused deputy rule (§0.3): executor re-checks triggering actor. `set_field` restricted to declared fields (mass-assignment defense). `notify` rate-limited per actor + per workspace (spam/logic-bomb budget). Cross-machine `create_record` (A06): target machine's create-permission evaluated, not bypassed.
- **Performance** — synchronous action chain budget: total P2; notify/webhook always P4 (queued). Fan-out cap per event (documented limit, logged when hit — "no silent caps").
- **Architecture** — executor seam per action type; ADR-0012's four error-isolation rules are constitutional (consumer failure never breaks source; per-handler isolation; nil-safe; batch continues). Domain engines (posting derivation) plug here.

### 2.4 Constraints (CAP-C*)

- **Security** — constraints are a *security control*, not just UX: they must run on **every** write path (Create, Update, event trigger — CAP-C09; API — X07; import — R06). Client-side validation is advisory only. Cross-record constraints (C08/C11): TOCTOU-safe (evaluated inside the write transaction, `SELECT … FOR UPDATE` or serializable).
- **Performance** — in-record constraints in-memory O(constraints); cross-record constraints must hit an index (declared hot field → X10); P2 budget includes full constraint pass.
- **Architecture** — operators are pure functions in the operator registry; unknown operator = load-time report (lifecycle §4.3); constraint evaluation deterministic and side-effect-free.

### 2.5 Permissions (CAP-P*)

- **Security** — ASVS V4 wholesale. **Deny by default** (today: allow by default — inversion required). Checks server-side at the guard chokepoint; IDOR defense = every record fetch scoped by workspace + permission, never by ID alone. Separation of duties (P03) evaluated against *identity*, not role string. Role strings namespaced (CAP-O01) to prevent cross-application collision escalation.
- **Performance** — P1 overhead ≈ 0: permission maps precomputed per interpreter version; record-level checks piggyback the record fetch (no second query).
- **Architecture** — exactly one guard chokepoint; no permission logic in handlers/templates. Fitness function: grep for role comparisons outside the guard = violation.

### 2.6 Views (CAP-V*)

- **Security** — output encoding everywhere (Templ auto-escaping verified by test, incl. metadata-sourced strings — §0.1); CSV/Excel export (R06) formula-injection escaped (`=`, `+`, `-`, `@` prefixes); list/detail/search results permission-trimmed (P05) — a view config must not widen access; report drill-down re-checks permission at each level.
- **Performance** — P1 lists (pagination mandatory — R05), P3 reports/dashboards on the separate pool; declared filters/sorts must be index-backed (X10) — a view whose filter can't use an index is a load-time warning.
- **Architecture** — view-type renderer seam; view config is presentation *intent* only — no data-access rules live in views (they live in permissions).

### 2.7 Record Lifecycle (CAP-R*)

- **Security** — audit log (R04) append-only at the DB level (no UPDATE/DELETE grants), carries actor + before-snapshot + correlation_id; immutability-after-state (R07) enforced by trigger/constraint in the database, not only by application code; import (R06) runs the full constraint + permission pipeline (no side door).
- **Performance** — P2 writes; event log partitioned by month, retention per workspace (Study 8); import is P4 batch with progress reporting.
- **Architecture** — one write path: every mutation flows through recorder logic that snapshots, validates, mutates, logs — in one transaction.

### 2.8 Cross-Cutting (CAP-X*)

- **Security** — authn (X02): ASVS V2/V3 (session fixation, rotation, expiry, CSRF tokens on all state-changing requests). Metadata validation (X05) is the **injection firewall**: schema-validate + sanity-check all metadata at load; reject dangling refs, oversized configs, cross-workspace references. API (X07): same guard chokepoint as HTML, token auth, rate limits. Package import (X08): signed/checksummed packages, dry-run diff before apply — a metadata package is executable content (§0).
- **Performance** — X11 cache: stampede-safe (singleflight), stale-while-revalidate acceptable, P5 budgets; X10 index reconciliation uses `CREATE INDEX CONCURRENTLY` only.
- **Architecture** — versioned schema; RLS as isolation floor (ADR-003); fitness functions run in CI (Portal GA model).

### 2.9 Cross-Machine Integration (CAP-I*)

- **Security** — events cross a trust boundary even in-process: consumers validate payloads against contract (I03) — `on_contract_violation` is a security response, not just resilience; correlation_id (I04) is an opaque UUID carrying no PII; contribution weights (I05) capped server-side (gaming/points-inflation defense); subscriptions (I01) cannot cross workspaces.
- **Performance** — P4: dispatch decoupled from the source's P2 budget; fan-out cap + backpressure; per-consumer SLO (Portal GA SLO registry model).
- **Architecture** — the four error-isolation rules again, at the dispatcher itself; event schema versioning with deprecation lifecycle (I02) — consumers pin compatible ranges.

### 2.10 Workspace Services (CAP-O*)

- **Security** — identity/role registry (O01) is the *single* authz source of truth — services may not keep shadow role copies. Global search (O04): permission-trimming is the hard requirement — an index that returns forbidden snippets is a data breach, so index entries carry ACL context. Notification center (O05): content minimization in push/email (titles, not payloads — Portal GA quiet-hours/severity precedent). Master data (O02): deactivation semantics defined (no orphan references, no accidental cascade delete). Calendar (O06): low sensitivity, but write access = ability to shift SLA/due-date computation org-wide → admin-only.
- **Performance** — search P1 with its own index (never LIKE-scans over JSONB); notification fan-out P4; calendar lookups cached per workspace (tiny, hot).
- **Architecture** — workspace-service seam (lifecycle §4); services are workspace-scoped singletons; no service reaches into another workspace ever.

---

# 3. How this binds (relationship to lifecycle)

This document is the **standard**; `capability-lifecycle.md` §3b makes it a **gate**: a capability cannot pass Incubating → Supported until its area's NFR profile is either satisfied (with evidence: test, fitness function, or measurement) or explicitly waived with reason in the registry row.

Study-only note: nothing in this document is implemented yet. The prototype at `aksi.menata.id` is a PoC and intentionally exempt (accepted risk, recorded in §0 Spoofing row).

---

# 4. Maintenance

- New capability area → add an NFR profile here before first implementation in that area.
- New threat class discovered → update §0 and the affected profiles.
- Budgets re-baselined when the Study 8 load-test matrix first runs on real hardware.
