# DDD Conceptual Labeling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Explicitly label and explain all DDD (Domain-Driven Design) concepts within the backend codebase, providing clear definitions for Layers, Entities, Services, Repositories, and DTOs.

**Architecture:** We will add block comments to the header of key files and inline comments to specific structures/functions to identify their DDD role.

**Tech Stack:** Go, DDD

---

### Task 1: Label Domain Layer (The Heart of the System)

**Files:**
- Modify: `internal/domain/entity/user.go`
- Modify: `internal/domain/entity/post.go`
- Modify: `internal/domain/entity/errors.go`
- Modify: `internal/domain/repository.go`

- [ ] **Step 1: Label Entities in `user.go` and `post.go`**
Explicitly mark `User` and `Post` as **Domain Entities**. Explain that they have identity and encapsulate both data and behavior.

- [ ] **Step 2: Label Domain Errors in `errors.go`**
Mark this as the **Domain Error Definition**, explaining why business errors are defined here (to be used across all layers).

- [ ] **Step 3: Label Repository Interfaces in `repository.go`**
Mark these as **Repository Interfaces (Domain Layer)**. Explain that they define the "contracts" for data access without mentioning technology.

---

### Task 2: Label Application Layer (The Orchestrator)

**Files:**
- Modify: `internal/application/user/user_service.go`
- Modify: `internal/application/post/post_service.go`

- [ ] **Step 1: Label Application Services**
Mark `userServiceStruct` and `postServiceStruct` as **Application Services**. Explain they are "Waiters/Commanders" that orchestrate flow but don't contain rules.

---

### Task 3: Label Interface Layer (The Delivery Mechanism)

**Files:**
- Modify: `internal/interfaces/http/handler/user_handler.go`
- Modify: `internal/interfaces/http/dto/request/user/user.go` (or similar)

- [ ] **Step 1: Label Handlers**
Mark these as **Interface Adapters / Handlers**. Explain they handle protocol-specific logic (HTTP/JSON).

- [ ] **Step 2: Label DTOs**
Mark request/response structs as **Data Transfer Objects (DTOs)**. Explain they isolate the internal Domain from the external API contract.

---

### Task 4: Label Infrastructure Layer (The Technical Detail)

**Files:**
- Modify: `internal/infrastructure/persistence/mysql/userdb/user.go`
- Modify: `internal/infrastructure/persistence/mysql/model/user.go`

- [ ] **Step 1: Label Repository Implementations**
Mark these as **Infrastructure Implementations of Domain Repositories**.

- [ ] **Step 2: Label Database Models**
Mark `model.User` as **Infrastructure Models (Persistence Models)**. Explain the difference between a Model (database schema) and an Entity (business logic).

---

### Task 5: Final Review & Build
- [ ] **Step 1: Run build check**
```bash
go build ./...
```
