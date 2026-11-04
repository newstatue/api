package storage

import (
	"bytes"
	"testing"
)

func TestNewBucket(t *testing.T) {
	ctx := t.Context()

	t.Setenv("AWS_ACCESS_KEY_ID", "dev")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "dev")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	bucket, err := NewBucket(ctx, "s3://my-bucket?endpoint=http%3A%2F%2Flocalhost%3A8333&region=us-east-1&use_path_style=true", "http://localhost:8333")
	if err != nil {
		t.Fatal(err)
	}
	defer bucket.Close()

	err = bucket.WriteAll(ctx, "hello.txt", []byte("Hello SeaweedFS"), nil)
	if err != nil {
		t.Fatal(err)
	}

	reader, err := bucket.NewReader(ctx, "hello.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := reader.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}

	err = bucket.Delete(ctx, "hello.txt")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("文件内容：%s", buf.String())
}
