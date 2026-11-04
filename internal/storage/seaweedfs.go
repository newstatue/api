package storage

import (
	"context"
	"net/url"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/s3blob"
)

type Bucket struct {
	*blob.Bucket
	publicURL url.URL
}

// NewBucket 使用内部 S3 地址读写对象，使用公网地址生成永久访问链接。
// 访问地址由 publicEndpoint 配置，不区分开发和生产环境。
func NewBucket(ctx context.Context, bucketURL, publicEndpoint string) (*Bucket, error) {
	u, err := url.Parse(bucketURL)
	if err != nil {
		return nil, err
	}
	publicURL, err := url.Parse(publicEndpoint)
	if err != nil {
		return nil, err
	}
	publicURL.Path = "/" + u.Host + "/"

	bucket, err := blob.OpenBucket(ctx, bucketURL)
	if err != nil {
		return nil, err
	}
	return &Bucket{Bucket: bucket, publicURL: *publicURL}, nil
}

// GetURL 返回对象的公开访问地址
func (b *Bucket) GetURL(key *string) *string {
	if key == nil || *key == "" {
		return nil
	}
	u := b.publicURL
	u.Path += *key
	objectURL := u.String()
	return &objectURL
}
