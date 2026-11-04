package storage

import (
	"context"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/s3blob"
)

func NewBucket(ctx context.Context, url string) (*blob.Bucket, error) {
	return blob.OpenBucket(ctx, url)
}
