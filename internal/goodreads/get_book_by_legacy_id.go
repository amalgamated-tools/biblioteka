package goodreads

import (
	"context"
)

func (c *Client) GetBookByLegacyID(ctx context.Context, legacyID int64) (*BookResult, error) {
	resp, err := GetBookByLegacyId(ctx, c.client, legacyID)
	if err != nil {
		return nil, err
	}

	return loadBookResult(ctx, resp.GetBookByLegacyId.Work.BestBook)
}
