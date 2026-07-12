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
//   /                                        workspace home — list of applications (CAP-O03)
//   /apps/{applicationID}                    one application's own machines
//   /login                                   email + password login (CAP-X02)
//   /admin/users                             workspace user/role management (CAP-O01, Admin-only)
//   /notifications                           in-app notification inbox (CAP-A10)
//   /{machineID}                             default list view
//   /{machineID}/new                         new record form
//   /{machineID}/{recordID}                  record detail
//   /{machineID}/{recordID}/edit             edit record form
//   POST /{machineID}                        create record
//   POST /{machineID}/{recordID}              update record
//   POST /{machineID}/{recordID}/events/{eventID}  trigger event
//   POST /notifications/{id}/read             mark one notification read
func Mount(r chi.Router, h *handler.Handler) {
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})
	r.Get("/", h.Apps)
	r.Get("/apps/{applicationID}", h.AppMachines)

	r.Get("/login", h.LoginForm)
	r.Post("/login", h.Login)
	r.Post("/logout", h.Logout)

	r.Get("/admin/users", h.AdminUsers)
	r.Post("/admin/users/{userID}", h.AdminUpdateUser)

	r.Get("/notifications", h.Notifications)
	r.Post("/notifications/{id}/read", h.MarkNotificationRead)

	r.Get("/{machineID}", h.List)
	r.Get("/{machineID}/new", h.NewForm)
	r.Post("/{machineID}", h.Create)
	r.Get("/{machineID}/{recordID}", h.Detail)
	r.Get("/{machineID}/{recordID}/edit", h.EditForm)
	r.Post("/{machineID}/{recordID}", h.Update)
	r.Post("/{machineID}/{recordID}/events/{eventID}", h.TriggerEvent)
}
