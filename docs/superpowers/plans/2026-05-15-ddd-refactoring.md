# DDD Domain Layer Refactoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor the codebase from an anemic domain model to a rich domain model by下沉 business logic into entities and ensuring strict separation between domain and infrastructure layers.

**Architecture:** DDD (Domain-Driven Design). Business logic resides in `internal/domain/entity`. Repository implementations in `internal/infrastructure/persistence` handle mapping between entities and ORM models. Application services orchestrate domain entities and infrastructure.

**Tech Stack:** Go, GORM, Bcrypt.

---

### Task 1: Enrich Domain Entities with Business Logic

**Files:**
- Modify: `internal/domain/entity/user.go`
- Modify: `internal/domain/entity/post.go`
- Modify: `internal/domain/entity/vote.go`
- Modify: `internal/domain/entity/remark.go`
- Create: `internal/domain/entity/entity_test.go`

- [ ] **Step 1: Add missing methods to entity files**

**internal/domain/entity/user.go**:
```go
// IsAdmin() and HashPassword(), CheckPassword() are already there, but let's ensure they are correct.
// Add validation or other logic if needed.
```

**internal/domain/entity/post.go**:
Already has `IsValid`, `CanBeDeletedBy`, etc. Ensure they are complete.

**internal/domain/entity/vote.go**:
Already has `Validate`, `ScoreDelta`.

**internal/domain/entity/remark.go**:
Already has `Validate`.

- [ ] **Step 2: Write unit tests for entity methods in `internal/domain/entity/entity_test.go`**

```go
package entity

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestUser_IsAdmin(t *testing.T) {
	u := &User{Role: RoleAdmin}
	assert.True(t, u.IsAdmin())
	u.Role = RoleUser
	assert.False(t, u.IsAdmin())
}

func TestPost_CanBeDeletedBy(t *testing.T) {
	p := &Post{AuthorID: 123}
	assert.Nil(t, p.CanBeDeletedBy(123))
	assert.Equal(t, ErrForbidden, p.CanBeDeletedBy(456))
}

func TestVote_Validate(t *testing.T) {
	v := &Vote{Direction: VoteUp}
	assert.Nil(t, v.Validate())
	v.Direction = 2
	assert.Equal(t, ErrInvalidParam, v.Validate())
}
```

- [ ] **Step 3: Run entity tests**

Run: `go test ./internal/domain/entity/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/domain/entity/
git commit -m "feat(domain): enrich entities with business logic and add unit tests"
```

---

### Task 2: Refactor Infrastructure Layer - Models and Repositories

**Files:**
- Modify: `internal/infrastructure/persistence/mysql/model/user.go`
- Modify: `internal/infrastructure/persistence/mysql/userdb/user.go`
- Modify: `internal/infrastructure/persistence/mysql/postdb/post.go`
- Modify: `internal/infrastructure/persistence/mysql/communitydb/community.go`
- Modify: `internal/infrastructure/persistence/mysql/postdb/remark.go`

- [ ] **Step 1: Remove GORM hooks from `model/user.go`**

Remove `BeforeCreate` from `internal/infrastructure/persistence/mysql/model/user.go`.

- [ ] **Step 2: Update `userdb/user.go` to handle `entity.User`**

Update all methods to accept/return `entity.User` and map them to `model.User` internally.

- [ ] **Step 3: Update `postdb/post.go` to handle `entity.Post`**

Ensure it returns `entity.Post` and maps `model.Post` to `entity.Post`.

- [ ] **Step 4: Update `communitydb/community.go` and `postdb/remark.go`**

Apply similar mapping logic.

- [ ] **Step 5: Verify build**

Run: `go build ./internal/infrastructure/persistence/mysql/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/infrastructure/persistence/mysql/
git commit -m "refactor(infra): decouple repositories from models and remove GORM hooks"
```

---

### Task 3: Refactor Application Layer - Services

**Files:**
- Modify: `internal/application/user/user_service.go`
- Modify: `internal/application/post/post_service.go`
- Modify: `internal/application/community/community_service.go`
- Modify: `internal/application/interfaces.go`

- [ ] **Step 1: Update `UserService` to use `entity.User` methods**

Use `entity.HashPassword()` during `SignUp`. Use `entity.CheckPassword()` during `Login`.

- [ ] **Step 2: Update `PostService` to use `entity.Post` and `entity.Vote` methods**

Use `post.CanBeDeletedBy(userID)` in `DeletePost`.
Use `post.IsValid()` for early returns.
Use `vote.Validate()` in `VoteForPost`.

- [ ] **Step 3: Update `CommunityService` to use `entity.User.IsAdmin()`**

Replace `role != model.RoleAdmin` with `user.IsAdmin()`.

- [ ] **Step 4: Update `interfaces.go`**

Update `UserService`, `PostService`, etc., signatures to use `entity` types.

- [ ] **Step 5: Verify build**

Run: `go build ./internal/application/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/application/
git commit -m "refactor(app): slim down services by leveraging rich domain entities"
```

---

### Task 4: Refactor MQ Layer and Final Verification

**Files:**
- Modify: `internal/infrastructure/mq/vote_consumer.go`

- [ ] **Step 1: Update `VoteConsumer` to use `entity.Vote.Validate()`**

Replace manual direction check with `vote.Validate()`.

- [ ] **Step 2: Run all tests**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 3: Final Build check**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/infrastructure/mq/
git commit -m "refactor(mq): use entity validation in vote consumer"
```
