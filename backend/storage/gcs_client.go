package storage

import (
	"context"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// GCSClient implements Client using the official cloud storage SDK.
type GCSClient struct {
	client *storage.Client
}

func NewGCSClient(client *storage.Client) *GCSClient {
	return &GCSClient{client: client}
}

func (c *GCSClient) List(ctx context.Context, req ListRequest) (*ListResponse, error) {
	if req.Bucket == "" {
		return nil, ErrBucketRequired
	}

	query := &storage.Query{Prefix: req.Prefix}
	if req.Delimiter != "" {
		query.Delimiter = req.Delimiter
	}

	it := c.client.Bucket(req.Bucket).Objects(ctx, query)
	pi := it.PageInfo()
	if req.PageToken != "" {
		pi.Token = req.PageToken
	}
	if req.PageSize > 0 {
		pi.MaxSize = req.PageSize
	}

	objects := make([]Object, 0, req.PageSize)
	prefixes := []string{}
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if attrs.Prefix != "" {
			prefixes = append(prefixes, attrs.Prefix)
			continue
		}
		objects = append(objects, Object{Name: attrs.Name})
	}

	return &ListResponse{
		Objects:       objects,
		Prefixes:      prefixes,
		NextPageToken: pi.Token,
	}, nil
}
