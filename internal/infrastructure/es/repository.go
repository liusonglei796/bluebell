package es

import (
	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"
	infratrace "bluebell/internal/infrastructure/trace"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

var tracer = infratrace.TracerForModule("dao/es/post")

type postSearchRepo struct {
	client *Client
}

func NewPostSearch(client *Client) domain.PostSearchRepository {
	return &postSearchRepo{client: client}
}

func (r *postSearchRepo) Search(ctx context.Context, keyword string, page, pageSize int) (*entity.SearchResponse, error) {
	ctx, span := tracer.Start(ctx, "ESPostDAO.Search")
	defer span.End()

	return r.client.Search(ctx, keyword, page, pageSize)
}

type postSyncRepo struct {
	client *Client
}

func NewPostSync(client *Client) domain.PostSearchSyncRepository {
	return &postSyncRepo{client: client}
}

func (r *postSyncRepo) SyncPostIndex(ctx context.Context, post *entity.Post) error {
	ctx, span := tracer.Start(ctx, "ESPostDAO.SyncPostIndex")
	defer span.End()

	body, err := json.Marshal(post)
	if err != nil {
		return fmt.Errorf("marshal post failed: %w", err)
	}
	return r.client.IndexDocument(ctx, IndexPost, post.PostID, bytes.NewReader(body))
}

func (r *postSyncRepo) DeletePostIndex(ctx context.Context, postID string) error {
	ctx, span := tracer.Start(ctx, "ESPostDAO.DeletePostIndex")
	defer span.End()

	return r.client.DeleteDocument(ctx, IndexPost, postID)
}
