# ObjectStack — Backend Architecture and Repository Structure

> Part of Study 37 (`../README.md`). Every path below is relative to the ObjectStack clone at commit
> `ac76425f` (2026-09-07) unless prefixed with `app/`, which means Menata Runtime's own codebase.
> Status: v1.0 | Created: 2026-09-07 | Updated: 2026-09-07

---

## 1. Positioning and stated principles

ObjectStack's own north star (`content/docs/concepts/north-star.mdx`) and design principles
(`content/docs/concepts/design-principles.mdx`) reduce to seven rules; the right-hand column is
Menata Runtime's counterpart so the overlap is visible before the differences:

| # | ObjectStack rule | Menata Runtime counterpart |
|---|---|---|
| 1 | **Zod first** — schemas are the source of truth; TS types and JSON Schemas derive from them | `runtime-metadata-schema.md` + `internal/model` structs + `internal/metadata` loader; no schema-language generator |
| 2 | **Metadata is the app contract** — objects, fields, views, flows, actions, agents, permissions, datasets, translations | `004-runtime-metadata.md` — Workspace › Application › Machine › {Field, Event, Constraint, Permission, View} |
| 3 | **Runtime identity is environment identity** (`OS_ENVIRONMENT_ID`, `X-Environment-Id`) | Workspace (`CAP-X06`, `CAP-X14` `/{slug}/`) |
| 4 | **Deployment config is not artifact content** — DB URLs, secrets, hostnames injected by host | `internal/config` env vars; same posture |
| 5 | **AI exposure is governed** — an action becomes an MCP tool only via `ai: { exposed: true }` | no AI surface yet (see comparison §H1) |
| 6 | **Automation is Flow-first** — event automation, scheduled work, approvals, waits, screens, notifications are Flow nodes; state machines model strict lifecycles | *Emergent* model — Events + Actions + Constraints + Permissions, plus the Process Overlay that compiles a declared `process` into them (Studies 19–21) — the opposite architectural bet, deliberately |
| 7 | **The console is a consumer, not the source of truth** — UI source lives in sibling repo `objectui` | `internal/ui` (Templ) is *inside* the runtime — the runtime owns rendering, not a client |

Principle I ("Protocol Neutrality: the Protocol is law, the Implementation is merely an opinion")
and III ("There is no 'Code'. There is only Schema.") are close to `001-design-principles.md`'s
Metadata First. Principle II ("Mechanism over Policy") is where they differ from Menata Language's
human-first stance: ObjectStack's policies are CEL predicates authored in TypeScript, optimised
for an *AI author's priors* (ADR-0020 says so explicitly — "meet the model where its priors are"),
not for a domain expert writing `.menata`.

## 2. Repository layout

Top level (`ls` of the clone):

```
AGENTS.md  ARCHITECTURE.md  CLAUDE.md  CHANGELOG.md  CONTRIBUTING.md  ROADMAP.md  RELEASE_NOTES.md
LICENSE  LICENSING.md  CODE_OF_CONDUCT.md
apps/docs/            Fumadocs + Next.js docs site (content comes from content/)
content/docs/         438 .mdx pages — the objectstack.ai/docs source; references/ is GENERATED from Zod .describe()
content/blog/
docker/               official runtime image (ghcr.io/objectstack-ai/objectstack)
docs/adr/             131 ADRs (0001–0131, a few withdrawn/duplicated numbers) + PRIORITIZATION.md
docs/{HARDENING,OBSERVABILITY,DX_ROADMAP,PLATFORM_GAPS_FROM_TEMPLATES,PLUGIN_ECOSYSTEM_MAP}.md
docs/{audits,design,handoff,notes,plans,qa,screenshots}/
examples/{app-todo,app-crm,app-showcase,app-multi-package,embed-objectql}/
packages/             the framework (below)
scripts/              repo gates (type-check coverage ratchet, cross-package test input gate, docs sync…)
skills/               AI-agent skill bundle installed into scaffolded projects (objectstack-{data,ui,automation,formula,query,api,ai,i18n,platform,upgrade,pm-dispatch})
.changeset/  .claude/  .githooks/  .github/  turbo.json  pnpm-workspace.yaml  sdui.manifest.json
```

`packages/` (45+ workspace packages; `README.md`'s own "Package Directory" is the canonical
list — categories below are the directory structure, roles from `ARCHITECTURE.md` + each
package's source):

| Directory | Package(s) | Role |
|---|---|---|
| `packages/spec` | `@objectstack/spec` | **The protocol.** 864 non-test `.ts` files in 17 domains: `data/` (object, field, query, filter, hook, validation, analytics cube, seed, datasource, driver contracts), `ui/` (app, view, page, dashboard, chart, report, dataset, action, i18n, notification), `automation/` (flow, nodes, approval, state-machine, webhook, time-relative trigger, BPMN interop), `security/` (permission set, RLS, sharing, tenancy posture, explain), `identity/` (user, organization, position, SCIM), `ai/` (agent, tool, MCP, knowledge, RAG, model registry), `kernel/` (plugin, manifest, lifecycle, metadata loader/protection, package registry/upgrade, cluster), `system/` (auth config, email, storage, cache, job, tenant, migration, encryption, metrics, tracing, license), `api/` (REST, batch, OData, realtime, error-code ledger, versioning), `contracts/` (every service interface: `IDataDriver`, `IMetadataService`, `IApprovalService`, …), `cloud/` (environment artifact, package, marketplace), `integration/` (connectors), `studio/`, `qa/`, `shared/`, `migrations/`. Zero runtime deps except Zod. Also ships `authorable-surface/`, `declaration-map/`, `json-schema.manifest/`, `prompts/` — generated projections of the same schemas |
| `packages/core` | `@objectstack/core` | **Microkernel**: `kernel.ts` (`ObjectKernel`), `lite-kernel.ts`, `plugin-loader.ts`, `plugin-order.ts`, `dependency-resolver.ts`, `hook-dispatch.ts`, `health-monitor.ts`, `hot-reload.ts`, `timeout-guard.ts`, `logger.ts` (pino), plus `security/` (authz context assembly, permission manager, grant cache posture ladder, plugin signature/integrity, sandbox runtime, API keys) and `utils/` (bulk-write, migration journal, metadata activation store) |
| `packages/types`, `packages/formula` | `@objectstack/types`, `@objectstack/formula` | shared TS utilities; **the expression engine** — `cel-engine.ts` (cel-js), `stdlib.ts`, `cel-to-filter.ts` (CEL → query filter pushdown), `cel-pushdown-limits.ts`, `rls-predicate.ts`, `template-engine.ts`, `cron-engine.ts`, `validate.ts` |
| `packages/platform-objects` | `@objectstack/platform-objects` | Built-in `sys_*` object schemas: `identity/`, `security/`, `audit/`, `metadata/`, `apps/`, `pages/`, `integration/`, `system/` — the platform's own tables are ordinary objects |
| `packages/objectql` | `@objectstack/objectql` | **Data engine**: `engine.ts` (CRUD, hooks, validation, transactions, cascade delete, autonumber, summary rollups, data events), `registry.ts` (schema registry), `hook-binder.ts`/`hook-wrappers.ts`/`hook-run-as.ts`, `master-detail.ts`, `search-filter.ts`, `validation/{record,rule}-validator.ts`, `integrity/dangling-reference-audit.ts`, `tenancy/`, `lifecycle/`, `action-governance.ts`, `secret-fields.ts` |
| `packages/metadata`, `metadata-core`, `metadata-fs`, `metadata-protocol` | `@objectstack/metadata*` | Metadata loading (TS/JSON/YAML), file watching, `sys_metadata` + `sys_metadata_history` persistence, ETag caching, the `metadata` service (sole provider, ADR-0008) |
| `packages/runtime` | `@objectstack/runtime` | Bootstrap: `DriverPlugin`, `AppPlugin`, dispatcher plugin (security headers, request id, rate limit, metrics seam), environment registry, artifact loading (`eager`/`lazy`/`artifact-only`) |
| `packages/rest` | `@objectstack/rest` | Auto-generated REST: `rest-server.ts`, `route-manager.ts`, `rest-route-ledger.ts`, `openapi-endpoints.ts`, `query-allowlist.ts`, CSV/XLSX `import-*.ts`/`export-format.ts`, `external-datasource-routes.ts`, `package-routes.ts` |
| `packages/drivers/*` | `driver-memory`, `driver-sql` (knex: pg / mysql2 / better-sqlite3 / tedious), `driver-mongodb`, `driver-turso` (libSQL), `driver-sqlite-wasm` | `IDataDriver` implementations; `driver-sql/src/sql-driver.ts` alone is 17,486 lines |
| `packages/adapters/hono` | `@objectstack/hono` | The supported HTTP adapter (Node, Bun, Deno, Cloudflare Workers) |
| `packages/plugins/*` (14) | `plugin-{hono-server,auth,security,sharing,approvals,audit,email,webhooks,reports,dev,pinyin-search}`, `embedder-openai`, `knowledge-{memory,ragflow}` | Optional capabilities as kernel plugins; `plugin-security` (RBAC/RLS/FLS, 40+ source files incl. `rls-compiler.ts`, `permission-evaluator.ts`, `explain-engine.ts`, `field-masker.ts`, `tenant-layer.ts`), `plugin-approvals` (approval node runtime + `sys_approval_{request,action,approver,delegation,token}` objects) |
| `packages/services/*` (17) | `service-{analytics,automation,cache,cluster,cluster-redis,datasource,i18n,job,knowledge,messaging,package,queue,realtime,settings,sms,storage}` | Long-running services behind spec contracts; `service-automation` is the flow engine (`engine.ts`, `builtin/{crud,logic,loop,map,parallel,wait,screen,subflow,http,connector,notify,try-catch}-node*.ts`, `suspended-run-store.ts`, `flow-dispatch-store.ts`) |
| `packages/connectors/*` | `connector-{rest,openapi,mcp,slack}` | Declarative outbound integrations (ADR-0097, ADR-0023, ADR-0024) |
| `packages/{client,client-react}` | `@objectstack/client`, `@objectstack/client-react` | Typed SDK (`client.meta.*`, `client.data.*`, `client.views.*` saved-view storage), React hooks |
| `packages/console` | `@object-ui/console` | Prebuilt bundle pulled from the sibling `objectui` repo (`pnpm objectui:refresh`) — the runtime serves it, does not build it |
| `packages/{cli,create-objectstack}` | `os` / `objectstack` CLI | `init`, `dev`, `start`, `serve`, `compile`, `validate`, `lint`, `generate`, `doctor`, `explain`, `migrate`, package publish/install against a Cloud control plane |
| `packages/{mcp,triggers,observability,lint,verify,qa,sdui-parser,cloud-connection}` | — | MCP server (`/api/v1/mcp`, OAuth per deployment), trigger family (ADR-0041), metrics/tracing contracts, metadata linter, conformance verifier (ADR-0060 ledgers), SDUI JSX parser (ADR-0080/0081: React pages parsed, never executed, unless trusted tier) |
| `packages/apps/{account,setup,studio}` | — | Identity/org portal, the built-in Setup app (rendered from the same protocol), Studio designer bindings |

**Reading the layout as a whole.** Everything is "spec first, then engine, then plugin": a
feature is legal only once its Zod schema exists in `packages/spec`, and the repo's own gates
(`scripts/check-*`, ADR-0054 "runtime proof for authorable surface", ADR-0078 "no silently inert
metadata") try to keep the engine from lagging the spec. That the gates exist at all is the tell —
the spec grew far ahead of enforcement early on (§9 below).

## 3. Microkernel and layering

`ARCHITECTURE.md` describes a microkernel (`packages/core/src/kernel.ts`) with four facilities and
a plugin contract:

```
ObjectKernel
├─ Plugin Lifecycle Manager   init → start → running → destroy; topological dependency sort
├─ Service Registry (DI)      registerService(name, svc) / getService<T>(name)
├─ Event Bus (hooks)          hook(name, handler) / trigger(name, ...args)   e.g. 'data:record:afterCreate'
└─ Logger (pino)

interface Plugin { name: 'com.acme.crm.x'; version?; dependencies?: string[];
                   init(ctx); start?(ctx); destroy?() }
```

Plugins never import each other; they couple at runtime through the service registry (rule 4 in
`ARCHITECTURE.md` §"Dependency Rules"). Plugins declare `implements / provides / requires /
extensionPoints / extensions` manifests (`packages/spec/src/kernel/plugin-capability.zod.ts`).
Above the kernel sit three protocol layers with explicit "knows about / doesn't know about"
boundaries:

| Layer | Packages | Knows | Doesn't know |
|---|---|---|---|
| **ObjectQL** (data) | `objectql`, `drivers/*` | schema, fields, queries, drivers | users, permissions, UI |
| **Kernel** (control) | `runtime`, `plugin-*`, `service-*` | auth, workflows, events | data structure, UI layout |
| **ObjectUI** (view) | `client-react`, `console` (sibling repo) | layout, navigation, actions | business logic, storage |

**Menata Runtime counterpart.** `app/ARCHITECTURE.md`'s layer diagram is a *compile-time* package
graph (handler → executor/interpreter/permission → constraint/metadata/ui/storage → model/store/
auth/db/config), measured cycle-free by `go list`. There is no runtime service registry, no plugin
lifecycle, no DI container; the "seam" for growth is `capability-lifecycle.md` §4's registry-seam
pattern (a Go map from a metadata keyword to an implementation, e.g. field types, action types)
— the same extension *idea* as ObjectStack's node/field/driver registries, realised as a
compile-time table rather than a runtime plugin bus. The cost ObjectStack pays for runtime
pluggability is visible in `packages/core/src/`: `plugin-order.ts`, `dependency-resolver.ts`,
`startup-orchestrator.zod.ts`, `plugin-loading.zod.ts`, `plugin-security*.zod.ts`,
`plugin-artifact-signature.ts` — an OSGi-class problem set Menata Runtime simply doesn't have.

## 4. Metadata pipeline: authoring → artifact → runtime

| Stage | ObjectStack | Menata Runtime (`app/`) |
|---|---|---|
| Authoring | TypeScript files: `ObjectSchema.create({...})`, `Field.text({...})`, `defineView`, `defineFlow`, `defineAction`, `definePermissionSet`, `definePosition`, `defineDashboard`, `defineJob`, all assembled by `defineStack({ manifest, objects, views, apps, dashboards, datasets, flows, hooks, positions, permissions, data, translations, ... })` in `objectstack.config.ts` (`examples/app-crm/objectstack.config.ts`). Studio (visual designer) authors the same metadata; AI agents author it via the `skills/` bundle | YAML per Machine (`docs/examples/*.yaml`) translated into SQL seeds (`app/seeds/*.sql`) inserted into metadata tables (`workspaces`, `applications`, `machines`, `fields`, `events`, `event_actions`, `constraints`, `permissions`, `views`, `event_subscriptions`) — `guides/writing-runtime-metadata.md`. An Authoring Layer is the intended producer (`004-runtime-metadata.md`); today it is by hand |
| Validation gate | Strict TS + Zod `.strict()` objects with typo hints and retired-key tombstones (`data/object.zod.ts` `UNKNOWN_KEY_GUIDANCE`), then `os validate` (dangling bindings, bad CEL, missing security posture) and `os lint` (best-practice rules, e.g. `security-owd-unset`, `sharing-rule-object-not-shareable`, `unique/double-declaration`) | `internal/metadata/validate.go` at load (`CAP-X05`): dangling references, unknown operators, `money` currency rule, etc.; one bad Machine fails the whole boot (`runtime-metadata-schema.md` §"Load-Time Contract") |
| Compile | `os compile` → `dist/objectstack.json` (`ObjectStackDefinitionSchema`); Cloud wraps it in an immutable `EnvironmentArtifact` (`schemaVersion`, `environmentId`, `commitId`, SHA-256 `checksum`, `metadata`, `grantedPermissions`) | `internal/metadata/compile.go` — the Process Overlay compiler (`process` block → Events/guards/Permissions, CAP-W01/W03/W04/W05) and approval-quorum injection; no artifact file — the DB *is* the artifact |
| Load | Read at boot into memory by `MetadataPlugin` (sole `metadata` service provider); ObjectQL syncs definitions into its registry and subscribes to metadata events. DB-backed metadata (`sys_metadata`, `sys_metadata_history`, ADR-0067 commit log + rollback) is an explicit control-plane opt-in, not the runtime default (`content/docs/releases/implementation-status.mdx` §"Metadata Framework") | `Loader.LoadAll` → `*model.Application` tree; `CAP-X04` admin-triggered atomic interpreter swap for live reload; `CAP-X08` export/import of one Application's tree |
| Versioning | Per-item `version_hash`; `executionPinned` types (flow, approval) keep every historical version resolvable forever so a paused run keeps its definition (ADR-0009); metadata authoring lifecycle draft→published with visibility gate (ADR-0027, ADR-0045) | `CAP-W07` `change_policy` (effective-dated: `new_records` / `records_in_states` / `all_records`) — a different, arguably better-scoped answer to the same problem; blanket pinning (`CAP-W02`) deliberately not built (Study 20 §6.4) |
| Multi-tenancy of metadata | One artifact per *environment*; `organization_id` on data rows; packages installable per environment with customization overlays (ADR-0005, ADR-0126) | Workspace tree loaded for all workspaces in one pass; per-workspace Postgres schema + RLS (`CAP-X06`) |

## 5. Request path (data write)

Reconstructed from `packages/runtime` (dispatcher), `packages/rest`, `packages/plugins/plugin-security`,
`packages/objectql/src/engine.ts` and `packages/services/service-automation`:

```
HTTP (Hono adapter)
 └─ dispatcher plugin: security headers · X-Request-Id · rate limit (opt-in) · metrics seam
    └─ auth (better-auth session / API key / MCP OAuth) → ExecutionContext (user, org, positions, memberships)
       └─ REST route (/api/v1/data/{object}) — OpenAPI-described; batch endpoint; import/export
          └─ plugin-security middleware on IObjectQL: Layer-0 tenant wall → object CRUD gate (permission-set union)
             → RLS policies compiled to filter (rls-compiler.ts, membership IN-sets) → FLS masking on read
             └─ ObjectQL engine: beforeInsert/beforeUpdate hooks → validation rules (script/cross_field/
                state_machine/unique/format/json_schema…) → autonumber/formula/summary → driver write
                (knex, ambient transaction ADR-0034) → afterX hooks → data events on the kernel bus
                └─ service-automation: record_change flows bound to the object → node graph, durable
                   pause (approval/screen/wait) persisted in sys_automation_run / suspended-run store
                   └─ notifications ingress → outbox → channels (email plugin, in-app, webhooks plugin)
```

**Menata Runtime path** (`app/internal/handler` → `permission.Guard` → `constraint` →
`executor.Executor` → `store.RecordStore` → `record_events` + `action_outbox`) is the same shape
with fewer stages: one permission model (role/ownership/SoD, `CAP-P01–P07`), one constraint
evaluator, one executor whose actions run synchronously except those routed to the
`CAP-W06` outbox, and no hook/plugin dispatch in between. Notably both runtimes converged on the
same three hard-won facts, independently: **writes must be atomic across records** (ADR-0034 ↔
`CAP-X12`), **webhooks must be idempotent** (`plugin-webhooks` outbox ↔ `CAP-X13`), and **the
server enforces, the client is courtesy** (ADR-0124 ↔ `CAP-C09`'s "client is advisory, server
enforces").

## 6. Storage model — the biggest structural difference

| | ObjectStack `driver-sql` | Menata Runtime `app/` |
|---|---|---|
| Physical shape | **One table per object**, one column per field (`initObjects` → `knex.schema.createTable` / `alterTable`, `sql-driver.ts:9641–9791`); multi-valued and composite/repeater fields stored as JSON columns; lookups as FK string columns | **One `records` table** per workspace schema: `id UUID, machine_id, data JSONB, created_by, created_at, updated_at` + `GIN(data)` (`app/migrations/002_data_schema.sql`); `record_events` append-only with pre-mutation `snapshot` |
| Schema evolution | Additive-only sync (create missing tables, add missing columns; never alter/drop); a separate **schema-drift detector** classifies divergence as `safe` / `needs_confirm` / `destructive` and `os migrate apply --allow-destructive` reconciles (`driver-sql/src/schema-drift.ts`) | None needed — adding a Field is a metadata row; no DDL per Machine. Cost: no per-field column types/constraints in the database |
| Uniqueness | `unique: 'organization'` materialises as `(COALESCE(organization_id,'__global__'), …)` — NULL-safe tenant uniqueness (ADR-0120); `unique: 'global'` = the listed columns | `CAP-C12` composite uniqueness enforced in the write path (`uniquenessViolations`) over JSONB, not by a DB constraint |
| Indexes | Declared `indexes[]` on the object → real DB indexes; lookup FK indexes | `GIN(data)` only; `CAP-X10` (metadata-driven expression indexes from view filters/sorts) registered ❌, deliberately deferred until something is measurably slow |
| Sequences | `SEQUENCES_TABLE` + `getNextSequenceValue` with `forUpdate` and savepoints | `field_sequences` table (`CAP-F18`) |
| Dialects | Postgres, MySQL, SQLite (+ MSSQL via tedious), MongoDB, Turso/libSQL, in-memory, SQLite-WASM | Postgres only (pgx) |
| Datasources | Multiple named datasources, object→datasource mapping, external datasource federation with read-only/validated external schemas (ADR-0015, ADR-0062) | one database per deployment |
| Transactions | Ambient transaction context threaded through every engine→driver call (ADR-0034, after a deadlock bug on SQLite's single connection) | `CAP-X12` one `pgx.Tx` per request across Machines |

**Performance reading.** Table-per-object gives ObjectStack native column types, B-tree indexes on
any field, real FKs and DB-level uniqueness for free — but it *forces* the whole drift/migration
subsystem into existence (a 17k-line driver + a drift classifier + a CLI migrate command), and
makes every metadata change a potential DDL. Menata's single JSONB table makes metadata changes
free and the runtime tiny, at the cost of range scans/sorts on hot fields being GIN-unfriendly
above ~10⁵–10⁶ rows per Machine (Study 8's own scale finding, `benchmarks/004-*`). The
recommendation doc (§R4) argues the right answer for Menata is **expression indexes derived from
metadata (`CAP-X10`)**, not table-per-object — same query-plan benefit, none of the DDL surface.

## 7. Expression layer

`@objectstack/formula` is used everywhere a predicate or value is computed — this is the single
most reused mechanism in ObjectStack and the biggest thing Menata Runtime lacks:

| Surface | Schema | Example (from `examples/app-crm`) |
|---|---|---|
| Formula field | `Field.formula({ expression: cel\`…\` })` | `record.amount * record.probability / 100`; `daysBetween(today(), record.close_date)` |
| Validation rule condition | `validations[].condition: P\`…\`` | `record.discount_percent > 40` (script), cross-field `record.close_date < now() && record.stage != "closed_won"` |
| Action visibility | `action.visible` | `has(record.status) && record.status != "converted"` |
| Flow edge condition | `edge.condition` | `lead_record.status == 'converted'` |
| RLS policy | `using` / `check` (CEL over `record.*`, `current_user.*` incl. pre-resolved membership sets) | `record.owner_id == current_user.id` |
| Sharing rule criteria | `condition` | `record.amount > 1000000` |
| Approver resolution | `approvers[].type: 'expression'` over `current.* / trigger.* / vars.*` | `trigger.regional_director` |
| Dataset / dashboard filters | filter objects with date macros (`{current_quarter_start}`, `{1_years_ago}`, 36 tokens) | — |
| Default values | context tokens (`current_user`, `today`) and CEL defaults | — |

Engine facts: CEL via `cel-js`; a registered stdlib (`stdlib.ts`; `daysBetween` was added because
templates couldn't express "days remaining" — `docs/PLATFORM_GAPS_FROM_TEMPLATES.md` #7);
`cel-to-filter.ts` pushes simple predicates down to the driver as query filters with declared
`cel-pushdown-limits.ts`; `rls-predicate.ts` compiles RLS to SQL WHERE using pre-resolved
`IN (...)` membership sets rather than subqueries (ADR-0055/0057); a unified expression layer
decision (ADR-0032, ADR-0058) states the surface once for all consumers.

**Menata Runtime today:** no expression language by design. Conditions are fixed-operator triples
(`field / operator / value` — `required`, `equals`, `after`, `greater_than`, `on_or_after`, …,
`runtime-metadata-schema.md` §"Event Conditions"/"Constraints"), `computed` is one multiply
(`CAP-F14` ⚠️), dynamic values are a closed token set (`now`, `today`, `current_user`,
`CAP-A02`), aggregates are declared structures (`CAP-A14`, `CAP-C10`). This is a real
flexibility ceiling (see gap analysis §R1), and CEL has a first-class Go implementation
(`github.com/google/cel-go`, the reference implementation — used by Kubernetes admission policies),
so adopting the *language* would not mean adopting any ObjectStack code.

## 8. Extension model and code escape hatches

ObjectStack is "metadata + code where metadata runs out"; Menata Runtime is "metadata, or it is
not a capability yet". The escape hatches, with what governs each:

| Hatch | Where | Governance |
|---|---|---|
| **Hooks** — TypeScript `handler: async (ctx) => {…}` on `beforeFind/afterFind/before*/after*` (`data/hook.zod.ts:107`) | `examples/app-crm/src/hooks/opportunity.hook.ts` | Runs in-process with full engine access; `runAs`, priority, `onError: abort|log` |
| **Hook bodies** — script source stored *as metadata*, executed in a VM sandbox | `data/hook-body.zod.ts` | Must declare capability tokens (`api.read`, `api.write`, `api.transaction`, `crypto.uuid`, `log`); **no `http.fetch`** — outbound goes through Connectors so it stays auditable; CPU budget (ADR-0102) |
| **Functions** — named callables registered in `defineStack({ functions })` | called by flow `script` nodes and `defineJob({ handler })` | not runtime-creatable (`allowRuntimeCreate: false`) — a deploy-time artifact |
| **Connectors** — declarative REST/OpenAPI/MCP/Slack integration instances | `packages/connectors/*`, ADR-0097/0023/0024 | credentials via `secret` fields (ADR-0100); connector actions as flow nodes |
| **Plugins** — anything implementing `Plugin` | `packages/plugins/*` | signed artifacts, granted permissions per plugin (ADR-0025, `plugin-security*.zod.ts`) |
| **React pages** — JSX authored as metadata; parsed and rendered by an allowlisted component registry, never `eval`'d, unless promoted to a trusted tier | ADR-0080/0081/0082, `packages/sdui-parser` | component contract governance |
| `managedBy` on objects (`platform` / `config` / `system-data` / `engine-owned` / `append-only` / `better-auth`) | `data/object.zod.ts:1713` | who may write which tables — the closest thing to Menata's `CAP-O02` master-data designation + `CAP-R07` immutability |

**Assessment for Menata Runtime.** The hatches are where ObjectStack's "AI can hold the whole
app" claim quietly weakens: a hook handler is arbitrary TypeScript, invisible to the metadata
diff the four gates review. Menata Runtime's refusal to add a code hatch is a stronger position
*as long as the metadata grammar keeps growing to cover what cases need* — which is exactly what
the registry/case process exists to do. The one hatch worth studying is the **hook-body capability
token model** (a script may only do what it declared; undeclared = throws): if Menata ever admits
an expression layer (§7), the same "declare what the expression may touch, fail closed
otherwise" discipline is how to keep it from becoming a code hatch by another name.

## 9. Deployment, clustering, operations

| Concern | ObjectStack | Menata Runtime |
|---|---|---|
| Packaging | `os start` / `os serve <artifact>`; official Docker image; scaffolded projects are container-ready; Kubernetes/bare-Node guides (`content/docs/deployment/self-hosting.mdx`) | `go build` + `server-manager.sh`, no containers by constraint (shared VPS RAM/CPU — `app/ARCHITECTURE.md` §"Deployment target") |
| Tenancy | Three postures `single` / `group` / `isolated` (`OS_TENANCY_POSTURE`); Layer-0 organization wall AND-composed ahead of business RLS; walled postures require the *enterprise* `@objectstack/organizations` runtime to activate (ADR-0093/0095/0105) | Workspace = Postgres schema + RLS, always on (`CAP-X06`); `/{slug}/` routing (`CAP-X14`); multi-workspace identity (`CAP-O11`) — all open, all conformance-proven |
| Cluster | Four protocol primitives — PubSub, Lock, KV, Counter — in-memory by default, Redis/NATS/Postgres-LISTEN drivers; leader-elected jobs, cache invalidation by PubSub (at-most-once, hence ADR-0127's mandatory TTL bound on any cached authz answer) (`content/docs/kernel/cluster.mdx`) | Single process by design; in-memory Application Model; `CAP-X11` (LISTEN/NOTIFY eviction) registered ❌ and explicitly *not* a gap against the actual deployment target |
| Hardening | Security headers on by default, HSTS opt-in, token-bucket rate limit opt-in, CSRF at adapter, auth via better-auth (SSO/MFA/lockout in the enterprise tier, ADR-0069) (`docs/HARDENING.md`) | per-IP rate limiter (`cmd/server/ratelimit.go`), CSRF, `ClientIPFromHeader`, `govulncheck` in CI (37→0 reachable CVEs, `app/README.md` Phase 5) |
| Observability | `MetricsRegistry` (counter/histogram/gauge, Prometheus adapter recipe), `ErrorReporter` (Sentry recipe), `X-Request-Id`, W3C `traceparent` parser, `Server-Timing` (`docs/OBSERVABILITY.md`) | chi `RequestID` propagated into every `record_events` row (`CAP-I04` ⚠️ half); structured access log; no metrics endpoint |
| Jobs/queues | `service-job` (cron/interval/one-off, retry policy, per-attempt timeout, run history), `service-queue` (memory, BullMQ) | `CAP-E02` schedule events + `CAP-W06` `action_outbox` (retrying async actions) |
| Secrets | `secret` field type encrypted via `ICryptoProvider`, ciphertext in `sys_secret`, fail-closed without a provider (ADR-0100) | plain env vars, decided and named (`app/README.md` Phase 3) |

## 10. Engineering discipline worth noting

- **`AGENTS.md` (1,058 lines) is the source of truth for contributors, human or AI**, with
  binding "Prime Directives": claim the GitHub issue before writing code, one worktree per task,
  never `git stash` (shared stash across worktrees), never edit release notes in a code PR. The
  repo is visibly developed by many concurrent AI-agent sessions (13,044 commits, 673 open issues,
  50 stars as of 2026-09-07) — a different operating model from this repo's single owner + one
  session at a time.
- **Ratchets in CI**: type-check coverage/debt ledgers (shrink-only), cross-package test-input
  gate, docs-link liveness (`lychee.toml`), published-list mirrors. Same idea as `app/`'s
  `scripts/check-quality-gates.sh` (handler-LOC, `gocyclo`, error-leak scan) and the 225-test
  conformance ratchet.
- **Enforce-or-remove (ADR-0049) and no-silently-inert-metadata (ADR-0078)**: a metadata key with
  no runtime reader is deleted, and its former spelling is kept as a *tombstone* that rejects with
  an upgrade prescription (`UNKNOWN_KEY_GUIDANCE` in `object.zod.ts`, `PERMISSION_SET_KEY_ALIASES`).
  This is the same principle as `runtime-metadata-schema.md`'s "Load-Time Contract — What's
  Enforced, What Silently No-ops" section, taken one step further: Menata *documents* the silent
  no-ops; ObjectStack *makes them load errors*. Worth copying (gap analysis §"discipline").
- **Conformance ledgers (ADR-0060)**: per-surface proof ledgers (`search-conformance`,
  aggregation lockstep, value round-trip) — the closest analogue to `capability-registry.md`'s
  Proof column, but per protocol surface rather than per capability.

## 11. Side-by-side summary

| Dimension | ObjectStack | Menata Runtime (`app/`) | Verdict for Menata |
|---|---|---|---|
| Authoring medium | TypeScript (Zod-checked), compiled | YAML → SQL seed rows (Authoring Layer intended) | keep — but see gap §"validation gates" |
| Behavior owner | runtime; client is a consumer | runtime; client has no framework at all | keep |
| Extension | runtime plugins + code hatches | compile-time registry seams, metadata only | keep |
| Expression | CEL everywhere | fixed operators | **adopt the language** (cel-go), not the code |
| Workflow | DAG flow engine + state-machine rule | emergent Events + Process Overlay | keep (Study 20); borrow specific nodes as actions |
| Storage | table-per-object + drift subsystem | JSONB + GIN | keep; build `CAP-X10` |
| Permissions | 4-layer (sets, scope depth, sharing, RLS/FLS) over BU tree + positions | role/ownership/SoD/CRUD/field visibility, groups | **adopt scope depth + org tree** |
| Analytics | semantic dataset → cube runtime → charts/reports | per-view aggregates, count tiles, no charts | **adopt dataset layer + chart widget** |
| AI surface | MCP server, agents, RAG | none | **adopt a governed MCP adapter** |
| Ops | cluster primitives, metrics, Docker | single binary, rate limit, CI | keep; add request-id/metrics later |
| Proof | tests + ledgers; spec ahead of enforcement | conformance-ratcheted registry | keep — this is the stronger model |
