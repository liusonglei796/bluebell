package es

import (
	"context"
	"fmt"
	"io"

	"bluebell/internal/config"

	"github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
)

const IndexPost = "post"

// Client wraps the Elasticsearch client.
type Client struct {
	es *elasticsearch.Client
}

// NewClient creates and validates a new ES client from config.
// Called by: cmd/bluebell/main.go (line 116: es.NewClient(cfg))
func NewClient(cfg *config.Config) (*Client, error) {
	esCfg := cfg.ES
	if esCfg == nil || len(esCfg.Addresses) == 0 {
		return nil, fmt.Errorf("ES config not found")
	}

	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: esCfg.Addresses,
		Username:  esCfg.Username,
		Password:  esCfg.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("create ES client failed: %w", err)
	}

	// 验证连接
	res, err := es.Ping()
	if err != nil {
		return nil, fmt.Errorf("ES ping failed: %w", err)
	}
	defer res.Body.Close()

	zap.L().Info("ES client initialized", zap.Strings("addresses", esCfg.Addresses))

	return &Client{es: es}, nil
}

// ES returns the underlying elasticsearch.Client for direct API calls.
// Called by: sync_consumer.go (indexDocument 和 deleteDocument 中 c.client.ES())
func (c *Client) ES() *elasticsearch.Client {
	return c.es
}

// DeleteDocument deletes a document from the specified index by ID
func (c *Client) DeleteDocument(ctx context.Context, index, docID string) error {
	res, err := c.es.Delete(index, docID, c.es.Delete.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("ES delete document failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ES delete document error: %s", string(body))
	}
	return nil
}
