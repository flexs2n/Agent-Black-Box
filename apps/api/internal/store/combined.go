package store

import (
	"context"

	"github.com/blackbox-agentdiff/api/internal/model"
)

type CombinedStore struct {
	Store
	clickHouse *ClickHouseSpanStore
}

func NewCombinedStore(primary Store, clickHouse *ClickHouseSpanStore) *CombinedStore {
	return &CombinedStore{Store: primary, clickHouse: clickHouse}
}

func (c *CombinedStore) SpanPutBatch(ctx context.Context, spans []model.Span) error {
	if c.clickHouse != nil {
		return c.clickHouse.SpanPutBatch(ctx, spans)
	}
	return c.Store.SpanPutBatch(ctx, spans)
}

func (c *CombinedStore) SpanList(ctx context.Context, traceID string) ([]model.Span, error) {
	if c.clickHouse != nil {
		return c.clickHouse.SpanList(ctx, traceID)
	}
	return c.Store.SpanList(ctx, traceID)
}
