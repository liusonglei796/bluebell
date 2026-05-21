# Backend DDD Commenting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add detailed, "why"-focused architectural and strategic comments to the backend codebase, aligning with the project's Domain-Driven Design (DDD) principles.

**Architecture:** We will systematically traverse the layers of the DDD architecture (Domain, Application, Infrastructure, Interface). In each layer, we will add comments that explain *why* the code is placed there, *why* certain dependencies exist (or don't exist), and the overarching responsibilities.

**Tech Stack:** Go, DDD

---

### Task 1: Comment Domain Layer (Entities & Interfaces)

**Files:**
- Modify: `internal/domain/entity/user.go`
- Modify: `internal/domain/entity/post.go`
- Modify: `internal/domain/repository.go`

- [ ] **Step 1: Add "Why" comments to `user.go`**
Add comments explaining why `HashPassword` and role definitions are in the Domain layer (they are core business rules that do not depend on external frameworks or databases).

- [ ] **Step 2: Add "Why" comments to `post.go`**
Explain why post state transitions and business constraints reside here.

- [ ] **Step 3: Add "Why" comments to `repository.go`**
Explain Dependency Inversion: why interfaces are defined in the Domain layer but implemented in Infrastructure.

- [ ] **Step 4: Commit Domain Comments**

---

### Task 2: Comment Application Layer (Services)

**Files:**
- Modify: `internal/application/user/user_service.go`
- Modify: `internal/application/post/post_service.go`

- [ ] **Step 1: Add "Why" comments to `user_service.go`**
Explain why `userServiceStruct` orchestrates flows (e.g., in `SocialLogin`, it fetches profile, creates user if missing, generates JWT) but pushes actual hashing to the domain layer. 

- [ ] **Step 2: Add "Why" comments to `post_service.go`**
Explain its role as an orchestrator: calling repositories and handling transactions without containing core business logic.

- [ ] **Step 3: Commit Application Comments**

---

### Task 3: Comment Infrastructure & Interface Layers

**Files:**
- Modify: `internal/interfaces/http/router/router.go`
- Modify: `internal/infrastructure/persistence/mysql/user_repo.go` (or similar)

- [ ] **Step 1: Add "Why" comments to `router.go`**
Explain why the HTTP delivery mechanism is isolated here, preventing HTTP context (`*gin.Context`) from leaking into the Application layer.

- [ ] **Step 2: Add "Why" comments to infrastructure persistence**
Explain that infrastructure depends on the Domain layer to implement interfaces, keeping tech stacks (MySQL/Redis) decoupled from business logic.

- [ ] **Step 3: Commit Infrastructure Comments**

