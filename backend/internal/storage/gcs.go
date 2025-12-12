package storage

import (
	"context"
	"fmt"
	"io"

	gcs "cloud.google.com/go/storage"
)

type GCSUploader struct {
	client *gcs.Client
	bucket string
}

func NewGCSUploader(ctx context.Context, bucket string) (*GCSUploader, error) {
	c, err := gcs.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &GCSUploader{client: c, bucket: bucket}, nil
}

func (u *GCSUploader) Close() error { return u.client.Close() }

func (u *GCSUploader) Upload(ctx context.Context, objectName string, contentType string, r io.Reader) (string, error) {
	obj := u.client.Bucket(u.bucket).Object(objectName)

	w := obj.NewWriter(ctx)
	w.ContentType = contentType

	if _, err := io.Copy(w, r); err != nil {
		_ = w.Close()
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	// make public (simple, langsung bisa diakses frontend)
	if err := obj.ACL().Set(ctx, gcs.AllUsers, gcs.RoleReader); err != nil {
		return "", err
	}

	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", u.bucket, objectName), nil
}
