package pagination

import (
	"context"

	"github.com/uptrace/bun"
)

type Params struct {
	Page  int64 `query:"page" default:"1" minimum:"1"`
	Limit int64 `query:"limit" default:"20" minimum:"1" maximum:"100"`
}

type Result[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
	Page  int64 `json:"page"`
	Limit int64 `json:"limit"`
}

func Query[T any](ctx context.Context, query *bun.SelectQuery, params Params) (*Result[T], error) {
	items := make([]T, 0)

	total, err := query.
		Model(&items).
		Limit(params.Limit).
		Offset((params.Page - 1) * params.Limit).
		ScanAndCount(ctx)

	if err != nil {
		return nil, err
	}

	return &Result[T]{
		Items: items,
		Total: total,
		Page:  params.Page,
		Limit: params.Limit,
	}, nil
}
