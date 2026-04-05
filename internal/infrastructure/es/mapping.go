package es

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"go.uber.org/zap"
)

// PostMapping defines the Elasticsearch index mapping for posts.
// Uses IK analyzer for Chinese full-text search on title and content.
const PostMapping = `{
    "mappings": {
        "properties": {
            "post_id": { "type": "keyword" },
            "author_id": { "type": "long" },
            "community_id": { "type": "long" },
            "post_title": {
                "type": "text",
                "analyzer": "ik_max_word",
                "search_analyzer": "ik_smart"
            },
            "content": {
                "type": "text",
                "analyzer": "ik_max_word",
                "search_analyzer": "ik_smart"
            },
            "status": { "type": "integer" },
            "created_at": { "type": "date" }
        }
    },
    "settings": {
        "number_of_shards": 1,
        "number_of_replicas": 0
    }
}`

// CreatePostIndex creates the post index if it does not already exist.
// Called by: cmd/bluebell/main.go (line 123: esClient.CreatePostIndex(ctx))
func (c *Client) CreatePostIndex(ctx context.Context) error {
	// 检查 index 是否存在谁拿到 *esapi.Response，谁负责 Body.Close()。
	exists, err := c.es.Indices.Exists([]string{IndexPost})
	if err != nil {
		return fmt.Errorf("check index existence failed: %w", err)
	}
	defer exists.Body.Close()

	if exists.StatusCode == 200 {
		zap.L().Info("ES index already exists", zap.String("index", IndexPost))
		return nil
	}

	// 创建 index
	req := bytes.NewReader([]byte(PostMapping))
	res, err := c.es.Indices.Create(IndexPost, c.es.Indices.Create.WithBody(req))
	if err != nil {
		return fmt.Errorf("create index failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("create index error: %s", string(body))
	}

	zap.L().Info("ES index created", zap.String("index", IndexPost))
	return nil
}
