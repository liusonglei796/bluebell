# DDD Refactoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor the project to delete service layer interfaces and the di package, flatten the application folder structure, and wire dependencies directly in the main function as per DDD best practices.

**Architecture:** We will move all application service files to the `internal/application/` root directory, delete the interfaces in `interfaces.go`, rename the structs to public `CommunityService`, `PostService`, etc., and update handlers and the main file to use these concrete types directly.

**Tech Stack:** Go (Golang)

---

### Task 1: Flatten and Refactor Community Service

**Files:**
- Create: `internal/application/community_service.go`
- Delete: `internal/application/community/community_service.go`

- [ ] **Step 1: Move and Refactor `community_service.go`**
  Write the new `internal/application/community_service.go` containing:
  ```go
  package application

  import (
  	"bluebell/internal/domain"
  	communityResp "bluebell/internal/application/dto/response/community"
  	"bluebell/internal/domain/entity"
  	"context"
  	"strconv"
  	"go.uber.org/zap"
  	"bluebell/internal/infrastructure/logger"
  	"bluebell/internal/infrastructure/trace"
  )

  var tracerCommunity = trace.TracerForModule("service/community")

  type CommunityService struct {
  	communityRepo domain.CommunityRepository
  	userRepo      domain.UserRepository
  }

  func NewCommunityService(communityRepo domain.CommunityRepository, userRepo domain.UserRepository) *CommunityService {
  	return &CommunityService{
  		communityRepo: communityRepo,
  		userRepo:      userRepo,
  	}
  }

  func toResponse(c *entity.Community) *communityResp.Response {
  	return &communityResp.Response{
  		ID:           strconv.FormatInt(c.ID, 10),
  		Name:         c.CommunityName,
  		Introduction: c.Introduction,
  	}
  }

  func (s *CommunityService) GetCommunityList(ctx context.Context) ([]*communityResp.Response, error) {
  	data, err := s.communityRepo.GetCommunityList(ctx)
  	if err != nil {
  		logger.WithContext(ctx).Error("communityRepo.GetCommunityList failed", zap.Error(err))
  		return nil, entity.Wrap(entity.ErrServerBusy, err)
  	}

  	result := make([]*communityResp.Response, 0, len(data))
  	for _, c := range data {
  		result = append(result, toResponse(c))
  	}
  	return result, nil
  }

  func (s *CommunityService) GetCommunityDetail(ctx context.Context, id int64) (*communityResp.Response, error) {
  	data, err := s.communityRepo.GetCommunityDetailByID(ctx, id)
  	if err != nil {
  		logger.WithContext(ctx).Error("communityRepo.GetCommunityDetailByID failed",
  			zap.Int64("community_id", id),
  			zap.Error(err))
  		return nil, entity.Wrap(entity.ErrServerBusy, err)
  	}

  	if data == nil {
  		return nil, entity.ErrNotFound
  	}

  	return toResponse(data), nil
  }

  func (s *CommunityService) CreateCommunity(ctx context.Context, name, introduction string, userID int64) error {
  	user, err := s.userRepo.GetUserByID(ctx, userID)
  	if err != nil {
  		logger.WithContext(ctx).Error("userRepo.GetUserByID failed",
  			zap.Int64("user_id", userID),
  			zap.Error(err))
  		return entity.Wrap(entity.ErrServerBusy, err)
  	}
  	if user == nil || !user.IsAdmin() {
  		return entity.ErrForbidden
  	}

  	community := &entity.Community{
  		CommunityName: name,
  		Introduction:  introduction,
  	}
  	if err := s.communityRepo.CreateCommunity(ctx, community); err != nil {
  		logger.WithContext(ctx).Error("communityRepo.CreateCommunity failed",
  			zap.String("community_name", name),
  			zap.Error(err))
  		return entity.Wrap(entity.ErrServerBusy, err)
  	}

  	return nil
  }
  ```
- [ ] **Step 2: Delete old file**
  Delete `internal/application/community/community_service.go` and its parent directory if empty.
- [ ] **Step 3: Commit**
  ```bash
  git add internal/application/community_service.go
  git rm internal/application/community/community_service.go
  git commit -m "refactor: flatten and concrete CommunityService"
  ```

---

### Task 2: Flatten and Refactor Bookmark Service

**Files:**
- Create: `internal/application/bookmark_service.go`
- Delete: `internal/application/bookmark/bookmark_service.go`

- [ ] **Step 1: Move and Refactor `bookmark_service.go`**
  Write the new `internal/application/bookmark_service.go` containing:
  ```go
  package application

  import (
  	bookmarkresp "bluebell/internal/application/dto/response/bookmark"
  	"bluebell/internal/domain"
  	"bluebell/internal/domain/entity"
  	"bluebell/internal/infrastructure/logger"
  	infratrace "bluebell/internal/infrastructure/trace"
  	"context"
  	"errors"
  	"fmt"
  	"strconv"
  	"go.uber.org/zap"
  )

  var tracerBookmark = infratrace.TracerForModule("service/bookmark")

  type BookmarkService struct {
  	bookmarkRepo  domain.BookmarkRepository
  	postRepo      domain.PostRepository
  	userRepo      domain.UserRepository
  	communityRepo domain.CommunityRepository
  }

  func NewBookmarkService(
  	bookmarkRepo domain.BookmarkRepository,
  	postRepo domain.PostRepository,
  	userRepo domain.UserRepository,
  	communityRepo domain.CommunityRepository,
  ) *BookmarkService {
  	return &BookmarkService{
  		bookmarkRepo:  bookmarkRepo,
  		postRepo:      postRepo,
  		userRepo:      userRepo,
  		communityRepo: communityRepo,
  	}
  }

  func (s *BookmarkService) CreateBookmark(ctx context.Context, userID, postID int64) error {
  	ctx, span := tracerBookmark.Start(ctx, "BookmarkService.CreateBookmark")
  	defer span.End()
  	infratrace.WithUserID(ctx, userID)
  	infratrace.WithPostID(ctx, postID)

  	post, err := s.postRepo.GetPostByID(ctx, postID)
  	if err != nil {
  		if errors.Is(err, entity.ErrUserNotExist) {
  			return entity.ErrInvalidParam
  		}
  		logger.WithContext(ctx).Error("GetPostByID failed",
  			zap.Int64("post_id", postID),
  			zap.Error(err))
  		return err
  	}
  	if post == nil {
  		return entity.ErrInvalidParam
  	}

  	bookmarked, err := s.bookmarkRepo.IsBookmarked(ctx, userID, postID)
  	if err != nil {
  		logger.WithContext(ctx).Error("Check bookmark status failed",
  			zap.Int64("user_id", userID),
  			zap.Int64("post_id", postID),
  			zap.Error(err))
  		return err
  	}
  	if bookmarked {
  		return nil
  	}

  	bookmark := &entity.Bookmark{
  		UserID: userID,
  		PostID: postID,
  	}
  	if err := s.bookmarkRepo.CreateBookmark(ctx, bookmark); err != nil {
  		logger.WithContext(ctx).Error("Create bookmark failed",
  			zap.Int64("user_id", userID),
  			zap.Int64("post_id", postID),
  			zap.Error(err))
  		return err
  	}

  	infratrace.SetSpanSuccess(ctx)
  	return nil
  }

  func (s *BookmarkService) DeleteBookmark(ctx context.Context, userID, postID int64) error {
  	ctx, span := tracerBookmark.Start(ctx, "BookmarkService.DeleteBookmark")
  	defer span.End()
  	infratrace.WithUserID(ctx, userID)
  	infratrace.WithPostID(ctx, postID)

  	if err := s.bookmarkRepo.DeleteBookmark(ctx, userID, postID); err != nil {
  		logger.WithContext(ctx).Error("Delete bookmark failed",
  			zap.Int64("user_id", userID),
  			zap.Int64("post_id", postID),
  			zap.Error(err))
  		return err
  	}

  	infratrace.SetSpanSuccess(ctx)
  	return nil
  }

  func (s *BookmarkService) GetUserBookmarks(ctx context.Context, userID int64, page, size int) (*bookmarkresp.BookmarkListResponse, error) {
  	ctx, span := tracerBookmark.Start(ctx, "BookmarkService.GetUserBookmarks")
  	defer span.End()
  	infratrace.WithUserID(ctx, userID)

  	if page <= 0 {
  		page = 1
  	}
  	if size <= 0 || size > 50 {
  		size = 20
  	}

  	bookmarks, err := s.bookmarkRepo.GetUserBookmarks(ctx, userID, page, size)
  	if err != nil {
  		logger.WithContext(ctx).Error("Get user bookmarks failed",
  			zap.Int64("user_id", userID),
  			zap.Error(err))
  		return nil, err
  	}

  	total, err := s.bookmarkRepo.GetBookmarkCount(ctx, userID)
  	if err != nil {
  		logger.WithContext(ctx).Error("Get bookmark count failed",
  			zap.Int64("user_id", userID),
  			zap.Error(err))
  		return nil, err
  	}

  	var postIDStrs []string
  	for _, b := range bookmarks {
  		postIDStrs = append(postIDStrs, strconv.FormatInt(b.PostID, 10))
  	}

  	posts, err := s.postRepo.GetPostListByIDsWithPreload(ctx, postIDStrs)
  	if err != nil {
  		logger.WithContext(ctx).Error("Get posts failed", zap.Error(err))
  	}

  	postMap := make(map[string]*entity.Post)
  	for _, p := range posts {
  		postMap[p.PostID] = p
  	}

  	var authorIDs []int64
  	for _, p := range posts {
  		if p.AuthorID > 0 {
  			authorIDs = append(authorIDs, p.AuthorID)
  		}
  	}

  	authorMap := make(map[int64]*entity.User)
  	if len(authorIDs) > 0 {
  		authors, err := s.userRepo.GetUsersByIDs(ctx, authorIDs)
  		if err != nil {
  			logger.WithContext(ctx).Error("Get authors failed", zap.Error(err))
  		} else {
  			for _, a := range authors {
  				authorMap[a.UserID] = a
  			}
  		}
  	}

  	var communityIDs []int64
  	for _, p := range posts {
  		if p.CommunityID > 0 {
  			communityIDs = append(communityIDs, p.CommunityID)
  		}
  	}

  	communityMap := make(map[int64]*entity.Community)
  	for _, cid := range communityIDs {
  		c, err := s.communityRepo.GetCommunityDetailByID(ctx, cid)
  		if err != nil {
  			logger.WithContext(ctx).Error("Get community failed", zap.Error(err))
  		} else {
  			communityMap[cid] = c
  		}
  	}

  	responses := make([]*bookmarkresp.BookmarkResponse, 0, len(bookmarks))
  	for _, b := range bookmarks {
  		postIDStr := strconv.FormatInt(b.PostID, 10)
  		resp := &bookmarkresp.BookmarkResponse{
  			PostID:    postIDStr,
  			CreatedAt: b.CreatedAt.Format("2006-01-02 15:04:05"),
  		}
  		if p, ok := postMap[postIDStr]; ok {
  			resp.PostTitle = p.PostTitle
  			if a, ok := authorMap[p.AuthorID]; ok {
  				resp.AuthorName = a.UserName
  			}
  			resp.CommunityID = p.CommunityID
  			if c, ok := communityMap[p.CommunityID]; ok {
  				resp.CommunityName = c.CommunityName
  			}
  		}
  		responses = append(responses, resp)
  	}

  	infratrace.SetSpanSuccess(ctx)
  	return &bookmarkresp.BookmarkListResponse{
  		Bookmarks: responses,
  		Total:     total,
  		Page:      page,
  		Size:      size,
  	}, nil
  }

  func (s *BookmarkService) IsBookmarked(ctx context.Context, userID, postID int64) (*bookmarkresp.BookmarkStatusResponse, error) {
  	ctx, span := tracerBookmark.Start(ctx, "BookmarkService.IsBookmarked")
  	defer span.End()
  	infratrace.WithUserID(ctx, userID)
  	infratrace.WithPostID(ctx, postID)

  	bookmarked, err := s.bookmarkRepo.IsBookmarked(ctx, userID, postID)
  	if err != nil {
  		logger.WithContext(ctx).Error("Check bookmark status failed",
  			zap.Int64("user_id", userID),
  			zap.Int64("post_id", postID),
  			zap.Error(err))
  		return nil, err
  	}

  	count, err := s.bookmarkRepo.GetBookmarkCount(ctx, userID)
  	if err != nil {
  		logger.WithContext(ctx).Error("Get bookmark count failed",
  			zap.Int64("user_id", userID),
  			zap.Error(err))
  		return nil, err
  	}

  	infratrace.SetSpanSuccess(ctx)
  	return &bookmarkresp.BookmarkStatusResponse{
  		Bookmarked: bookmarked,
  		Count:      count,
  	}, nil
  }
  ```
- [ ] **Step 2: Delete old file**
  Delete `internal/application/bookmark/bookmark_service.go`.
- [ ] **Step 3: Commit**
  ```bash
  git add internal/application/bookmark_service.go
  git rm internal/application/bookmark/bookmark_service.go
  git commit -m "refactor: flatten and concrete BookmarkService"
  ```

---

### Task 3: Flatten and Refactor Post Service

**Files:**
- Create: `internal/application/post_service.go`
- Delete: `internal/application/post/post_service.go`

- [ ] **Step 1: Move and Refactor `post_service.go`**
  Write the new `internal/application/post_service.go` containing:
  ```go
  package application

  import (
  	"bluebell/internal/domain"
  	postreq "bluebell/internal/application/dto/request/post"
  	postResp "bluebell/internal/application/dto/response/post"
  	searchResp "bluebell/internal/application/dto/response/search"
  	"bluebell/internal/infrastructure/snowflake"
  	"bluebell/internal/infrastructure/mq"
  	"bluebell/internal/domain/entity"
  	"context"
  	"fmt"
  	"strconv"
  	"time"
  	"go.uber.org/zap"
  	"bluebell/internal/infrastructure/logger"
  	"bluebell/internal/infrastructure/trace"
  )

  var tracerPost = trace.TracerForModule("service/post")

  type PostService struct {
  	postRepo       domain.PostRepository
  	postCache      domain.PostCacheRepository
  	voteRepo       domain.VoteRepository
  	remarkRepo     domain.RemarkRepository
  	publisher      *mq.Publisher
  	searchRepo     domain.PostSearchRepository
  	searchSyncRepo domain.PostSearchSyncRepository
  }

  func NewPostService(
  	postRepo domain.PostRepository,
  	postCache domain.PostCacheRepository,
  	voteRepo domain.VoteRepository,
  	remarkRepo domain.RemarkRepository,
  	publisher *mq.Publisher,
  	searchRepo domain.PostSearchRepository,
  	searchSyncRepo domain.PostSearchSyncRepository,
  ) *PostService {
  	return &PostService{
  		postRepo:       postRepo,
  		postCache:      postCache,
  		voteRepo:       voteRepo,
  		remarkRepo:     remarkRepo,
  		publisher:      publisher,
  		searchRepo:     searchRepo,
  		searchSyncRepo: searchSyncRepo,
  	}
  }

  func (s *PostService) CreatePost(ctx context.Context, p *postreq.CreatePostRequest, authorID int64) (postID string, err error) {
  	postIDInt := snowflake.GenID()
  	postID = strconv.FormatInt(postIDInt, 10)

  	post := &entity.Post{
  		PostID:      postID,
  		AuthorID:    authorID,
  		CommunityID: p.CommunityID,
  		PostTitle:   p.Title,
  		Content:     p.Content,
  		Status:      entity.PostStatusPublished,
  	}

  	if !post.IsValid() {
  		return "", entity.ErrInvalidParam
  	}

  	err = s.postRepo.CreatePost(ctx, post)
  	if err != nil {
  		logger.WithContext(ctx).Error("postRepo.CreatePost failed",
  			zap.Int64("post_id", postIDInt),
  			zap.Error(err))
  		return "", entity.Wrap(entity.ErrServerBusy, err)
  	}

  	err = s.postCache.CreatePost(ctx, postIDInt, p.CommunityID)
  	if err != nil {
  		logger.WithContext(ctx).Error("postCache.CreatePost failed",
  			zap.Int64("post_id", postIDInt),
  			zap.Error(err))
  	}

  	_ = s.publisher.PublishActivity(ctx, &mq.ActivityMessage{
  		UserID:     authorID,
  		Type:       "post_created",
  		TargetID:   postID,
  		TargetName: p.Title,
  		Timestamp:  time.Now().Unix(),
  	})

  	if s.searchSyncRepo != nil {
  		if err := s.searchSyncRepo.SyncPostIndex(ctx, post); err != nil {
  			logger.WithContext(ctx).Warn("searchSyncRepo.SyncPostIndex failed",
  				zap.String("post_id", post.PostID),
  				zap.Error(err))
  		}
  	}

  	return postID, nil
  }

  func (s *PostService) GetPostByID(ctx context.Context, pid int64) (data *postResp.DetailResponse, err error) {
  	post, err := s.postRepo.GetPostByID(ctx, pid)
  	if err != nil {
  		logger.WithContext(ctx).Error("postRepo.GetPostByID failed",
  			zap.Int64("post_id", pid),
  			zap.Error(err))
  		return nil, entity.Wrap(entity.ErrServerBusy, err)
  	}

  	if post == nil || post.PostID == "" {
  		return nil, entity.ErrNotFound
  	}

  	if post.Author == nil || post.Author.UserID == 0 {
  		logger.WithContext(ctx).Warn("author not found for post",
  			zap.Int64("post_id", pid),
  			zap.Int64("author_id", post.AuthorID))
  		return nil, entity.ErrNotFound
  	}

  	if post.Community == nil || post.Community.ID == 0 {
  		logger.WithContext(ctx).Warn("community not found for post",
  			zap.Int64("post_id", pid),
  			zap.Int64("community_id", post.CommunityID))
  		return nil, entity.ErrNotFound
  	}

  	data = &postResp.DetailResponse{
  		ID:          post.PostID,
  		AuthorID:    strconv.FormatInt(post.AuthorID, 10),
  		CommunityID: post.CommunityID,
  		Status:      post.Status,
  		Title:       post.PostTitle,
  		Content:     post.Content,
  		CreateTime:  post.CreatedAt,
  		AuthorName:  post.Author.UserName,
  	}

  	voteData, err := s.postCache.GetPostsVoteData(ctx, []string{post.PostID})
  	if err != nil {
  		logger.WithContext(ctx).Warn("postCache.GetPostsVoteData failed in GetPostByID",
  			zap.String("post_id", post.PostID),
  			zap.Error(err))
  	} else if len(voteData) > 0 {
  		data.VoteNum = voteData[0]
  	}

  	return data, nil
  }

  func (s *PostService) GetPostList(ctx context.Context, p *postreq.PostListRequest) (data []*postResp.DetailResponse, err error) {
  	ids, err := s.postCache.GetPostIDsInOrder(ctx, p.Order, p.Page, p.Size)
  	if err != nil {
  		logger.WithContext(ctx).Error("postCache.GetPostIDsInOrder failed",
  			zap.String("order", p.Order),
  			zap.Error(err))
  		return nil, entity.Wrap(entity.ErrServerBusy, err)
  	}

  	if len(ids) == 0 {
  		logger.WithContext(ctx).Warn("postCache.GetPostIDsInOrder() return 0 data")
  		data = make([]*postResp.DetailResponse, 0)
  		return
  	}

  	posts, err := s.postRepo.GetPostListByIDsWithPreload(ctx, ids)
  	if err != nil {
  		logger.WithContext(ctx).Error("postRepo.GetPostListByIDsWithPreload failed", zap.Error(err))
  		return nil, entity.Wrap(entity.ErrServerBusy, err)
  	}

  	voteData, err := s.postCache.GetPostsVoteData(ctx, ids)
  	if err != nil {
  		logger.WithContext(ctx).Error("postCache.GetPostsVoteData failed", zap.Error(err))
  		voteData = make([]int64, len(ids))
  	}

  	for idx, post := range posts {
  		var authorName string
  		if post.Author != nil {
  			authorName = post.Author.UserName
  		} else {
  			logger.WithContext(ctx).Error("author not preloaded for post",
  				zap.String("post_id", post.PostID),
  				zap.Int64("author_id", post.AuthorID))
  		}

  		postDetail := &postResp.DetailResponse{
  			ID:          post.PostID,
  			AuthorID:    strconv.FormatInt(post.AuthorID, 10),
  			CommunityID: post.CommunityID,
  			Status:      post.Status,
  			Title:       post.PostTitle,
  			Content:     post.Content,
  			CreateTime:  post.CreatedAt,
  			AuthorName:  authorName,
  			VoteNum:     voteData[idx],
  		}
  		data = append(data, postDetail)
  	}

  	return
  }

  func (s *PostService) GetCommunityPostList(ctx context.Context, p *postreq.PostListRequest) (data []*postResp.DetailResponse, err error) {
  	ids, err := s.postCache.GetCommunityPostIDsInOrder(ctx, p.CommunityID, p.Order, p.Page, p.Size)
  	if err != nil {
  		logger.WithContext(ctx).Error("postCache.GetCommunityPostIDsInOrder failed",
  			zap.Int64("community_id", p.CommunityID),
  			zap.String("order", p.Order),
  			zap.Error(err))
  		return nil, entity.Wrap(entity.ErrServerBusy, err)
  	}

  	if len(ids) == 0 {
  		data = make([]*postResp.DetailResponse, 0)
  		return data, nil
  	}

  	posts, err := s.postRepo.GetPostListByIDsWithPreload(ctx, ids)
  	if err != nil {
  		logger.WithContext(ctx).Error("postRepo.GetPostListByIDsWithPreload failed", zap.Error(err))
  		return nil, entity.Wrap(entity.ErrServerBusy, err)
  	}

  	voteData, err := s.postCache.GetPostsVoteData(ctx, ids)
  	if err != nil {
  		logger.WithContext(ctx).Error("postCache.GetPostsVoteData failed", zap.Error(err))
  		voteData = make([]int64, len(ids))
  	}

  	data = make([]*postResp.DetailResponse, 0, len(posts))
  	for idx, post := range posts {
  		var authorName string
  		if post.Author != nil {
  			authorName = post.Author.UserName
  		} else {
  			logger.WithContext(ctx).Error("author not preloaded for post",
  				zap.String("post_id", post.PostID),
  				zap.Int64("author_id", post.AuthorID))
  		}

  		postDetail := &postResp.DetailResponse{
  			ID:          post.PostID,
  			AuthorID:    strconv.FormatInt(post.AuthorID, 10),
  			CommunityID: post.CommunityID,
  			Status:      post.Status,
  			Title:       post.PostTitle,
  			Content:     post.Content,
  			CreateTime:  post.CreatedAt,
  			AuthorName:  authorName,
  			VoteNum:     voteData[idx],
  		}
  		data = append(data, postDetail)
  	}

  	return data, nil
  }

  func (s *PostService) DeletePost(ctx context.Context, postID int64, userID int64) error {
  	post, err := s.postRepo.GetPostByID(ctx, postID)
  	if err != nil {
  		logger.WithContext(ctx).Error("postRepo.GetPostByID failed",
  			zap.Int64("post_id", postID),
  			zap.Error(err))
  		return entity.Wrap(entity.ErrServerBusy, err)
  	}
  	if !post.IsValid() {
  		return entity.ErrNotFound
  	}

  	if err := post.CanBeDeletedBy(userID); err != nil {
  		return err
  	}

  	if err := s.remarkRepo.DeleteRemarksByPostID(ctx, postID); err != nil {
  		logger.WithContext(ctx).Error("remarkRepo.DeleteRemarksByPostID failed",
  			zap.Int64("post_id", postID),
  			zap.Error(err))
  		return entity.Wrap(entity.ErrServerBusy, err)
  	}

  	err = s.postRepo.DeletePostByAuthor(ctx, postID, userID)
  	if err != nil {
  		logger.WithContext(ctx).Error("postRepo.DeletePostByAuthor failed",
  			zap.Int64("post_id", postID),
  			zap.Int64("user_id", userID),
  			zap.Error(err))
  		return entity.Wrap(entity.ErrServerBusy, err)
  	}

  	if s.searchSyncRepo != nil {
  		postIDStr := strconv.FormatInt(postID, 10)
  		if err := s.searchSyncRepo.DeletePostIndex(ctx, postIDStr); err != nil {
  			logger.WithContext(ctx).Error("searchSyncRepo.DeletePostIndex failed",
  				zap.Int64("post_id", postID),
  				zap.Error(err))
  		}
  	}

  	if err := s.postCache.DeletePost(ctx, postID, post.CommunityID); err != nil {
  		logger.WithContext(ctx).Error("postCache.DeletePost failed",
  			zap.Int64("post_id", postID),
  			zap.Error(err))
  	}

  	return nil
  }

  func (s *PostService) VoteForPost(ctx context.Context, userID int64, p *postreq.VoteRequest) error {
  	vote := &entity.Vote{
  		PostID:    p.PostID,
  		UserID:    userID,
  		Direction: p.Direction,
  	}
  	if err := vote.Validate(); err != nil {
  		return err
  	}

  	postIDStr := strconv.FormatInt(p.PostID, 10)
  	userIDStr := strconv.FormatInt(userID, 10)

  	communityID, err := s.postCache.GetPostCommunityID(ctx, p.PostID)
  	if err != nil {
  		return fmt.Errorf("get community id for vote failed: %w", err)
  	}

  	if err := s.postCache.VoteForPost(ctx, userIDStr, postIDStr, strconv.FormatInt(communityID, 10), float64(p.Direction)); err != nil {
  		return err
  	}

  	if s.publisher != nil {
  		_ = s.publisher.PublishVote(ctx, &mq.VoteMessage{
  			MsgID:  strconv.FormatInt(snowflake.GenID(), 10),
  			PostID: postIDStr,
  			UserID: userIDStr,
  			Action: int(p.Direction),
  		})
  		_ = s.publisher.PublishActivity(ctx, &mq.ActivityMessage{
  			UserID:    userID,
  			Type:      "vote_up",
  			TargetID:  postIDStr,
  			Timestamp: time.Now().Unix(),
  		})
  	}

  	return nil
  }

  func (s *PostService) RemarkPost(ctx context.Context, req *postreq.RemarkRequest, userID int64) (remarkID uint, err error) {
  	post, err := s.postRepo.GetPostByID(ctx, req.PostID)
  	if err != nil {
  		logger.WithContext(ctx).Error("remarkPost: postRepo.GetPostByID failed",
  			zap.Int64("post_id", req.PostID),
  			zap.Error(err))
  		return 0, entity.Wrap(entity.ErrServerBusy, err)
  	}
  	if !post.IsValid() {
  		return 0, entity.ErrNotFound
  	}

  	remark := &entity.Remark{
  		PostID:   req.PostID,
  		Content:  req.Content,
  		AuthorID: userID,
  		ReplyTo:  req.ReplyTo,
  	}
  	if err := remark.Validate(); err != nil {
  		return 0, err
  	}

  	if err := s.remarkRepo.CreateRemark(ctx, remark); err != nil {
  		logger.WithContext(ctx).Error("remarkPost: remarkRepo.CreateRemark failed",
  			zap.Int64("post_id", req.PostID),
  			zap.Int64("author_id", userID),
  			zap.Error(err))
  		return 0, entity.Wrap(entity.ErrServerBusy, err)
  	}

  	return remark.ID, nil
  }

  func (s *PostService) GetPostRemarks(ctx context.Context, postID int64, replyTo int64) ([]*postResp.RemarkDetail, error) {
  	remarks, err := s.remarkRepo.GetRemarksByPostID(ctx, postID)
  	if err != nil {
  		logger.WithContext(ctx).Error("getPostRemarks: remarkRepo.GetRemarksByPostID failed",
  			zap.Int64("post_id", postID),
  			zap.Error(err))
  		return nil, entity.Wrap(entity.ErrServerBusy, err)
  	}

  	resp := make([]*postResp.RemarkDetail, 0, len(remarks))
  	for _, r := range remarks {
  		if replyTo > 0 && r.ReplyTo != replyTo {
  			continue
  		}
  		if replyTo == 0 && r.ReplyTo > 0 {
  			continue
  		}

  		authorName := "已注销用户"
  		if r.Author != nil {
  			authorName = r.Author.UserName
  		}
  		resp = append(resp, &postResp.RemarkDetail{
  			ID:         r.ID,
  			Content:    r.Content,
  			AuthorName: authorName,
  			CreateTime: r.CreatedAt,
  		})
  	}

  	return resp, nil
  }

  func (s *PostService) SearchPosts(ctx context.Context, keyword string, page, pageSize int) (*searchResp.SearchResponse, error) {
  	if s.searchRepo == nil {
  		logger.WithContext(ctx).Warn("searchRepo is not initialized")
  		return &searchResp.SearchResponse{Posts: []searchResp.SearchPostDoc{}}, nil
  	}

  	res, err := s.searchRepo.Search(ctx, keyword, page, pageSize)
  	if err != nil {
  		return nil, err
  	}

  	resp := &searchResp.SearchResponse{
  		Total:    res.Total,
  		Page:     res.Page,
  		PageSize: res.PageSize,
  		Posts:    make([]searchResp.SearchPostDoc, 0, len(res.Posts)),
  	}
  	for _, p := range res.Posts {
  		resp.Posts = append(resp.Posts, searchResp.SearchPostDoc{
  			PostID:           p.PostID,
  			AuthorID:         p.AuthorID,
  			CommunityID:      p.CommunityID,
  			PostTitle:        p.PostTitle,
  			Content:          p.Content,
  			Status:           p.Status,
  			CreatedAt:        p.CreatedAt,
  			HighlightTitle:   p.HighlightTitle,
  			HighlightContent: p.HighlightContent,
  		})
  	}
  	return resp, nil
  }
  ```
- [ ] **Step 2: Delete old file**
  Delete `internal/application/post/post_service.go`.
- [ ] **Step 3: Commit**
  ```bash
  git add internal/application/post_service.go
  git rm internal/application/post/post_service.go
  git commit -m "refactor: flatten and concrete PostService"
  ```

---

### Task 4: Flatten and Refactor Social Service

**Files:**
- Create: `internal/application/social_service.go`
- Delete: `internal/application/social/social_service.go`

- [ ] **Step 1: Move and Refactor `social_service.go`**
  Write the new `internal/application/social_service.go` containing:
  ```go
  package application

  import (
  	"bluebell/internal/domain"
  	"bluebell/internal/domain/entity"
  	"bluebell/internal/infrastructure/mq"
  	socialResp "bluebell/internal/application/dto/response/social"
  	"bluebell/internal/infrastructure/trace"
  	"context"
  	"fmt"
  	"time"
  )

  var tracerSocial = trace.TracerForModule("service/social")

  type SocialService struct {
  	socialRepo domain.SocialRepository
  	userRepo   domain.UserRepository
  	publisher  *mq.Publisher
  }

  func NewSocialService(socialRepo domain.SocialRepository, userRepo domain.UserRepository, publisher *mq.Publisher) *SocialService {
  	return &SocialService{
  		socialRepo: socialRepo,
  		userRepo:   userRepo,
  		publisher:  publisher,
  	}
  }

  func (s *SocialService) GetProfile(ctx context.Context, userID, currentUserID int64) (*socialResp.ProfileResponse, error) {
  	user, err := s.userRepo.GetUserByID(ctx, userID)
  	if err != nil || user == nil {
  		return nil, err
  	}

  	profile, err := s.socialRepo.GetUserProfile(ctx, userID)
  	if err != nil || profile == nil {
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
  		Username:       user.UserName,
  		AvatarURL:      profile.AvatarURL,
  		Bio:            profile.Bio,
  		GitHubURL:      profile.GitHubURL,
  		FollowerCount:  followerCount,
  		FollowingCount: followingCount,
  		IsFollowing:    isFollowing,
  	}, nil
  }

  func (s *SocialService) FollowUser(ctx context.Context, followerID, followingID int64) error {
  	if followerID == followingID {
  		return fmt.Errorf("cannot follow yourself")
  	}

  	err := s.socialRepo.FollowUser(ctx, followerID, followingID)
  	if err != nil {
  		return err
  	}

  	_ = s.publisher.PublishActivity(ctx, &mq.ActivityMessage{
  		UserID:    followerID,
  		Type:      "follow",
  		TargetID:  fmt.Sprintf("%d", followingID),
  		Timestamp: time.Now().Unix(),
  	})

  	return nil
  }

  func (s *SocialService) UnfollowUser(ctx context.Context, followerID, followingID int64) error {
  	err := s.socialRepo.UnfollowUser(ctx, followerID, followingID)
  	if err != nil {
  		return err
  	}

  	_ = s.publisher.PublishActivity(ctx, &mq.ActivityMessage{
  		UserID:    followerID,
  		Type:      "unfollow",
  		TargetID:  fmt.Sprintf("%d", followingID),
  		Timestamp: time.Now().Unix(),
  	})

  	return nil
  }

  func (s *SocialService) GetActivities(ctx context.Context, userID int64, page, size int) ([]*socialResp.ActivityResponse, error) {
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
- [ ] **Step 2: Delete old file**
  Delete `internal/application/social/social_service.go`.
- [ ] **Step 3: Commit**
  ```bash
  git add internal/application/social_service.go
  git rm internal/application/social/social_service.go
  git commit -m "refactor: flatten and concrete SocialService"
  ```

---

### Task 5: Flatten and Refactor User Service & tests

**Files:**
- Create: `internal/application/user_service.go`
- Create: `internal/application/user_service_test.go`
- Delete: `internal/application/user/user_service.go`
- Delete: `internal/application/user/user_service_test.go`

- [ ] **Step 1: Move and Refactor `user_service.go`**
  Write the new `internal/application/user_service.go` containing:
  ```go
  package application

  import (
  	"bluebell/internal/domain"
  	userreq "bluebell/internal/application/dto/request/user"
  	"bluebell/internal/infrastructure/snowflake"
  	"bluebell/internal/domain/entity"
  	"context"
  	"errors"
  	"fmt"
  	"strings"
  	"go.uber.org/zap"
  	"bluebell/internal/infrastructure/logger"
  	"bluebell/internal/infrastructure/trace"
  )

  var tracerUser = trace.TracerForModule("service/user")

  type UserService struct {
  	userRepo     domain.UserRepository
  	socialRepo   domain.SocialRepository
  	tokenCache   domain.UserTokenCacheRepository
  	tokenService domain.TokenService
  }

  func NewUserService(
  	userRepo domain.UserRepository,
  	socialRepo domain.SocialRepository,
  	tokenCache domain.UserTokenCacheRepository,
  	tokenService domain.TokenService,
  ) *UserService {
  	return &UserService{
  		userRepo:     userRepo,
  		socialRepo:   socialRepo,
  		tokenCache:   tokenCache,
  		tokenService: tokenService,
  	}
  }

  func (s *UserService) SocialLogin(ctx context.Context, githubID, username, email, avatarURL string) (string, string, error) {
  	profile, err := s.socialRepo.GetProfileByGitHubID(ctx, githubID)
  	var userID int64

  	if err != nil {
  		userID = snowflake.GenID()
  		u := &entity.User{
  			UserID:   userID,
  			UserName: username,
  			Password: "",
  			Role:     entity.RoleUser,
  		}
  		if err := s.userRepo.CreateUser(ctx, u); err != nil {
  			return "", "", entity.Wrap(entity.ErrServerBusy, err)
  		}

  		profile = &entity.UserProfile{
  			UserID:    userID,
  			AvatarURL: avatarURL,
  			Bio:       "GitHub User",
  			GitHubID:  githubID,
  			GitHubURL: fmt.Sprintf("https://github.com/%s", username),
  		}
  		if err := s.socialRepo.SaveUserProfile(ctx, profile); err != nil {
  			return "", "", entity.Wrap(entity.ErrServerBusy, err)
  		}
  	} else {
  		userID = profile.UserID
  	}

  	aToken, rToken, err := s.tokenService.GenToken(userID)
  	if err != nil {
  		return "", "", entity.Wrap(entity.ErrServerBusy, err)
  	}

  	accessTokenExp := s.tokenService.GetAccessExpiry()
  	refreshTokenExp := s.tokenService.GetRefreshExpiry()
  	_ = s.tokenCache.SetUserToken(ctx, userID, aToken, rToken, accessTokenExp, refreshTokenExp)

  	return aToken, rToken, nil
  }

  func (s *UserService) SignUp(ctx context.Context, p *userreq.SignUpRequest) (err error) {
  	userID := snowflake.GenID()

  	hashedPassword, err := entity.HashPassword(p.Password)
  	if err != nil {
  		logger.WithContext(ctx).Error("entity.HashPassword failed", zap.Error(err))
  		return entity.Wrap(entity.ErrServerBusy, err)
  	}

  	u := &entity.User{
  		UserID:   userID,
  		UserName: p.Username,
  		Password: hashedPassword,
  		Role:     entity.RoleUser,
  	}

  	err = s.userRepo.CreateUser(ctx, u)
  	if err != nil {
  		if errors.Is(err, entity.ErrUserExist) {
  			return err
  		}
  		logger.WithContext(ctx).Error("userRepo.CreateUser failed",
  			zap.Int64("user_id", u.UserID),
  			zap.String("username", p.Username),
  			zap.Error(err))
  		return entity.Wrap(entity.ErrServerBusy, err)
  	}

  	return nil
  }

  func (s *UserService) Login(ctx context.Context, p *userreq.LoginRequest) (string, string, error) {
  	u, err := s.userRepo.GetUserByName(ctx, p.Username)
  	if err != nil {
  		if errors.Is(err, entity.ErrUserNotExist) {
  			return "", "", err
  		}
  		logger.WithContext(ctx).Error("userRepo.GetUserByName failed",
  			zap.String("username", p.Username),
  			zap.Error(err))
  		return "", "", entity.Wrap(entity.ErrServerBusy, err)
  	}

  	if !entity.CheckPassword(p.Password, u.Password) {
  		return "", "", entity.ErrInvalidPassword
  	}

  	aToken, rToken, err := s.tokenService.GenToken(u.UserID)
  	if err != nil {
  		logger.WithContext(ctx).Error("jwt.GenToken failed",
  			zap.Int64("user_id", u.UserID),
  			zap.Error(err))
  		return "", "", entity.Wrap(entity.ErrServerBusy, err)
  	}

  	accessTokenExp := s.tokenService.GetAccessExpiry()
  	refreshTokenExp := s.tokenService.GetRefreshExpiry()

  	err = s.tokenCache.SetUserToken(ctx, u.UserID, aToken, rToken, accessTokenExp, refreshTokenExp)
  	if err != nil {
  		logger.WithContext(ctx).Error("tokenCache.SetUserToken failed",
  			zap.Int64("user_id", u.UserID),
  			zap.Error(err))
  		return "", "", entity.Wrap(entity.ErrServerBusy, err)
  	}

  	return aToken, rToken, nil
  }

  func (s *UserService) RefreshToken(ctx context.Context, p *userreq.RefreshTokenRequest) (newAToken, newRToken string, err error) {
  	parts := strings.SplitN(p.RefreshToken, " ", 2)
  	if !(len(parts) == 2 && parts[0] == "Bearer") {
  		return "", "", fmt.Errorf("%w: Token格式错误", entity.ErrInvalidToken)
  	}

  	userID, err := s.tokenService.ParseToken(parts[1], "refresh")
  	if err != nil {
  		return "", "", entity.ErrInvalidToken
  	}

  	user, err := s.userRepo.GetUserByID(ctx, userID)
  	if err != nil || user == nil {
  		logger.WithContext(ctx).Error("userRepo.GetUserByID failed",
  			zap.Int64("user_id", userID),
  			zap.Error(err))
  		return "", "", entity.Wrap(entity.ErrServerBusy, err)
  	}

  	newAToken, newRToken, err = s.tokenService.GenToken(user.UserID)
  	if err != nil {
  		logger.WithContext(ctx).Error("jwt.GenToken failed in refresh",
  			zap.Int64("user_id", user.UserID),
  			zap.Error(err))
  		return "", "", entity.Wrap(entity.ErrServerBusy, err)
  	}

  	accessTokenExp := s.tokenService.GetAccessExpiry()
  	refreshTokenExp := s.tokenService.GetRefreshExpiry()

  	err = s.tokenCache.SetUserToken(ctx, user.UserID, newAToken, newRToken, accessTokenExp, refreshTokenExp)
  	if err != nil {
  		logger.WithContext(ctx).Error("tokenCache.SetUserToken failed in refresh",
  			zap.Int64("user_id", user.UserID),
  			zap.Error(err))
  		return "", "", entity.Wrap(entity.ErrServerBusy, err)
  	}

  	return newAToken, newRToken, nil
  }

  func (s *UserService) Logout(ctx context.Context, userID int64) error {
  	if err := s.tokenCache.DeleteUserToken(ctx, userID); err != nil {
  		logger.WithContext(ctx).Error("tokenCache.DeleteUserToken failed",
  			zap.Int64("user_id", userID),
  			zap.Error(err))
  		return entity.Wrap(entity.ErrServerBusy, err)
  	}

  	return nil
  }

  func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*entity.User, error) {
  	user, err := s.userRepo.GetUserByName(ctx, username)
  	if err != nil {
  		return nil, entity.Wrap(entity.ErrServerBusy, err)
  	}
  	return user, nil
  }

  func (s *UserService) UploadAvatar(ctx context.Context, userID int64, avatarURL string) error {
  	profile := &entity.UserProfile{
  		UserID:    userID,
  		AvatarURL: avatarURL,
  	}
  	return s.socialRepo.SaveUserProfile(ctx, profile)
  }

  func (s *UserService) GetAvatarURL(ctx context.Context, userID int64) (string, error) {
  	profile, err := s.socialRepo.GetUserProfile(ctx, userID)
  	if err != nil || profile == nil {
  		return "", nil
  	}
  	return profile.AvatarURL, nil
  }
  ```
- [ ] **Step 2: Move and Refactor `user_service_test.go`**
  Write the new `internal/application/user_service_test.go` containing:
  ```go
  package application

  import (
  	"bluebell/internal/config"
  	"bluebell/internal/domain/entity"
  	"bluebell/internal/infrastructure/snowflake"
  	userreq "bluebell/internal/application/dto/request/user"
  	"context"
  	"testing"
  	"time"
  )

  type MockUserRepository struct {
  	CheckUserExistFunc func(ctx context.Context, username string) error
  	CreateUserFunc      func(ctx context.Context, user *entity.User) error
  	VerifyUserFunc      func(ctx context.Context, user *entity.User) error
  	GetUserByIDFunc     func(ctx context.Context, uid int64) (*entity.User, error)
  	GetUsersByIDsFunc   func(ctx context.Context, ids []int64) ([]*entity.User, error)
  	GetUserRoleByIDFunc func(ctx context.Context, uid int64) (int, error)
  	GetUserByNameFunc   func(ctx context.Context, username string) (*entity.User, error)
  }

  func (m *MockUserRepository) CheckUserExist(ctx context.Context, username string) error {
  	return m.CheckUserExistFunc(ctx, username)
  }
  func (m *MockUserRepository) CreateUser(ctx context.Context, user *entity.User) error {
  	return m.CreateUserFunc(ctx, user)
  }
  func (m *MockUserRepository) VerifyUser(ctx context.Context, user *entity.User) error {
  	return m.VerifyUserFunc(ctx, user)
  }
  func (m *MockUserRepository) GetUserByID(ctx context.Context, uid int64) (*entity.User, error) {
  	return m.GetUserByIDFunc(ctx, uid)
  }
  func (m *MockUserRepository) GetUsersByIDs(ctx context.Context, ids []int64) ([]*entity.User, error) {
  	return m.GetUsersByIDsFunc(ctx, ids)
  }
  func (m *MockUserRepository) GetUserRoleByID(ctx context.Context, uid int64) (int, error) {
  	return m.GetUserRoleByIDFunc(ctx, uid)
  }
  func (m *MockUserRepository) GetUserByName(ctx context.Context, username string) (*entity.User, error) {
  	return m.GetUserByNameFunc(ctx, username)
  }

  type MockTokenCacheRepository struct {
  	SetUserTokenFunc       func(ctx context.Context, userID int64, aToken, rToken string, aExp, rExp time.Duration) error
  	GetUserAccessTokenFunc func(ctx context.Context, userID int64) (string, error)
  	GetUserRefreshTokenFunc func(ctx context.Context, userID int64) (string, error)
  	DeleteUserTokenFunc    func(ctx context.Context, userID int64) error
  }

  func (m *MockTokenCacheRepository) SetUserToken(ctx context.Context, userID int64, aToken, rToken string, aExp, rExp time.Duration) error {
  	return m.SetUserTokenFunc(ctx, userID, aToken, rToken, aExp, rExp)
  }
  func (m *MockTokenCacheRepository) GetUserAccessToken(ctx context.Context, userID int64) (string, error) {
  	return m.GetUserAccessTokenFunc(ctx, userID)
  }
  func (m *MockTokenCacheRepository) GetUserRefreshToken(ctx context.Context, userID int64) (string, error) {
  	return m.GetUserRefreshTokenFunc(ctx, userID)
  }
  func (m *MockTokenCacheRepository) DeleteUserToken(ctx context.Context, userID int64) error {
  	return m.DeleteUserTokenFunc(ctx, userID)
  }

  type MockTokenService struct {
  	GenTokenFunc func(userID int64) (string, string, error)
  	ParseTokenFunc func(tokenString string, expectedType string) (int64, error)
  }

  func (m *MockTokenService) GenToken(userID int64) (string, string, error) {
  	return m.GenTokenFunc(userID)
  }
  func (m *MockTokenService) ParseToken(tokenString string, expectedType string) (int64, error) {
  	return m.ParseTokenFunc(tokenString, expectedType)
  }
  func (m *MockTokenService) GetAccessExpiry() time.Duration {
  	return 2 * time.Hour
  }
  func (m *MockTokenService) GetRefreshExpiry() time.Duration {
  	return 7 * 24 * time.Hour
  }

  func TestUserService_SignUp(t *testing.T) {
  	snowflake.Init(&config.Config{
  		Snowflake: &config.SnowflakeConfig{
  			StartTime: 1775539200000,
  			MachineID: 1,
  		},
  	})

  	mockRepo := &MockUserRepository{
  		CheckUserExistFunc: func(ctx context.Context, username string) error {
  			return nil
  		},
  		CreateUserFunc: func(ctx context.Context, user *entity.User) error {
  			if user.UserName != "testuser" {
  				t.Errorf("expected username testuser, got %s", user.UserName)
  			}
  			if user.Password == "password123" {
  				t.Error("expected hashed password, got plain text")
  			}
  			return nil
  		},
  	}
  	mockTokenService := &MockTokenService{
  		GenTokenFunc: func(userID int64) (string, string, error) {
  			return "access", "refresh", nil
  		},
  	}
  	s := NewUserService(mockRepo, nil, nil, mockTokenService)
  	err := s.SignUp(context.Background(), &userreq.SignUpRequest{
  		Username:   "testuser",
  		Password:   "password123",
  		RePassword: "password123",
  	})
  	if err != nil {
  		t.Errorf("SignUp failed: %v", err)
  	}
  }
  ```
- [ ] **Step 3: Delete old files**
  Delete `internal/application/user/user_service.go` and `internal/application/user/user_service_test.go`.
- [ ] **Step 4: Commit**
  ```bash
  git add internal/application/user_service.go internal/application/user_service_test.go
  git rm internal/application/user/user_service.go internal/application/user/user_service_test.go
  git commit -m "refactor: flatten and concrete UserService"
  ```

---

### Task 6: Delete Interfaces file `interfaces.go`

**Files:**
- Delete: `internal/application/interfaces.go`

- [ ] **Step 1: Delete the file**
  Delete `internal/application/interfaces.go` since we no longer define service interfaces.
- [ ] **Step 2: Commit**
  ```bash
  git rm internal/application/interfaces.go
  git commit -m "refactor: remove service layer interfaces.go"
  ```

---

### Task 7: Update Handlers

**Files:**
- Modify: `internal/interfaces/http/handler/bookmark_handler/handler.go`
- Modify: `internal/interfaces/http/handler/community_handler/handler.go`
- Modify: `internal/interfaces/http/handler/post_handler/handler.go`
- Modify: `internal/interfaces/http/handler/search_handler/handler.go`
- Modify: `internal/interfaces/http/handler/social_handler/social.go`
- Modify: `internal/interfaces/http/handler/user_handler/handler.go`
- Modify: `internal/interfaces/http/handler/handler.go`

- [ ] **Step 1: Modify `bookmark_handler/handler.go`**
  Change lines 16-22 to:
  ```go
  type Handler struct {
  	bookmarkService *application.BookmarkService
  }

  func New(bookmarkService *application.BookmarkService) *Handler {
  	return &Handler{bookmarkService: bookmarkService}
  }
  ```
- [ ] **Step 2: Modify `community_handler/handler.go`**
  Change lines 21-31 to:
  ```go
  type Handler struct {
  	communityService *application.CommunityService
  }

  func New(communityService *application.CommunityService) *Handler {
  	return &Handler{
  		communityService: communityService,
  	}
  }
  ```
- [ ] **Step 3: Modify `post_handler/handler.go`**
  Change lines 21-27 to:
  ```go
  type Handler struct {
  	postService *application.PostService
  }

  func New(postService *application.PostService) *Handler {
  	return &Handler{postService: postService}
  }
  ```
- [ ] **Step 4: Modify `search_handler/handler.go`**
  Change lines 13-22 to:
  ```go
  type Handler struct {
  	postSvc *application.PostService
  }

  func New(postSvc *application.PostService) *Handler {
  	return &Handler{
  		postSvc: postSvc,
  	}
  }
  ```
- [ ] **Step 5: Modify `social_handler/social.go`**
  Change the handler struct and constructor to accept `*application.SocialService`.
- [ ] **Step 6: Modify `user_handler/handler.go`**
  Change the handler struct and constructor to accept `*application.UserService`.
- [ ] **Step 7: Modify `handler/handler.go`**
  Modify package imports to remove service subpackages, and update the signature of `NewProvider` to accept the concrete pointer types:
  ```go
  func NewProvider(
  	userService *application.UserService,
  	postService *application.PostService,
  	communityService *application.CommunityService,
  	socialService *application.SocialService,
  	bookmarkService *application.BookmarkService,
  ...
  ```
- [ ] **Step 8: Commit**
  ```bash
  git add internal/interfaces/http/handler/
  git commit -m "refactor: update handlers to depend on concrete service pointers"
  ```

---

### Task 8: Delete `di.go` and Wire in `cmd/server/main.go`

**Files:**
- Modify: `cmd/server/main.go`
- Delete: `internal/di/di.go`

- [ ] **Step 1: Modify `cmd/server/main.go`**
  Remove `"bluebell/internal/di"` import.
  Import `"bluebell/internal/application"`.
  Replace line 134:
  ```go
  svc := di.NewServices(dbRepos, cacheRepos, tokenService, searchRepo, searchSyncRepo, publisher, cfg)
  ```
  with direct service instantiation:
  ```go
  postService := application.NewPostService(dbRepos.Post, cacheRepos.PostCache, dbRepos.Vote, dbRepos.Remark, publisher, searchRepo, searchSyncRepo)
  communityService := application.NewCommunityService(dbRepos.Community, dbRepos.User)
  userService := application.NewUserService(dbRepos.User, dbRepos.Social, cacheRepos.TokenCache, tokenService)
  socialService := application.NewSocialService(dbRepos.Social, dbRepos.User, publisher)
  bookmarkService := application.NewBookmarkService(dbRepos.Bookmark, dbRepos.Post, dbRepos.User, dbRepos.Community)
  ```
  And update `hp := handler.NewProvider(...)` to pass these individual services:
  ```go
  hp := handler.NewProvider(
  	userService,
  	postService,
  	communityService,
  	socialService,
  	bookmarkService,
  	publisher,
  	gormDB,
  	rdb,
  	searchClient,
  	cfg.Upload.Dir,
  	sseHub,
  )
  ```
- [ ] **Step 2: Delete `internal/di/di.go` and its parent folder**
  Remove the `internal/di` directory.
- [ ] **Step 3: Commit**
  ```bash
  git rm internal/di/di.go
  git add cmd/server/main.go
  git commit -m "refactor: remove di.go and wire dependencies directly in main.go"
  ```

---

### Task 9: Verification

- [ ] **Step 1: Verify compilation**
  Run: `go build ./cmd/server`
  Expected: Compile successfully.
- [ ] **Step 2: Run all tests**
  Run: `go test ./...`
  Expected: All tests pass.
- [ ] **Step 3: Commit**
  ```bash
  git commit --allow-empty -m "refactor: successfully verified compilation and all tests pass"
  ```
