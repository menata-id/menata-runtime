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
//   /                                        home — list of machines
//   /login                                   role selection (prototype auth)
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
	r.Get("/", h.Home)

	r.Get("/login", h.LoginForm)
	r.Post("/login", h.Login)

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
