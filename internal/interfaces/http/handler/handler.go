package handler

import (
	"bluebell/internal/application/port"
	"bluebell/internal/interfaces/http/handler/bookmark_handler"
	"bluebell/internal/interfaces/http/handler/community_handler"
	"bluebell/internal/interfaces/http/handler/health"
	"bluebell/internal/interfaces/http/handler/post_handler"
	"bluebell/internal/interfaces/http/handler/search_handler"
	"bluebell/internal/interfaces/http/handler/social_handler"
	"bluebell/internal/interfaces/http/handler/sse_handler"
	"bluebell/internal/interfaces/http/handler/user_handler"

	"bluebell/internal/infrastructure/es"
	sse "bluebell/internal/infrastructure/sse"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Provider struct {
	UserHandler      *user_handler.Handler
	PostHandler      *post_handler.Handler
	CommunityHandler *community_handler.Handler
	SocialHandler    *social_handler.Handler
	SearchHandler    *search_handler.Handler
	HealthHandler    *health.Handler
	BookmarkHandler  *bookmark_handler.Handler
	SSEHandler       *sse_handler.Handler
}

func NewProvider(
	userService port.UserService,
	postService port.PostService,
	communityService port.CommunityService,
	socialService port.SocialService,
	bookmarkService port.BookmarkService,
	idGen port.IDGenerator,
	db *gorm.DB,
	rdb *redis.Client,
	esClient *es.Client,
	uploadDir string,
	sseHub *sse.Hub,
) *Provider {
	return &Provider{
		UserHandler:      user_handler.New(userService, uploadDir, idGen),
		PostHandler:      post_handler.New(postService),
		CommunityHandler: community_handler.New(communityService),
		SocialHandler:    social_handler.New(socialService),
		SearchHandler:    search_handler.New(postService),
		HealthHandler:    health.New(db, rdb, esClient),
		BookmarkHandler:  bookmark_handler.New(bookmarkService),
		SSEHandler:       sse_handler.New(sseHub),
	}
}
