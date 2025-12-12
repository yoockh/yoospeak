package storage

import (
	"context"
	"errors"
	"io"
	"time"

	gcs "cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
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

// Private bucket: store only the object key (objectName) in DB.
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

	// IMPORTANT: do NOT set ACLs here (breaks with UBLA/private buckets).
	return objectName, nil
}

// Signed URL via service account credentials (GOOGLE_APPLICATION_CREDENTIALS must be a JSON key file).
func (u *GCSUploader) SignedGetURL(ctx context.Context, objectName string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}

	// Find service account email from the JSON key already loaded by ADC is non-trivial without parsing.
	// Simplest: require env GCS_SIGNING_EMAIL when using SignedURL with key file.
	// If you prefer Workload Identity (recommended), we should implement IAMCredentials SignBlob flow instead.
	email := "" // set via env in next patch if you want
	_ = email

	// Minimal validation: ensure object exists (optional)
	_, err := u.client.Bucket(u.bucket).Object(objectName).Attrs(ctx)
	if err != nil {
		if err == iterator.Done {
			return "", errors.New("object not found")
		}
		// if not found, Storage returns error; bubble up
	}

	return "", errors.New("SignedGetURL is not configured: set up signer (see next patch request)")
}
