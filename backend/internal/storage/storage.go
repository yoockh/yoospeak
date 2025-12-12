package storage

import (
	"context"
	"io"
	"time"
)

type Uploader interface {
	Upload(ctx context.Context, objectName string, contentType string, r io.Reader) (storedPath string, err error)
}

type Signer interface {
	SignedGetURL(ctx context.Context, objectName string, ttl time.Duration) (string, error)
}
