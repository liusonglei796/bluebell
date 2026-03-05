# Bluebell Project Context

This document provides essential information for Gemini CLI to understand and interact with the Bluebell project effectively.

## Project Overview
**Bluebell** is a high-performance community forum backend (similar to Reddit) built with Go. It follows **DDD (Domain-Driven Design)** and **Clean Architecture** principles to ensure maintainability and scalability.

### Core Tech Stack
- **Framework:** [Gin](https://github.com/gin-gonic/gin) (HTTP Web Framework)
- **Database:** MySQL with [GORM](https://gorm.io) (ORM)
- **Cache/Ranking:** Redis with [go-redis](https://github.com/redis/go-redis)
- **Logging:** [Zap](https://github.com/uber-go/zap) with Lumberjack for rotation
- **Configuration:** [Viper](https://github.com/spf13/viper)
- **Authentication:** JWT (Access Token + Refresh Token)
- **ID Generation:** Snowflake (Distributed ID)
- **API Documentation:** Swagger (Swaggo)

### Architecture Layers (`internal/`)
- **`handler/`**: HTTP layer. Handles request parsing, validation, and calls services.
- **`service/`**: Business logic layer. Orchestrates domain logic and interacts with repositories.
- **`domain/repository/`**: Interface definitions for data access (Dependency Inversion).
- **`dao/`**: Data Access Object layer. Implements repository interfaces for MySQL (GORM) and Redis.
- **`model/`**: GORM entities/models.
- **`dto/`**: Data Transfer Objects for requests and responses.
- **`infrastructure/`**: Middleware (Auth, RateLimit, Timeout) and core utilities.

## Building and Running

### Prerequisites
- **Go:** 1.19+ (Project `go.mod` specifies 1.26.0)
- **MySQL:** 5.7+
- **Redis:** 5.0+

### Key Commands
- **Run Application:**
  ```bash
  go run ./cmd/bluebell/ -conf ./config.yaml
  ```
- **Hot Reload (Development):**
  ```bash
  air
  ```
- **Build Binary:**
  ```bash
  make build
  ```
- **Generate Swagger Docs:**
  ```bash
  swag init -g cmd/bluebell/main.go
  ```

## Development Conventions

### 1. Dependency Injection (DI)
The project uses manual constructor-based DI in `cmd/bluebell/main.go`. When adding new services or repositories, update the assembly logic there.

### 2. Error Handling
- Use the custom `pkg/errorx` package for structured error handling.
- Prefer `errorx.Wrap(err, code, msg)` to provide context and business error codes.
- Business error codes are defined in `pkg/errorx`.

### 3. Request & Response
- **Requests:** Use structs in `internal/dto/request/` with `binding` tags for validation.
- **Responses:** Use `handler.ResponseSuccess(c, data)` or `handler.ResponseError(c, err)` to maintain a consistent JSON structure:
  ```json
  {
    "code": 1000,
    "msg": "success",
    "data": { ... }
  }
  ```

### 4. Context & Identity
- Authenticated user ID is stored in Gin context using `handler.CtxUserIDKey` ("userID").
- Retrieve it using `handler.GetCurrentUser(c)`.

### 5. Coding Style
- Follow standard Go naming conventions (CamelCase).
- Repositories should return interfaces defined in `internal/domain/repository/`.
- Use `zap.L()` for logging throughout the `internal/` package.

## Project Structure (Core)
- `cmd/bluebell/`: Application entry point and DI container.
- `internal/`: Private application code.
- `pkg/`: Public library code (can be imported by other projects).
- `docs/`: Generated Swagger documentation.
- `教学文档/`: Comprehensive project documentation (18 chapters).
