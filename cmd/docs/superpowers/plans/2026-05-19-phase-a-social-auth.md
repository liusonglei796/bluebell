# Phase A: Social Infrastructure & OAuth Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the backend and frontend for User Profiles, Event-Driven Activity Feed, Follow System, and GitHub OAuth Framework.

**Architecture:** Event-driven using RabbitMQ for activities. Layered DDD for profiles and follows. Clean GitHub OAuth integration.

**Tech Stack:** Go, GORM, RabbitMQ, Vue 3, Lucide Icons.

---

### Task 1: Database Models & Repositories

**Files:**
- Create: `internal/infrastructure/persistence/mysql/model/social.go`
- Create: `internal/infrastructure/persistence/mysql/socialdb/social.go`
- Modify: `internal/domain/repository.go`

- [ ] **Step 1: Define `UserProfile`, `Follow`, and `Activity` models**
- [ ] **Step 2: Implement `SocialRepository` interface and GORM implementation**
- [ ] **Step 3: Update `AutoMigrate` in `internal/infrastructure/persistence/mysql/init.go`**
- [ ] **Step 4: Commit**

---

### Task 2: Activity Event System (MQ)

**Files:**
- Create: `internal/infrastructure/mq/activity_consumer.go`
- Modify: `internal/infrastructure/mq/publisher.go`
- Modify: `internal/application/post/post_service.go`

- [ ] **Step 1: Define `ActivityMessage` in `internal/infrastructure/mq/message.go`**
- [ ] **Step 2: Implement `PublishActivity` in `publisher.go`**
- [ ] **Step 3: Create `ActivityConsumer` to save messages to the `Activity` table**
- [ ] **Step 4: Emit activities during `CreatePost`, `VoteForPost`, etc.**
- [ ] **Step 5: Commit**

---

### Task 3: Social & Profile API (Backend)

**Files:**
- Create: `internal/application/social/social_service.go`
- Create: `internal/interfaces/http/handler/social_handler/social.go`
- Modify: `internal/interfaces/http/router/router.go`

- [ ] **Step 1: Implement `SocialService` (GetProfile, GetActivities, Follow/Unfollow)**
- [ ] **Step 2: Implement `SocialHandler` for the new endpoints**
- [ ] **Step 3: Wire routes in `router.go`**
- [ ] **Step 4: Commit**

---

### Task 4: GitHub OAuth Framework

**Files:**
- Create: `internal/infrastructure/jwt/oauth.go`
- Modify: `internal/interfaces/http/handler/user_handler/user.go`

- [ ] **Step 1: Implement OAuth2 config and Redirect/Callback logic**
- [ ] **Step 2: Add mock handler if `CLIENT_ID` is missing**
- [ ] **Step 3: Commit**

---

### Task 5: Frontend: User Profile & Activities Page

**Files:**
- Create: `frontend/src/pages/Profile.vue`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/components/PostCard.vue`

- [ ] **Step 1: Create the `Profile.vue` page with Glassmorphism styles**
- [ ] **Step 2: Implement Activity Feed component in Profile page**
- [ ] **Step 3: Link user avatars/names to the Profile page**
- [ ] **Step 4: Commit**
