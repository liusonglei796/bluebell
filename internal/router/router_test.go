package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bluebell/internal/config"
	"bluebell/internal/controller"
	"bluebell/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouter_PublicCommunityDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		App: &config.AppConfig{
			Mode: "test",
		},
		RateLimit: &config.RateLimitConfig{
			FillInterval: "1s",
			Capacity:     100,
		},
		Timeout: &config.TimeoutConfig{
			Timeout: "10s",
		},
		JWT: &config.JWTConfig{
			Secret:        "test-secret-12345",
			AccessExpiry:  "1h",
			RefreshExpiry: "168h",
		},
	}

	userSvc := service.NewUserService(nil, nil, cfg)
	postSvc := service.NewPostService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	communitySvc := service.NewCommunityService(nil, nil)
	commentSvc := service.NewCommentService(nil, nil, nil, nil, nil)
	relationSvc := service.NewRelationService(nil, nil, nil, nil)
	notifSvc := service.NewNotificationService(nil, nil, nil)
	bookmarkSvc := service.NewBookmarkService(nil, nil, nil, postSvc)
	tagSvc := service.NewTagService(nil, nil, nil)

	userCtrl := controller.NewUserController(userSvc)
	postCtrl := controller.NewPostController(postSvc)
	communityCtrl := controller.NewCommunityController(communitySvc)
	commentCtrl := controller.NewCommentController(commentSvc)
	relationCtrl := controller.NewRelationController(relationSvc)
	notifCtrl := controller.NewNotificationController(notifSvc)
	bookmarkCtrl := controller.NewBookmarkController(bookmarkSvc)
	tagCtrl := controller.NewTagController(tagSvc)

	r, err := NewRouter("test", userCtrl, postCtrl, communityCtrl, commentCtrl, relationCtrl, notifCtrl, bookmarkCtrl, tagCtrl, cfg, nil)
	require.NoError(t, err)

	// 1. GET /api/v1/community/:id 没有 Authorization header 应该不会返回 401（未授权）
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/community/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusUnauthorized, w.Code, "GET /api/v1/community/:id 应为公开路由，不应返回 401")

	// 2. POST /api/v1/community 没有 Authorization header 必须返回 401
	postReq, _ := http.NewRequest(http.MethodPost, "/api/v1/community", nil)
	wPost := httptest.NewRecorder()
	r.ServeHTTP(wPost, postReq)
	assert.Equal(t, http.StatusUnauthorized, wPost.Code, "POST /api/v1/community 必须受 JWT 保护，返回 401")

	// 3. GET /api/v1/posts 带非法或空 Token 时，可选认证中间件不应阻断请求（不返回 401）
	postListReq, _ := http.NewRequest(http.MethodGet, "/api/v1/posts", nil)
	postListReq.Header.Set("Authorization", "Bearer invalid-token-string")
	wList := httptest.NewRecorder()
	r.ServeHTTP(wList, postListReq)
	assert.NotEqual(t, http.StatusUnauthorized, wList.Code, "GET /api/v1/posts 为可选认证公开路由，即使 Token 无效也不应返回 401")
}
