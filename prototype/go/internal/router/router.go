package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"menata.id/runtime/internal/handler"
)

// Mount registers all routes. Routes are derived from the Application Model
// (loaded from Runtime Metadata), not hardcoded.
//
// URL scheme:
//
//	/                                        workspace home — list of applications (CAP-O03)
//	/apps/{applicationID}                    one application's own machines
//	/login                                   email + password login (CAP-X02)
//	/admin/users                             workspace user/role management (CAP-O01, Admin-only)
//	POST /admin/reload                       re-load metadata without restart, Admin-only (CAP-X04)
//	/notifications                           in-app notification inbox (CAP-A10)
//	/apps/{applicationID}/export             Application metadata export, Admin-only (CAP-X08)
//	POST /apps/import                        Application metadata import, Admin-only (CAP-X08)
//	GET  /api/{machineID}                    JSON list (CAP-X07)
//	GET  /api/{machineID}/{recordID}         JSON detail (CAP-X07)
//	POST /api/{machineID}                    JSON create (CAP-X07)
//	/{machineID}                             default list view
//	/{machineID}/new                         new record form
//	/{machineID}/{recordID}                  record detail
//	/{machineID}/{recordID}/edit             edit record form
//	POST /{machineID}                        create record
//	POST /{machineID}/{recordID}              update record
//	POST /{machineID}/{recordID}/events/{eventID}  trigger event
//	POST /notifications/{id}/read             mark one notification read
//	POST /webhooks/{machineID}/{recordID}/{eventID}  CAP-E04, secret-token authenticated, no session
func Mount(r chi.Router, h *handler.Handler) {
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})
	r.Get("/", h.Apps)
	r.Get("/apps/{applicationID}", h.AppMachines)
	r.Get("/apps/{applicationID}/export", h.APIExportApplication) // CAP-X08, Admin-only
	r.Post("/apps/import", h.APIImportApplication)                // CAP-X08, Admin-only

	r.Get("/login", h.LoginForm)
	r.Post("/login", h.Login)
	r.Post("/logout", h.Logout)

	r.Get("/admin/users", h.AdminUsers)
	r.Post("/admin/users/{userID}", h.AdminUpdateUser)
	r.Post("/admin/reload", h.Reload) // CAP-X04, Admin-only

	r.Post("/admin/groups", h.AdminCreateGroup)                       // CAP-O07, Admin-only
	r.Get("/admin/groups/{groupID}", h.AdminGroupDetail)              // CAP-O07, Admin-only
	r.Post("/admin/groups/{groupID}/members", h.AdminSetGroupMembers) // CAP-O07, Admin-only
	r.Post("/admin/groups/{groupID}/roles", h.AdminSetGroupRoles)     // CAP-O07, Admin-only

	r.Get("/notifications", h.Notifications)
	r.Post("/notifications/{id}/read", h.MarkNotificationRead)
	r.Post("/notifications/preference", h.SetNotificationPreference) // CAP-O05

	r.Get("/search", h.Search) // CAP-O04

	r.Get("/files/{key}", h.ServeFile) // CAP-F06 -- unguessable key IS the access control, see upload.go

	// CAP-X07: auto-generated outbound JSON API, one route family per
	// Machine, same session+CSRF auth and permission trimming as the HTML
	// routes below -- see internal/handler/api.go's own top-of-file note
	// for the deliberate first-pass scope (no child rows, no wizard steps,
	// no event triggering yet).
	r.Get("/api/{machineID}", h.APIList)
	r.Get("/api/{machineID}/{recordID}", h.APIGet)
	r.Post("/api/{machineID}", h.APICreate)

	r.Get("/{machineID}", h.List)
	r.Get("/{machineID}/new", h.NewForm)
	r.Post("/{machineID}", h.Create)
	r.Get("/{machineID}/report", h.Report)              // CAP-V13
	r.Get("/{machineID}/calendar", h.Calendar)          // CAP-V07
	r.Get("/{machineID}/timeline", h.Timeline)          // CAP-V07
	r.Get("/{machineID}/dashboard", h.Dashboard)        // CAP-V10
	r.Get("/{machineID}/process-map", h.ProcessMap)     // CAP-W05
	r.Get("/{machineID}/process-lift", h.LiftProcess)   // CAP-W05 backward direction (B6), Admin-only
	r.Get("/{machineID}/field-options", h.FieldOptions) // CAP-V16 typeahead search fragment
	r.Get("/{machineID}/board", h.Board)                // CAP-V14 Tier 2 kanban board
	r.Get("/{machineID}/{recordID}", h.Detail)
	r.Get("/{machineID}/{recordID}/edit", h.EditForm)
	r.Get("/{machineID}/{recordID}/document", h.Document) // CAP-F21
	r.Post("/{machineID}/{recordID}", h.Update)
	r.Post("/{machineID}/{recordID}/events/{eventID}", h.TriggerEvent)
	r.Post("/{machineID}/{recordID}/move/{direction}", h.MoveRecord) // CAP-V14
	r.Post("/{machineID}/{recordID}/board-move", h.BoardMove)        // CAP-V14 Tier 2
	r.Get("/{machineID}/{recordID}/place", h.CoordPlace)             // CAP-V21
	r.Post("/{machineID}/{recordID}/place", h.SetCoordPlace)         // CAP-V21
	r.Post("/{machineID}/{recordID}/archive", h.Archive)             // CAP-R03
	r.Post("/{machineID}/{recordID}/restore", h.Restore)             // CAP-R03
	r.Get("/{machineID}/export.csv", h.ExportCSV)                    // CAP-R06
	r.Get("/{machineID}/import", h.ImportCSVForm)                    // CAP-R06
	r.Post("/{machineID}/import", h.ImportCSV)                       // CAP-R06

	// CAP-E04: an external system triggers an event directly, authenticated
	// by a per-Machine shared secret (Machine.Config's "webhook_secret"),
	// not a user session -- exempted from sessionAuth/csrfProtect the same
	// way /login is (cmd/server/main.go's isPublicPath/csrfProtect).
	r.Post("/webhooks/{machineID}/{recordID}/{eventID}", h.Webhook)
}
