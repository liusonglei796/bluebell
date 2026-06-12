# Design Document: DDD Refactoring - Flattening Application Layer & Removing Interfaces

This document outlines the design and implementation details for refactoring the `bluebell` project to strictly follow the DDD best practices described in the reference Notion docs.

## 1. Objectives

- **Remove service-to-handler interfaces**: Delete `internal/application/interfaces.go`. Direct handlers to depend on concrete application service struct pointers.
- **Flatten Application Layer**: Move all service implementations (`bookmark`, `community`, `post`, `social`, `user`) directly under `internal/application/` to simplify imports and prevent stuttering packages.
- **Eliminate `di.go`**: Remove the `internal/di` package entirely. Perform all dependency injection and composition directly in the Composition Root at `cmd/server/main.go`.
- **Align with DDD reference docs**: Follow the conventions demonstrated in the `go-ddd-mvp` project.

---

## 2. Architecture & File Relocations

### 2.1 File Changes
```
[Delete] internal/application/interfaces.go
[Delete] internal/di/di.go
[Delete] internal/di/

[Move] internal/application/bookmark/bookmark_service.go  -> internal/application/bookmark_service.go
[Move] internal/application/community/community_service.go -> internal/application/community_service.go
[Move] internal/application/post/post_service.go           -> internal/application/post_service.go
[Move] internal/application/social/social_service.go         -> internal/application/social_service.go
[Move] internal/application/user/user_service.go           -> internal/application/user_service.go
[Move] internal/application/user/user_service_test.go      -> internal/application/user_service_test.go
```

### 2.2 Package Re-alignment
All moved service files will belong to `package application`. All imports of `bluebell/internal/application/user`, `bluebell/internal/application/post`, etc., will be removed, and imports of `bluebell/internal/application` will be used instead.

---

## 3. Detailed Struct and Constructor Modifications

We will rename the unexported structs and change constructors to return concrete pointers:

| Original Struct | New Struct | Original Constructor | New Constructor Signature |
| :--- | :--- | :--- | :--- |
| `bookmarkService` | `BookmarkService` | `NewBookmarkService(...) application.BookmarkService` | `NewBookmarkService(...) *BookmarkService` |
| `communityServiceStruct` | `CommunityService` | `NewCommunityService(...) application.CommunityService` | `NewCommunityService(...) *CommunityService` |
| `postServiceStruct` | `PostService` | `NewPostService(...) application.PostService` | `NewPostService(...) *PostService` |
| `socialService` | `SocialService` | `NewSocialService(...) application.SocialService` | `NewSocialService(...) *SocialService` |
| `userServiceStruct` | `UserService` | `NewUserService(...) application.UserService` | `NewUserService(...) *UserService` |

---

## 4. Composition Root (`cmd/server/main.go`)

We will wire all services directly in `main.go` using concrete types:

```go
// cmd/server/main.go
dbRepos := database.NewRepositories(gormDB)
cacheRepos := redisrepo.NewRepositories(rdb)
tokenService := jwt.NewJWTService(cfg)

searchRepo := es.NewPostSearch(searchClient)
searchSyncRepo := es.NewPostSync(searchClient)

// Wire Application Services directly
postService := application.NewPostService(dbRepos.Post, cacheRepos.PostCache, dbRepos.Vote, dbRepos.Remark, publisher, searchRepo, searchSyncRepo)
communityService := application.NewCommunityService(dbRepos.Community, dbRepos.User)
userService := application.NewUserService(dbRepos.User, dbRepos.Social, cacheRepos.TokenCache, tokenService)
socialService := application.NewSocialService(dbRepos.Social, dbRepos.User, publisher)
bookmarkService := application.NewBookmarkService(dbRepos.Bookmark, dbRepos.Post, dbRepos.User, dbRepos.Community)

// Instantiate HTTP Handler Provider
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

---

## 5. User Interface (HTTP Handlers) Adjustments

Each handler will hold concrete pointers to the respective services instead of interfaces:

```go
// Example: community_handler/handler.go
type Handler struct {
    communityService *application.CommunityService
}

func New(communityService *application.CommunityService) *Handler {
    return &Handler{
        communityService: communityService,
    }
}
```

This pattern will be applied uniformly across `user_handler`, `post_handler`, `community_handler`, `search_handler`, `social_handler`, and `bookmark_handler`.
