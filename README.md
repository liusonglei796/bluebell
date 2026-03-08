# Bluebell

A modern Go web application built with Gin, featuring MySQL and Redis integration for scalable backend services.

## Overview

Bluebell is a Go-based web backend service that demonstrates best practices in:
- RESTful API design with Gin framework
- Database integration with GORM and MySQL
- Caching layer with Redis
- Configuration management with Viper
- JWT authentication
- Input validation and error handling
- Containerized deployment with Docker

## Tech Stack

- **Language**: Go 1.26.0
- **Web Framework**: Gin
- **Database**: MySQL 8.0 with GORM
- **Cache**: Redis
- **Authentication**: JWT (golang-jwt/jwt)
- **Configuration**: Viper
- **Logging**: Uber Zap with Lumberjack
- **API Documentation**: Swagger/OpenAPI

## Prerequisites

- Docker & Docker Compose (recommended)
- Or locally: Go 1.26+, MySQL 8.0+, Redis

## Quick Start with Docker Compose

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd bluebell
   ```

2. **Start all services**
   ```bash
   docker-compose up -d
   ```

   This will start:
   - MySQL database (port 3307)
   - Redis cache (port 6380)
   - Bluebell application (port 8081)

3. **Check service health**
   ```bash
   docker-compose ps
   docker-compose logs -f bluebell
   ```

4. **Access the application**
   - API: `http://localhost:8081`
   - Swagger UI: `http://localhost:8081/swagger/index.html`

## Local Development

### Build

```bash
# Build the application
go build -o bluebell ./cmd/bluebell

# Or use the Makefile
make build
```

### Run

```bash
# Start dependencies (MySQL + Redis) using Docker Compose
docker-compose up -d mysql redis

# Run the application
./bluebell -conf config.toml
```

### Environment Configuration

1. **config.toml**: Main configuration file for local development
2. **config.docker.toml**: Docker-optimized configuration

Key configuration sections:
- Database credentials and connection pool settings
- Redis connection parameters
- Server port and logging level
- JWT secret and token expiration

## Docker Deployment

### Build Docker Image

```bash
docker build -t bluebell:latest .
```

### Run with Docker Compose

```bash
docker-compose up -d
```

### View Logs

```bash
docker-compose logs -f bluebell
docker-compose logs -f mysql
docker-compose logs -f redis
```

### Stop Services

```bash
docker-compose down
```

## Project Structure

```
.
├── cmd/
│   └── bluebell/           # Main application entry point
├── internal/               # Internal packages (not exported)
├── pkg/                    # Public packages
├── sql/                    # Database initialization scripts
├── docs/                   # Documentation
├── Dockerfile              # Multi-stage Docker build
├── docker-compose.yml      # Docker Compose configuration
├── config.toml             # Local configuration
├── config.docker.toml      # Docker configuration
├── Makefile                # Build targets
└── go.mod/go.sum          # Go dependencies
```

## Database Setup

Database initialization scripts are located in `./sql/` and are automatically run when MySQL starts via Docker Compose.

To manually initialize:
```bash
mysql -u root -p bluebell < sql/schema.sql
```

## API Documentation

Once running, Swagger documentation is available at:
```
http://localhost:8081/swagger/index.html
```

## Features

- ✅ RESTful API endpoints
- ✅ JWT-based authentication
- ✅ Input validation with detailed error messages
- ✅ Structured logging with Zap
- ✅ Redis caching
- ✅ MySQL database with GORM ORM
- ✅ Rate limiting
- ✅ Docker support with multi-stage builds
- ✅ Post Remark (Comment) functionality
- ✅ Optimized remark listing with GORM preloading
- ✅ Automated database migrations

## Contributing

1. Create a feature branch
2. Make your changes
3. Run tests: `go test ./...`
4. Commit with descriptive messages
5. Push and create a pull request

## Troubleshooting

### MySQL Connection Issues
- Check if MySQL container is running: `docker-compose ps`
- Verify credentials in config file match docker-compose.yml
- Check MySQL logs: `docker-compose logs mysql`

### Redis Connection Issues
- Ensure Redis service is healthy: `docker-compose ps`
- Test connection: `redis-cli -p 6380 ping`

### Application Won't Start
- Review logs: `docker-compose logs bluebell`
- Verify database initialization completed
- Check port 8081 is not already in use

## License

[Specify your license here]

## Contact

[Your contact information]
