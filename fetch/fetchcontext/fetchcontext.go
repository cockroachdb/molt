package fetchcontext

import (
	"context"
	"time"

	"github.com/cockroachdb/cockroachdb-parser/pkg/util/uuid"
)

type fetchKey struct{}

type FetchContextData struct {
	RunID     uuid.UUID
	StartedAt time.Time
}

func ContextWithFetchData(ctx context.Context, req FetchContextData) context.Context {
	return context.WithValue(ctx, fetchKey{}, req)
}

func GetFetchContextData(ctx context.Context) FetchContextData {
	x := ctx.Value(fetchKey{})
	if x != nil {
		return x.(FetchContextData)
	}
	return FetchContextData{}
}
