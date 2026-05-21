package domain

import (
	"bluebell/internal/domain/entity"
	"context"
)

// PostSearchRepository 帖子搜索引擎仓储接口 (读操作)
type PostSearchRepository interface {
	Search(ctx context.Context, keyword string, page, pageSize int) (*entity.SearchResponse, error)
}

// PostSearchSyncRepository 帖子搜索索引同步接口 (写操作)
type PostSearchSyncRepository interface {
	SyncPostIndex(ctx context.Context, post *entity.Post) error
	DeletePostIndex(ctx context.Context, postID string) error
}
