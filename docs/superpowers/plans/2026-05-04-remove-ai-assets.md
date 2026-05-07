# Remove AI-Related Assets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Completely remove all AI-related code, configurations, dependencies, and hidden assets from the project while maintaining core functionality.

**Architecture:** Surgical removal of AI auditor injection from the service layer, cleanup of RabbitMQ audit infrastructure, and deletion of AI-specific directories and files.

**Tech Stack:** Go, RabbitMQ, Git.

---

### Task 1: Clean Up Configuration and Hidden Assets

**Files:**
- Modify: `config.yaml`
- Modify: `config.docker.toml`
- Modify: `internal/config/config.go`
- Delete: `AGENTS.md`, `QWEN.md`
- Delete: `.qwen/`, `.playwright-mcp/`, `.superpowers/`, `.omg/`, `.sisyphus/`

- [ ] **Step 1: Remove AI sections from config files**
  - Remove `Modelscope` and `ai_audit` sections from `config.yaml` and `config.docker.toml`.

- [ ] **Step 2: Update config struct in `internal/config/config.go`**
  - Delete `aiAuditConfig` struct.
  - Remove `AIAudit *aiAuditConfig` from `Config` struct.

- [ ] **Step 3: Delete AI-related documentation and hidden directories**
  - Run: `rm AGENTS.md, QWEN.md`
  - Run: `rm -rf .qwen .playwright-mcp .superpowers .omg .sisyphus`

- [ ] **Step 4: Commit**
  ```bash
  git add config.yaml config.docker.toml internal/config/config.go
  git rm AGENTS.md QWEN.md
  # Note: hidden dirs might not be tracked, but if they are:
  git rm -r .qwen .playwright-mcp .superpowers .omg .sisyphus
  git commit -m "chore: remove AI configurations and hidden assets"
  ```

---

### Task 2: Remove AI Auditor from Service Layer

**Files:**
- Modify: `internal/service/services.go`
- Modify: `internal/service/postsvc/post_service.go`
- Delete: `internal/infrastructure/ai/`

- [ ] **Step 1: Remove Auditor from `internal/service/services.go`**
  - Remove `"bluebell/internal/infrastructure/ai"` import.
  - Remove `auditor *ai.Auditor` from `NewServices` parameters.
  - Remove `auditor` from `postsvc.NewPostService` call.

- [ ] **Step 2: Remove Auditor from `internal/service/postsvc/post_service.go`**
  - Remove `"bluebell/internal/infrastructure/ai"` import.
  - Remove `auditor *ai.Auditor` field from `postServiceStruct`.
  - Update `NewPostService` to remove the `auditor` parameter and initialization.
  - Remove auditing logic from `CreatePost` and `RemarkPost`.
  - Delete `UpdatePostStatus` and `DeleteRemark` methods.

- [ ] **Step 3: Delete infrastructure AI directory**
  - Run: `rm -rf internal/infrastructure/ai`

- [ ] **Step 4: Commit**
  ```bash
  git add internal/service/services.go internal/service/postsvc/post_service.go
  git rm -r internal/infrastructure/ai
  git commit -m "feat: remove AI auditor from service layer"
  ```

---

### Task 3: Clean Up RabbitMQ Audit Infrastructure

**Files:**
- Modify: `internal/service/mq/connection.go`
- Modify: `internal/service/mq/publisher.go`
- Modify: `internal/service/mq/message.go`
- Delete: `internal/service/mq/ai_consumer.go`

- [ ] **Step 1: Remove Audit Exchange/Queue from `internal/service/mq/connection.go`**
  - Remove `ExchangeAudit`, `QueueAudit`, `RoutingKeyAuditPost`, `RoutingKeyAuditRemark` constants.
  - Remove `ExchangeAudit` declaration from `DeclareExchanges`.
  - Remove `QueueAudit` declaration and bindings from `DeclareQueues`.

- [ ] **Step 2: Remove Audit method from `internal/service/mq/publisher.go`**
  - Delete `PublishAudit` method.

- [ ] **Step 3: Remove Audit message from `internal/service/mq/message.go`**
  - Delete `AuditMessage` struct.

- [ ] **Step 4: Delete AI Consumer file**
  - Run: `rm internal/service/mq/ai_consumer.go`

- [ ] **Step 5: Commit**
  ```bash
  git add internal/service/mq/connection.go internal/service/mq/publisher.go internal/service/mq/message.go
  git rm internal/service/mq/ai_consumer.go
  git commit -m "chore: remove AI-related RabbitMQ infrastructure"
  ```

---

### Task 4: Clean Up Entry Point and Dependencies

**Files:**
- Modify: `cmd/bluebell/main.go`
- Modify: `go.mod`

- [ ] **Step 1: Clean up `cmd/bluebell/main.go`**
  - Remove `"bluebell/internal/infrastructure/ai"` import.
  - Remove AI Auditor initialization block.
  - Remove `AuditConsumer` setup and its callback logic within the `if conn != nil` block.
  - Ensure `services` and `handlerProvider` calls match updated signatures.

- [ ] **Step 2: Clean up `go.mod`**
  - Run: `go mod tidy`

- [ ] **Step 3: Final Build Verification**
  - Run: `go build ./cmd/bluebell/`
  - Expected: Build SUCCESS without errors.

- [ ] **Step 4: Commit and Push**
  ```bash
  git add cmd/bluebell/main.go go.mod go.sum
  git commit -m "chore: final AI cleanup and dependency tidy"
  git push
  ```
