# Design Spec: Re-applying "Why"-Focused Comments to Backend

**Goal:** Restore descriptive comments explaining the architectural "why" across multiple layers of the backend (Domain, Application, Interface, Infrastructure) to improve maintainability and alignment with DDD principles.

**Scope:** 7 files in the backend.

## 1. Domain Layer

### internal/domain/entity/user.go
- **Top Comment:** Explain why the Domain Layer holds core business rules like `bcryptCost` and role constants.
- **IsAdmin/HashPassword:** Note that these are domain rules isolated from external frameworks.

### internal/domain/entity/post.go
- **Top Comment:** Explain why post status constants are in the domain.
- **Validate/CanBeDeletedBy:** Note that these are core business constraints and permissions consistent across the system.

### internal/domain/repository.go
- **Top Comment:** Explain Dependency Inversion (Interfaces in Domain, Implementation in Infrastructure).
- **Cache Repositories:** Explain why technical needs (caching) are abstracted as domain requirements.

## 2. Application Layer

### internal/application/user/user_service.go
- **Package Comment:** Explain Application layer's role as an "orchestrator" (commander).
- **Struct/Methods:** Explain how they coordinate between repositories and entities without containing core logic.

### internal/application/post/post_service.go
- **Package Comment:** Explain Application layer's role as an "orchestrator".
- **Struct Comment:** Explain why it holds multiple repositories/clients to fulfill complex use cases.

## 3. Interface Layer

### internal/interfaces/http/router/router.go
- **Package Comment:** Explain Interface layer's role in isolating communication protocols (HTTP/Gin) from business logic.
- **NewRouter:** Note the use of dependency injection for handlers.

## 4. Infrastructure Layer

### internal/infrastructure/persistence/mysql/userdb/user.go
- **Package Comment:** Explain Infrastructure layer's role in implementing domain interfaces and handling tech-specific details.
- **fromModelUser:** Note the necessity of model-to-entity conversion to keep the domain pure.

## Implementation Strategy
- Use `replace` for surgical edits.
- Ensure comments follow Go documentation standards.
