# Social & Profile API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Social Service and HTTP Handler to support User Profiles, Follow/Unfollow, and Activity Feeds.

**Architecture:** Following DDD principles, we'll define the service interface in the application layer, implement it in a dedicated social package, and expose it via HTTP handlers. Activity messages will be emitted to RabbitMQ for asynchronous processing (if needed) and audit logs.

**Tech Stack:** Go, Gin, GORM, RabbitMQ.

---

### Task 1: Define Social Response DTOs

**Files:**
- Create: `internal/interfaces/http/dto/response/social/social.go`

- [ ] **Step 1: Create the DTO file**

```go
package social

// ProfileResponse 用户资料响应
type ProfileResponse struct {
	UserID         int64  `json:"user_id"`
	Username       string `json:"username"`
	AvatarURL      string `json:"avatar_url"`
	Bio            string `json:"bio"`
	GitHubURL      string `json:"github_url"`
	FollowerCount  int64  `json:"follower_count"`
	FollowingCount int64  `json:"following_count"`
	IsFollowing    bool   `json:"is_following"`
}

// ActivityResponse 用户动态响应
type ActivityResponse struct {
	ID          uint   `json:"id"`
	UserID      int64  `json:"user_id"`
	Type        string `json:"type"`
	TargetID    string `json:"target_id"`
	TargetName  string `json:"target_name"`
	Description string `json:"description"`
	CreatedAt   int64  `json:"created_at"`
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/interfaces/http/dto/response/social/social.go
git commit -m "feat: add social response DTOs"
```

---

### Task 2: Define SocialService Interface

**Files:**
- Modify: `internal/application/interfaces.go`

- [ ] **Step 1: Update imports and add SocialService interface**

```go
// In internal/application/interfaces.go

// Add to imports:
// socialResp "bluebell/internal/interfaces/http/dto/response/social"

// Add interface:
// ========== Social Service 接口 ==========

type SocialService interface {
	GetProfile(ctx context.Context, userID, currentUserID int64) (*socialResp.ProfileResponse, error)
	FollowUser(ctx context.Context, followerID, followingID int64) error
	UnfollowUser(ctx context.Context, followerID, followingID int64) error
	GetActivities(ctx context.Context, userID int64, page, size int) ([]*socialResp.ActivityResponse, error)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/application/interfaces.go
git commit -m "feat: define SocialService interface"
```

---

### Task 3: Implement Social Service

**Files:**
- Create: `internal/application/social/social_service.go`

- [ ] **Step 1: Create the service implementation**

```go
package social

import (
	"bluebell/internal/application"
	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"
	"bluebell/internal/infrastructure/mq"
	socialResp "bluebell/internal/interfaces/http/dto/response/social"
	"context"
	"fmt"
	"time"
)

type socialService struct {
	socialRepo domain.SocialRepository
	userRepo   domain.UserRepository
	publisher  *mq.Publisher
}

func NewSocialService(socialRepo domain.SocialRepository, userRepo domain.UserRepository, publisher *mq.Publisher) application.SocialService {
	return &socialService{
		socialRepo: socialRepo,
		userRepo:   userRepo,
		publisher:  publisher,
	}
}

func (s *socialService) GetProfile(ctx context.Context, userID, currentUserID int64) (*socialResp.ProfileResponse, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	profile, err := s.socialRepo.GetUserProfile(ctx, userID)
	if err != nil {
		// If profile doesn't exist, return basic user info
		profile = &entity.UserProfile{UserID: userID}
	}

	followerCount, _ := s.socialRepo.GetFollowerCount(ctx, userID)
	followingCount, _ := s.socialRepo.GetFollowingCount(ctx, userID)
	
	isFollowing := false
	if currentUserID > 0 {
		isFollowing, _ = s.socialRepo.IsFollowing(ctx, currentUserID, userID)
	}

	return &socialResp.ProfileResponse{
		UserID:         user.UserID,
		Username:       user.Username,
		AvatarURL:      profile.AvatarURL,
		Bio:            profile.Bio,
		GitHubURL:      profile.GitHubURL,
		FollowerCount:  followerCount,
		FollowingCount: followingCount,
		IsFollowing:    isFollowing,
	}, nil
}

func (s *socialService) FollowUser(ctx context.Context, followerID, followingID int64) error {
	if followerID == followingID {
		return fmt.Errorf("cannot follow yourself")
	}

	err := s.socialRepo.FollowUser(ctx, followerID, followingID)
	if err != nil {
		return err
	}

	// Emit activity message
	_ = s.publisher.PublishActivity(ctx, &mq.ActivityMessage{
		UserID:    followerID,
		Type:      "follow",
		TargetID:  fmt.Sprintf("%d", followingID),
		Timestamp: time.Now().Unix(),
	})

	return nil
}

func (s *socialService) UnfollowUser(ctx context.Context, followerID, followingID int64) error {
	err := s.socialRepo.UnfollowUser(ctx, followerID, followingID)
	if err != nil {
		return err
	}

	// Emit activity message
	_ = s.publisher.PublishActivity(ctx, &mq.ActivityMessage{
		UserID:    followerID,
		Type:      "unfollow",
		TargetID:  fmt.Sprintf("%d", followingID),
		Timestamp: time.Now().Unix(),
	})

	return nil
}

func (s *socialService) GetActivities(ctx context.Context, userID int64, page, size int) ([]*socialResp.ActivityResponse, error) {
	activities, err := s.socialRepo.GetActivitiesByUserID(ctx, userID, page, size)
	if err != nil {
		return nil, err
	}

	resp := make([]*socialResp.ActivityResponse, 0, len(activities))
	for _, a := range activities {
		resp = append(resp, &socialResp.ActivityResponse{
			ID:          a.ID,
			UserID:      a.UserID,
			Type:        a.Type,
			TargetID:    a.TargetID,
			TargetName:  a.TargetName,
			Description: a.Description,
			CreatedAt:   a.CreatedAt.Unix(),
		})
	}

	return resp, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/application/social/social_service.go
git commit -m "feat: implement SocialService"
```

---

### Task 4: Implement Social Handler

**Files:**
- Create: `internal/interfaces/http/handler/social_handler/social.go`

- [ ] **Step 1: Create the handler**

```go
package social_handler

import (
	"bluebell/internal/application"
	"bluebell/internal/interfaces/http/render"
	"bluebell/pkg/enum/errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	socialSvc application.SocialService
}

func New(socialSvc application.SocialService) *Handler {
	return &Handler{socialSvc: socialSvc}
}

func (h *Handler) GetProfileHandler(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		render.Error(c, http.StatusBadRequest, errors.ErrorInvalidParam)
		return
	}

	// Get current user ID from context (optional)
	currentUserIDValue, exists := c.Get("userID")
	var currentUserID int64
	if exists {
		currentUserID = currentUserIDValue.(int64)
	}

	profile, err := h.socialSvc.GetProfile(c.Request.Context(), userID, currentUserID)
	if err != nil {
		render.Error(c, http.StatusInternalServerError, err)
		return
	}

	render.Success(c, profile)
}

func (h *Handler) FollowHandler(c *gin.Context) {
	followingIDStr := c.Param("id")
	followingID, err := strconv.ParseInt(followingIDStr, 10, 64)
	if err != nil {
		render.Error(c, http.StatusBadRequest, errors.ErrorInvalidParam)
		return
	}

	followerID := c.MustGet("userID").(int64)

	err = h.socialSvc.FollowUser(c.Request.Context(), followerID, followingID)
	if err != nil {
		render.Error(c, http.StatusInternalServerError, err)
		return
	}

	render.Success(c, nil)
}

func (h *Handler) UnfollowHandler(c *gin.Context) {
	followingIDStr := c.Param("id")
	followingID, err := strconv.ParseInt(followingIDStr, 10, 64)
	if err != nil {
		render.Error(c, http.StatusBadRequest, errors.ErrorInvalidParam)
		return
	}

	followerID := c.MustGet("userID").(int64)

	err = h.socialSvc.UnfollowUser(c.Request.Context(), followerID, followingID)
	if err != nil {
		render.Error(c, http.StatusInternalServerError, err)
		return
	}

	render.Success(c, nil)
}

func (h *Handler) GetActivitiesHandler(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		render.Error(c, http.StatusBadRequest, errors.ErrorInvalidParam)
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	sizeStr := c.DefaultQuery("size", "10")
	page, _ := strconv.Atoi(pageStr)
	size, _ := strconv.Atoi(sizeStr)

	activities, err := h.socialSvc.GetActivities(c.Request.Context(), userID, page, size)
	if err != nil {
		render.Error(c, http.StatusInternalServerError, err)
		return
	}

	render.Success(c, activities)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/interfaces/http/handler/social_handler/social.go
git commit -m "feat: implement SocialHandler"
```

---

### Task 5: Wiring and Registration

**Files:**
- Modify: `internal/interfaces/http/handler/handler.go`
- Modify: `internal/interfaces/http/router/router.go`
- Modify: `internal/di/di.go`

- [ ] **Step 1: Update Handler Provider**

In `internal/interfaces/http/handler/handler.go`:
- Add `SocialHandler *social_handler.Handler` to `Provider` struct.
- Add `social_handler "bluebell/internal/interfaces/http/handler/social_handler"` to imports.
- Update `NewProvider` to accept `SocialService` and initialize `SocialHandler`.

- [ ] **Step 2: Register Routes**

In `internal/interfaces/http/router/router.go`:
- Add routes for social features in `apiV1` and `authGroup`.

- [ ] **Step 3: Update DI Container**

In `internal/di/di.go`:
- Add `Social application.SocialService` to `Services` struct.
- Add `socialsvc "bluebell/internal/application/social"` to imports.
- Initialize `SocialService` in `NewServices`.

- [ ] **Step 4: Commit**

```bash
git add internal/interfaces/http/handler/handler.go internal/interfaces/http/router/router.go internal/di/di.go
git commit -m "feat: wire social service and handler"
```

---

### Task 6: Verification

- [ ] **Step 1: Run build**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 2: Finalize**
