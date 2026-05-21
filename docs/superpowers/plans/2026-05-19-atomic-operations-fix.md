# Backend Atomic Operations & Error Refinement Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate "Check-then-Write" race conditions and refine error handling across the backend by using atomic SQL operations (Upsert) and specific error mapping.

**Architecture:** We will move decision-making logic from the Application layer down to the Infrastructure (MySQL) layer using `ON DUPLICATE KEY UPDATE` or `ON CONFLICT` clauses. We will also introduce error translation to map technical database errors (like Duplicate Entry 1062) to Domain-specific errors.

**Tech Stack:** Go, GORM, MySQL

---

### Task 1: Refine User Repository & SignUp Flow

**Files:**
- Modify: `internal/infrastructure/persistence/mysql/userdb/user.go`
- Modify: `internal/application/user/user_service.go`

- [ ] **Step 1: Update `CreateUser` to handle unique constraint conflicts**
Instead of just returning a raw error, check if the error is a MySQL 1062 (Duplicate Entry) and return `entity.ErrUserExist`.

- [ ] **Step 2: Streamline `SignUp` in Application Layer**
Remove the redundant `CheckUserExist` call. Rely on `CreateUser` to perform the atomic check-and-insert.

- [ ] **Step 3: Verify SignUp concurrency**
Ensure that two identical registration attempts result in a clear "User exists" error rather than "Server busy".

---

### Task 2: Atomic User Profile Updates

**Files:**
- Modify: `internal/infrastructure/persistence/mysql/socialdb/social.go`

- [ ] **Step 1: Convert `SaveUserProfile` to Upsert**
Use `Clauses(clause.OnConflict{...})` to handle profile creation and updates in a single atomic SQL call.

---

### Task 3: Graceful Social Follow Operations

**Files:**
- Modify: `internal/infrastructure/persistence/mysql/socialdb/social.go`

- [ ] **Step 1: Update `FollowUser` to ignore duplicate follows**
Use `OnConflict` with `DoNothing: true` to prevent errors when a user follows someone they are already following.

- [ ] **Step 2: Update `UnfollowUser` to handle non-existent relations gracefully**
Ensure `RowsAffected` is used to determine if a delete actually happened, rather than just checking `err`.

---

### Task 4: Global Error Refinement Audit

**Files:**
- Modify: `internal/infrastructure/persistence/mysql/postdb/post.go`
- Modify: `internal/infrastructure/persistence/mysql/communitydb/community.go`

- [ ] **Step 1: Audit all `Create` methods**
Ensure all repositories wrap MySQL errors correctly and convert technical violations into domain errors (e.g., `ErrDuplicateCommunityName`).

- [ ] **Step 2: Run final build and validation**
```bash
go build ./...
go test ./...
```
