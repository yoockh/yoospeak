package storage

import (
	"context"
	"io"
)

type Uploader interface {
	Upload(ctx context.Context, objectName string, contentType string, r io.Reader) (publicURL string, err error)
}
