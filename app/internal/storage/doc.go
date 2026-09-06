// Package storage is a NEW package — it did not exist in prototype/go.
// It abstracts uploaded-file bytes behind a small interface (conceptually
// Put/Get/Delete, keyed by the same content-hashed token
// prototype/go/internal/handler/upload.go already generates) so the actual
// backing store can be swapped without touching any caller.
//
// Day one implementation is a local-disk backend, behavior-identical to
// prototype/go's own uploads/ directory (CAP-F06) — this package changes
// nothing about how files are served or compressed, only where that logic
// lives. An S3-compatible backend can be added later purely inside this
// package, addressing Study 34 §5's file-storage gap (multi-instance/
// multi-host deployments needing shared storage) without a rewrite of
// anything that calls it. See ARCHITECTURE.md's "New package: internal/
// storage" section for the full rationale.
package storage
