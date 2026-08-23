# Inventory / Warehouse Benchmark

> Supporting benchmark for Case 5 (`../case-portfolio.md`) — external map before the case is written,
> per the roadmap's dual-track discovery method ("the map before the territory").
>
> Grounded in the Workflow Patterns Initiative-equivalent for this vertical: the six-stage WMS
> process flow (receiving → putaway → storage → picking → packing → shipping) and the APICS/ASCM
> Body of Knowledge for inventory control (FIFO/FEFO, cycle counting, lot/serial tracking,
> reservation/allocation, perpetual vs periodic counting).
>
> Status: v0.1 | Created: 2026-07-10

**Why inventory:** after accounting (Study 6), inventory is the second hardest widely-standardized
vertical — it shares accounting's numeric-integrity demands (a balance that must never silently
drift) but adds a dimension accounting doesn't have: **physical reality** (a unit of measure, a
location, a lot, an expiry date all have to reconcile with a number in a database). If Case 5 only
proves the simplest slice of this, the registry's `Quantity` prediction (Study 15) is under-tested.
This document maps the full vertical first, so Case 5's scope is a **deliberate** subset, not an
accidental one.

---

# World-class reference: the six-stage WMS flow

| Stage | What happens | Menata-relevant pattern |
|-------|--------------|--------------------------|
| Receiving | Incoming goods checked against expected quantity, condition, barcode | Cross-record match (expected vs actual) — same shape as Case 8's webhook reconciliation |
| Putaway | Stock assigned to a bin/location by velocity, cube, weight rules | Location as a dimension on stock, not just on the Item — **not modeled today** |
| Storage | Stock sits at a location; multiple locations may hold the same Item | Multi-location balance (one Item, many location-scoped balances) |
| Picking | Discrete / batch / zone / cluster strategies select stock to fulfill a demand | Reservation/allocation — stock "spoken for" before it physically moves |
| Packing | Picked stock consolidated for shipment | Out of Menata's metadata scope — physical/logistics execution, not a business object |
| Shipping | Stock leaves the warehouse; ownership/location changes | A Stock Movement variant Case 5 already covers |

# APICS/ASCM inventory-control concepts

| Concept | What it requires | Nature |
|---------|-------------------|--------|
| Perpetual vs periodic counting | Perpetual: every movement updates the balance in real time. Periodic: balance is only true after a scheduled physical count | Case 5 assumes perpetual (movement-driven balance) — periodic reconciliation is a distinct pattern |
| FIFO / FEFO | FIFO: oldest receipt picked first. FEFO: nearest-expiry picked first — requires lot tracking + expiry date capture at receiving | Needs lot/batch as its own entity with identity + expiry — a `reference`-sugar candidate, same shape as `money`'s Currency (Study 15) |
| Lot / serial tracking | Each lot or unit has traceable identity across its lifecycle (receipt → movement → sale/consumption) | Genuinely new Machine, not a field — an Item's lot is closer to `Currency` (CAP-O02 master-data candidate) than to a primitive |
| Reservation / allocation | Stock is "soft-committed" to a demand before it physically moves — the balance must show available vs reserved | A second computed quantity (`Available = On Hand - Reserved`) — CAP-F14 variant, not yet in Case 5's scope |
| Quality inspection hold | Received stock is not available until it clears inspection — an extra state between "received" and "in stock" | State-conditional availability, i.e. CAP-E06 (already registered, Prio 2) applied to a stock-status state machine |
| Multi-location / bin-level balance | The same Item has independent balances per warehouse/bin | A missing dimension on Stock Ledger — parallel to CAP-X09 (org-unit scoping) but at the physical-location grain |
| Valuation (standard cost / moving average / FIFO costing) | Every stock movement carries not just a quantity but a costed value, computed per costing method | Composes CAP-F17 (multi-currency money) + CAP-F14 (computed) — a costing *engine* problem, same boundary Study 6 already drew around posting derivation ("domain engines as pluggable executors beneath declarative metadata") |
| Cycle counting | Rolling, targeted physical counts reconcile system balance to physical reality without a full shutdown | Produces an "Adjustment" movement type — composable from what Case 5 already models, not a new concept |

---

# Case 5 scope decision — covered vs deliberately out of scope

Per the Case Portfolio rule "document the surprises" and the registry rule "silence is not a
decision," every concept above is marked, not silently dropped:

| Concept | Case 5 status | Reason |
|---------|---------------|--------|
| Perpetual balance via movement | **Covered** | Core of Case 5 — Stock Movement → Stock Ledger → Item.Stock On Hand |
| Unit-of-measure conversion (Tier 1 and Tier 2) | **Covered** | Direct proof target for Study 15's Quantity tiering prediction — see declaration below |
| Multi-record write atomicity | **Covered** | Movement confirmation, ledger append, and balance recompute must commit as one unit — the cluster the original Case 5 declaration already flagged as untargeted |
| Negative-stock prevention | **Covered** | Cross-record constraint (Stock Movement vs Item's current balance) |
| Cycle counting / adjustment | **Covered (minimal)** | Modeled as a third Movement Type (`Adjustment`) — reuses the same ledger mechanism, no new capability |
| Lot / serial tracking + FIFO/FEFO | **Out of scope** | Needs a new master-data Machine (lot/batch as an entity with its own identity and expiry) — same shape as CAP-O02, but no case has evidenced it yet. Candidate for a future case, not this one |
| Reservation / allocation (soft commit) | **Out of scope** | Needs a second computed quantity (`Available` vs `Reserved`) on top of `Stock On Hand` — real, but doubles the case's dominant cluster (calculation) and dilutes the single-cluster rule. Candidate to fold into a future refinement once CAP-F14 is implemented |
| Multi-location / bin-level balance | **Out of scope** | Adds a location dimension to every balance — parallel to CAP-X09 (org-unit scoping), which is already registered from Study 5 and awaiting implementation; revisit together rather than duplicating the finding |
| Quality inspection hold | **Out of scope** | Already fully explained by CAP-E06 (state-conditional event availability, Prio 2, registered since Study 1) applied to a stock-status machine — no new capability, and forcing it into Case 5 would just be a state-machine restatement of an already-registered gap |
| Costed valuation (standard/moving-average/FIFO costing) | **Out of scope** | Same architectural boundary Study 6 drew for posting derivation — a costing engine is a pluggable executor beneath declarative metadata, not a metadata concept itself. Belongs with Case 9 (Accounting) or a dedicated costing study, not Case 5 |
| Packing / physical logistics execution | **Out of scope by design** | Physical execution has no business-object shape — nothing to model as metadata |

**Net effect:** Case 5 stays a single dominant cluster (calculation + multi-record transaction) while
proving the Quantity tiering framework in full (Tier 1 and Tier 2), and explicitly queues four
follow-on case candidates (lot/serial + FEFO, reservation/allocation, multi-location balance, costed
valuation) instead of silently declaring the vertical "done."

> **Correction (2026-08-23):** the two `CAP-X09` references above (line 43's "parallel... at the
> physical-location grain" and this table's "already registered from Study 5 and awaiting
> implementation; revisit together") assumed CAP-X09 would land as one buildable capability.
> Design review closed CAP-X09 without building it — see `capability-registry.md`'s CAP-X09 row
> and `roadmap.md`'s 2026-08-23 Track E update — it dissolved into composition of other
> capabilities (CAP-F13 for the org-context data model, CAP-O07+CAP-P02 for permission scoping,
> a session cookie + CAP-V09 filter for selectors, CAP-A02 for timezone). "Revisit together" no
> longer has a single row to revisit against: if a future case needs multi-location/bin-level
> balance, evaluate it on its own admission evidence rather than waiting on CAP-X09.

---

# New registry candidates surfaced by this benchmark

| Candidate | Description | Status |
|-----------|--------------|--------|
| **CAP-F19** | `quantity` field with tiered unit-of-measure conversion (Tier 1: flat factor pair · Tier 2: child table · Tier 3: history-tracked Machine) — Study 15's prediction, now given case evidence | Proposed — see Case 5 declaration |
| **CAP-X12** | Cross-record write atomicity (an event's actions across two-or-more Machines — e.g. append a ledger entry and recompute a rollup — must commit as a single unit or not at all) | Proposed — first named in original Case 5 declaration (`multi-record atomicity`), still unregistered until now |

Both pass the dual-evidence admission bar from `capability-lifecycle.md`: CAP-F19 has Study 15
(benchmark) + Case 5 (case); CAP-X12 has this benchmark (worked example: receiving reconciliation,
picking allocation) + Case 5 (case) — see `capability-registry.md` for the registered rows.
