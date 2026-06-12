# Bluebell System Optimization Design Spec

**Date:** 2026-05-19
**Status:** Draft
**Focus:** DDD Refactoring, High-Concurrency Performance, Glassmorphism UI

---

## 1. Executive Summary
This design aims to modernize the Bluebell platform by strictly enforcing DDD principles, optimizing the voting system to handle 10k+ TPS via local buffering, and implementing a premium "Glassmorphism" UI style.

## 2. Backend Architecture: DDD Decoupling
Currently, business logic (like password hashing) is mixed with infrastructure (GORM hooks). We will shift this to a rich domain model.

### Key Changes:
- **Domain Layer (`internal/domain/entity`)**:
    - `User` entity will own `HashPassword` and `CheckPassword` methods.
    - All validation logic will reside in `Validate()` methods within entities.
- **Infrastructure Layer (`internal/infrastructure/persistence`)**:
    - Remove all `BeforeCreate` / `AfterUpdate` hooks from GORM models.
    - Repositories must perform 100% mapping between ORM models and Domain entities.
- **Application Layer (`internal/application`)**:
    - Orchestrate entities and repositories without leaking infrastructure details.

## 3. Performance: High-Concurrency Vote Buffering
To mitigate Redis pressure during 10k+ TPS bursts, we will implement a local aggregation layer in the Go service.

### Technical Design:
- **Buffer Mechanism**: A thread-safe `Sync.Map` or a worker pool with a buffer channel will aggregate votes per `PostID` over a 100ms window.
- **Batching**: Instead of 1 Redis call per vote, the service will send 1 batch update (using Redis Pipeline or a bulk Lua script) per window.
- **Reliability**: Use a graceful shutdown hook to flush remaining buffered votes to Redis before the process exits.

## 4. UI/UX: Glassmorphism Modernization
The frontend will be refactored to use a modern, translucent aesthetic.

### Visual Style:
- **Backgrounds**: Multi-layered radial gradients with `backdrop-filter: blur(12px)`.
- **Cards**: Semi-transparent white backgrounds (`rgba(255, 255, 255, 0.4)`) with subtle borders.
- **Typography**: Inter/System Sans-Serif with high contrast and variable weights.
- **Interactions**: Immediate visual feedback for votes using local state before API confirmation.

## 5. Implementation Phases
1. **Phase 1**: DDD Refactoring (User & Post domains).
2. **Phase 2**: High-Concurrency Vote Buffer implementation.
3. **Phase 3**: Glassmorphism UI Theme application.
4. **Phase 4**: Load testing and performance validation.

---

## 6. Self-Review
- **Ambiguity**: The buffering window is set to 100ms; this balances real-time feel with performance.
- **Consistency**: All layers follow the established DDD pattern in `docs/ddd_design_principles.md`.
- **Scope**: Focused on core hot paths (User/Vote/Home).
