package interpreter

import "sync/atomic"

// Store (CAP-X04, metadata live reload — Option A of docs/decisions/
// 002-metadata-loading.md) holds the currently-active *Interpreter behind an
// atomic pointer, so a fresh Interpreter built from a re-run of
// metadata.Loader.LoadAll can be swapped in without ever exposing a partially
// built model to an in-flight request. A single atomic.Pointer read/write —
// no lock, safe for arbitrarily many concurrent readers (every Handler
// method) and the rare admin-triggered writer (Reload).
//
// Callers must call Get() fresh at the point of use, not once and cache the
// result across a request or a scheduler tick — see handler.Reload's own doc
// comment and cmd/server/main.go's sessionAuth/runScheduler for why a stale
// capture would silently defeat the whole point of this type.
type Store struct {
	p atomic.Pointer[Interpreter]
}

// NewStore wraps an already-built Interpreter (the boot-time LoadAll result)
// as the initial value.
func NewStore(i *Interpreter) *Store {
	s := &Store{}
	s.p.Store(i)
	return s
}

// Get returns the currently-active Interpreter.
func (s *Store) Get() *Interpreter {
	return s.p.Load()
}

// Swap atomically replaces the active Interpreter — the one moment a reload
// actually takes effect. Never called with a partially built Interpreter:
// the caller (handler.Reload) only calls this after metadata.Loader.LoadAll
// has already succeeded and interpreter.New has already finished building
// the new model in full.
func (s *Store) Swap(i *Interpreter) {
	s.p.Store(i)
}
