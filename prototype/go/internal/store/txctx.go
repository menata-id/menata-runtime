package store

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// querier abstracts over *pgxpool.Pool and pgx.Tx -- both already satisfy
// this shape, so RecordStore/NotificationStore work unchanged whether given
// the raw pool (metadata.Loader's boot-time LoadAll, trusted, sees every
// workspace) or a request-scoped transaction (every handler-driven query,
// CAP-X06 -- see cmd/server/main.go's request-scoped transaction
// middleware, which sets app.workspace_id via SET LOCAL before handing the
// tx to the handler through WithTx).
type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txCtxKey struct{}

// WithTx attaches a request-scoped transaction to ctx -- every store method
// downstream uses it instead of the raw pool, so RLS's
// current_setting('app.workspace_id', ...) (set on this same transaction,
// SET LOCAL) actually applies to the queries they run.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

// dbFromContext returns the request-scoped transaction if one was attached
// via WithTx, falling back to pool otherwise. The fallback exists for
// trusted, non-request code paths (metadata.Loader's boot-time LoadAll) --
// every HTTP-request-driven call always has a tx by the time it reaches a
// store method, via cmd/server/main.go's middleware.
func dbFromContext(ctx context.Context, pool querier) querier {
	if tx, ok := ctx.Value(txCtxKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}
