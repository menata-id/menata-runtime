package router

import (
	"github.com/go-chi/chi/v5"

	"menata.id/runtime/internal/handler"
)

// Mount registers all routes. Routes are derived from the Application Model
// (loaded from Runtime Metadata), not hardcoded.
//
// URL scheme:
//   /                                        home — list of machines
//   /login                                   role selection (prototype auth)
//   /{machineID}                             default list view
//   /{machineID}/new                         new record form
//   /{machineID}/{recordID}                  record detail
//   POST /{machineID}                        create record
//   POST /{machineID}/{recordID}/events/{eventID}  trigger event
func Mount(r chi.Router, h *handler.Handler) {
	r.Get("/", h.Home)

	r.Get("/login", h.LoginForm)
	r.Post("/login", h.Login)

	r.Get("/{machineID}", h.List)
	r.Get("/{machineID}/new", h.NewForm)
	r.Post("/{machineID}", h.Create)
	r.Get("/{machineID}/{recordID}", h.Detail)
	r.Post("/{machineID}/{recordID}/events/{eventID}", h.TriggerEvent)
}
