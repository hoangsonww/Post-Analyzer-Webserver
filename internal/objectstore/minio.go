// Package objectstore wraps MinIO — a self-hosted, S3-API-compatible
// object store — for persisting generated post exports. It's local-only
// infrastructure (no cloud account/credentials involved) while still
// giving genuine S3-compatible PUT/GET/LIST semantics, which is what
// makes it a straightforward swap for real S3 later if this is ever
// deployed to a cloud provider.
package objectstore

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const ExportsBucket = "post-exports"

type Store struct {
	client *minio.Client
}

type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	ContentType  string
}

// New connects to a MinIO (or any S3-compatible) endpoint and ensures the
// exports bucket exists.
func New(ctx context.Context, endpoint, accessKey, secretKey string, useSSL bool) (*Store, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	exists, err := client.BucketExists(ctx, ExportsBucket)
	if err != nil {
		return nil, fmt.Errorf("check minio bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, ExportsBucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("create minio bucket: %w", err)
		}
	}

	return &Store{client: client}, nil
}

func (s *Store) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, ExportsBucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("put object %s: %w", key, err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, ExportsBucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object %s: %w", key, err)
	}
	// GetObject doesn't itself error on a missing key — the error only
	// surfaces on first read/stat, so confirm existence eagerly here.
	if _, err := obj.Stat(); err != nil {
		obj.Close()
		return nil, fmt.Errorf("get object %s: %w", key, err)
	}
	return obj, nil
}

func (s *Store) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	var out []ObjectInfo
	for obj := range s.client.ListObjects(ctx, ExportsBucket, minio.ListObjectsOptions{Prefix: prefix}) {
		if obj.Err != nil {
			return nil, fmt.Errorf("list objects: %w", obj.Err)
		}
		out = append(out, ObjectInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
			ContentType:  obj.ContentType,
		})
	}
	return out, nil
}
