# Bluebell

**Project Context for AI Assistants**

Bluebell is a high-performance community forum backend written in Go, inspired by Reddit. It features a three-tier architecture (Controller-Logic-DAO) and supports user authentication, community management, post creation, and a voting system.

## 📂 Project Overview

*   **Type:** Go Web Backend
*   **Architecture:** 3-Tier (Controller -> Logic -> DAO)
*   **Framework:** Gin
*   **Database:** MySQL (Storage), Redis (Caching & Voting/Ranking)
*   **Auth:** JWT (Access & Refresh Tokens)
*   **Config:** Viper (`config.yaml`)
*   **Logging:** Zap

### Directory Structure

*   `controller/`: HTTP handlers, request validation.
*   `logic/`: Business logic, orchestrating calls between DAO and other services.
*   `dao/`: Data Access Objects for MySQL and Redis.
*   `models/`: Data structures (structs) for DB tables and API request/response.
*   `routers/`: Route definitions and middleware registration.
*   `middlewares/`: Custom middlewares (e.g., JWT Auth, Logger).
*   `pkg/`: Shared utilities (Snowflake ID, JWT, Error codes).
*   `settings/`: Configuration loading.
*   `教学文档/`: Comprehensive tutorial documentation (18 chapters, in Chinese) explaining the build process.

## 🛠 Building and Running

The project uses a `Makefile` for common tasks.

*   **Run (Dev):** `make run` (or `go run main.go`)
*   **Hot Reload (Dev):** `air` (Requires `air` installed)
*   **Build:** `make build` (Produces binary `bluebell`)
*   **Format & Vet:** `make gotool`
*   **Clean:** `make clean`

### Configuration

*   **File:** `config.yaml`
*   **Key Sections:** `app` (port, mode), `mysql` (dsn), `redis` (addr), `log` (level, file), `snowflake` (machine_id).

## 💻 Development Conventions

*   **Entry Point:** `main.go` handles initialization (Settings -> Snowflake -> Logger -> MySQL -> Redis -> Validator -> Router -> Server).
*   **API Design:** RESTful.
    *   **Public:** `/api/v1/signup`, `/api/v1/login`, `/api/v1/refresh_token`
    *   **Protected:** Requires `Authorization: Bearer <token>` header. Includes Community, Post, and Vote endpoints.
*   **Error Handling:** Custom error codes in `pkg/errno` and `controller/code.go`. Unified response format (`code`, `msg`, `data`).
*   **Comments:** Public functions must have comments. Swagger annotations are used for API documentation.
*   **Documentation:** Swagger docs generated at `http://127.0.0.1:8080/swagger/index.html`.

## 🔍 Key Files

*   `main.go`: Application bootstrap and graceful shutdown logic.
*   `routers/routers.go`: API route definitions.
*   `config.yaml`: Configuration settings (ensure you check this for DB creds).
*   `README.md`: Detailed project introduction (in Chinese).
*   `Makefile`: Build commands.
