package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"menata.id/runtime/internal/model"
)

// CAP-X07: an auto-generated, read/create JSON API per Machine -- the
// *outbound* surface (this runtime exposing its own Machines' records as
// JSON), distinct from CAP-E04's webhooks (an *inbound* surface, arbitrary
// external payload shape, secret-token authenticated, no session at all).
// Auth here is the SAME session cookie + CSRF discipline every HTML route
// already uses (see cmd/server/main.go's csrfProtect, extended to also
// accept X-CSRF-Token as a header for exactly this JSON-body case) --
// deliberately not a separate API-key mechanism, which is its own
// capability this prototype doesn't have yet. Every response is
// permission-trimmed the same way List/Detail already are: CAP-P05's
// Guard.CanRead/CanCreate gates access to the Machine at all, and CAP-P06's
// hidden_fields strips restricted fields out of the JSON the same way it
// strips them out of rendered HTML.
//
// Deliberately out of scope for this first pass (name the gap, don't fake
// it): CAP-F16 child-table rows, CAP-V12 wizard steps, and event triggering
// are not reachable through this API -- only plain Create/Read on a
// Machine's own fields. A future pass can extend this the same way HTML
// Create/Update already handle those, once a real case asks for it here.

func apiJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// stripHidden returns a copy of data with every hidden field id removed --
// never mutates the caller's own map (records.Data is shared with whatever
// else is still holding a reference to the same *store.Record in this
// request).
func stripHidden(data map[string]any, hidden map[string]bool) map[string]any {
	if len(hidden) == 0 {
		return data
	}
	out := make(map[string]any, len(data))
	for k, v := range data {
		if !hidden[k] {
			out[k] = v
		}
	}
	return out
}

type apiRecord struct {
	ID   string         `json:"id"`
	Data map[string]any `json:"data"`
}

// APIList handles GET /api/{machineID}.
func (h *Handler) APIList(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanRead(machine, role) {
		h.logPermissionDenied(r.Context(), "api_list", machineID, "", role, h.identity(r))
		apiJSON(w, http.StatusForbidden, map[string]string{"error": "not permitted"})
		return
	}
	records, err := h.records.List(r.Context(), machineID, "", "")
	if err != nil {
		apiJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list records"})
		return
	}
	hidden := h.hiddenFields(machine, role)
	out := make([]apiRecord, len(records))
	for i, rec := range records {
		out[i] = apiRecord{ID: rec.ID, Data: stripHidden(rec.Data, hidden)}
	}
	apiJSON(w, http.StatusOK, out)
}

// APIGet handles GET /api/{machineID}/{recordID}.
func (h *Handler) APIGet(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanRead(machine, role) {
		h.logPermissionDenied(r.Context(), "api_get", machineID, "", role, h.identity(r))
		apiJSON(w, http.StatusForbidden, map[string]string{"error": "not permitted"})
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil || rec.MachineID != machineID {
		http.NotFound(w, r)
		return
	}
	hidden := h.hiddenFields(machine, role)
	apiJSON(w, http.StatusOK, apiRecord{ID: rec.ID, Data: stripHidden(rec.Data, hidden)})
}

// APICreate handles POST /api/{machineID} -- same validation set (CAP-C01..
// C12 Constraints, CAP-F13/CAP-F05 reference/user integrity, CAP-C12
// uniqueness) as the HTML Create path, minus CAP-F16 child rows and CAP-V12
// wizard steps (see this file's own top-of-file scope note).
func (h *Handler) APICreate(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanCreate(machine, role) {
		h.logPermissionDenied(r.Context(), "api_create", machineID, "", role, h.identity(r))
		apiJSON(w, http.StatusForbidden, map[string]string{"error": "not permitted"})
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		apiJSON(w, http.StatusBadRequest, map[string]string{"error": "expected application/json"})
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	data := make(map[string]any)
	for _, f := range machine.Fields {
		if v, present := body[f.ID]; present {
			data[f.ID] = v
		} else if f.Type == model.FieldTypeValueList && len(f.Options.Values) > 0 {
			data[f.ID] = f.Options.Values[0]
		}
	}

	var violations []string
	if !h.inScratchState(machine, data) {
		violations = h.engine.Violations(machine, data)
	}
	refViolations, err := h.referenceViolations(r.Context(), machine, data)
	if err != nil {
		apiJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to validate references"})
		return
	}
	violations = append(violations, refViolations...)
	userViolations, err := h.userReferenceViolations(r.Context(), machine, data)
	if err != nil {
		apiJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to validate references"})
		return
	}
	violations = append(violations, userViolations...)
	uniqueViolations, err := h.uniquenessViolations(r.Context(), machine, data, "")
	if err != nil {
		apiJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to validate uniqueness"})
		return
	}
	violations = append(violations, uniqueViolations...)

	if len(violations) > 0 {
		h.logRuleViolation(r.Context(), "api_create", machineID, "", role, h.identity(r), strings.Join(violations, "; "))
		apiJSON(w, http.StatusBadRequest, map[string]any{"violations": violations})
		return
	}

	rec, err := h.records.Create(r.Context(), machineID, workspaceID, data)
	if err != nil {
		apiJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create record"})
		return
	}
	hidden := h.hiddenFields(machine, role)
	apiJSON(w, http.StatusCreated, apiRecord{ID: rec.ID, Data: stripHidden(rec.Data, hidden)})
}

// APIExportApplication handles GET /apps/{applicationID}/export (CAP-X08) --
// a portable dump of one Application's full metadata tree (every Machine's
// Fields/Events/Constraints/Permissions/Views/Config/Subscriptions), read
// straight from the already-loaded in-memory Application Model, not the DB
// -- the same structure Router/Renderer/Executor/Engine already operate on,
// so this is provably what the runtime actually interprets, not a second,
// possibly-drifted representation. Admin-only: this reveals the full
// internal shape of an Application (every Field id, every Constraint
// expression), not something every role should be able to pull.
//
// Import is deliberately NOT built in this pass -- a generic importer needs
// the same load-time validation rigor CAP-X05 already applies (dangling
// references, bad operators, owner_field type-checking) run transactionally
// against a package that could otherwise corrupt the shared production
// database if malformed, and a real reload story (today: apply as SQL,
// restart the server, same as every other metadata change in this
// codebase) -- worth its own dedicated pass, not a rushed addition here.
// capability-registry.md's CAP-X08 row names this scope explicitly.
func (h *Handler) APIExportApplication(w http.ResponseWriter, r *http.Request) {
	if !h.isWorkspaceAdmin(r) {
		apiJSON(w, http.StatusForbidden, map[string]string{"error": "not permitted"})
		return
	}
	applicationID := chi.URLParam(r, "applicationID")
	app, ok := h.interp.Get().GetApplication(applicationID)
	if !ok || app.WorkspaceID != h.workspace(r) {
		http.NotFound(w, r)
		return
	}
	apiJSON(w, http.StatusOK, app)
}
