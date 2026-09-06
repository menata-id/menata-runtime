// Package db provides Connect, establishing and pinging a pgxpool.Pool
// connection to Postgres via DATABASE_URL.
//
// Graduated as-is from prototype/go/internal/db — a leaf package, 0
// internal/ dependencies (Study 34 §7 Method 2). No architectural change
// intended.
package db
