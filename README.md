# 🚀 User Authentication Service

A production-ready, high-performance user authentication and management service written in Go, featuring secure login/registration, dual token-based authentication (PASETO/JWT), Redis caching, session management, and comprehensive user operations. Built with clean hexagonal architecture principles for maintainability, testability, and extensibility.

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.20+-00ADD8?logo=go" alt="Go Version"/>
  <img src="https://img.shields.io/badge/PostgreSQL-16+-336791?logo=postgresql" alt="PostgreSQL"/>
  <img src="https://img.shields.io/badge/License-MIT-green" alt="License"/>
  <img src="https://img.shields.io/badge/Status-Active-brightgreen" alt="Status"/>
  <img src="https://img.shields.io/badge/Architecture-Hexagonal-blue" alt="Architecture"/>
</p>

---

## 📚 Table of Contents
- [Features](#-features)
- [Architecture](#-architecture)
- [API Endpoints](#-api-endpoints)
- [Data Models](#-data-models)
- [Authentication & Tokens](#-authentication--tokens)
- [Configuration](#-configuration)
- [Database](#-database)
- [Setup & Usage](#-setup--usage)
- [Extending & Customizing](#-extending--customizing)
- [Project Structure](#-project-structure)
- [License](#-license)
- [Contributing](#-contributing)

---

## ✨ Features

- 🔒 **Secure authentication** with bcrypt password hashing
- 🛡️ **Role-based access control** (USER/ADMIN roles)
- 🪪 **Dual token authentication** (PASETO or JWT - configurable)
- ♻️ **Access & refresh token** system with automatic renewal
- 🗂️ **Session management** with Redis caching and database persistence
- 🏗️ **Clean hexagonal architecture** with clear separation of concerns
- ✅ **Comprehensive input validation** with custom validation rules
- 🚦 **Rate limiting** middleware (40 req/sec) for API protection
- 🐘 **PostgreSQL** integration with GORM ORM
- 🔄 **Redis caching** for session and performance optimization
- 🐳 **Dockerized** with multi-service setup (PostgreSQL, Redis, RedisInsight)
- ⚙️ **Environment-based configuration** with Viper
- 🔍 **User management** (CRUD operations with self-only access)
- 🛡️ **Security features** (token expiration, session blocking, IP tracking)

---

## 🏗️ Architecture

This project follows **Clean Hexagonal Architecture** principles with the following structure:

```
main.go                    # Application entry point and dependency injection
internal/                  # Application-specific code
├── adapter/              # External interfaces and adapters
│   ├── controller/       # HTTP handlers and routing (Gin framework)
│   ├── storage/          # Data storage implementations
│   │   ├── database/     # PostgreSQL with GORM
│   │   │   └── repository/ # Data access layer
│   │   └── redis/        # Redis caching layer
│   ├── middleware/       # HTTP middleware (auth, rate limiting)
│   ├── validator/        # Input validation with custom rules
│   └── auth/             # Token service implementations (JWT/PASETO)
└── core/                 # Business logic and domain
    ├── domain/           # Core domain models (User, Session)
    ├── service/          # Business logic services
    ├── dto/              # Data transfer objects
    │   ├── common/       # Shared DTOs
    │   ├── request/      # Request DTOs
    │   └── response/     # Response DTOs
    └── port/             # Interface definitions (contracts)
pkg/                      # Reusable packages
├── config/               # Configuration management (Viper)
└── util/                 # Utility functions (password, random)
app.env                   # Environment configuration
Makefile                  # Build and run commands
docker-compose.yaml       # Multi-service Docker setup
go.mod, go.sum           # Go modules and dependencies
```

### Architecture Layers:
- **Adapters**: Handle external concerns (HTTP controllers, database repositories, Redis cache, authentication services, middleware, validation)
- **Core**: Contains business logic, domain models, services, and interface contracts
- **Ports**: Define contracts between layers (repository interfaces, service interfaces)
- **Pkg**: Reusable utilities and configuration management

---

## 📡 API Endpoints

> 📋 **Testing**: Download the Postman collection for ready-to-use API requests with examples and authentication tokens.
>
> [📥 Download Postman Collection](user-auth-service.postman_collection.json)

### 🔑 Auth
| Method | Endpoint                  | Description                                 |
|--------|---------------------------|---------------------------------------------|
| POST   | `/api/auth/login`         | User login (returns access/refresh tokens) |
| POST   | `/api/auth/register`      | User registration                           |
| POST   | `/api/auth/refresh_token` | Renew access token using refresh token      |

### 👤 Users
| Method | Endpoint                | Description                        |
|--------|-------------------------|------------------------------------|
| POST   | `/api/users`            | Create user (admin only)           |
| GET    | `/api/users/:username`  | Get user details (self only)       |
| PUT    | `/api/users/:username`  | Update user details (self only)    |
| DELETE | `/api/users/:username`  | Delete user (self only)            |

### 🚦 Rate Limiting
- **Rate Limit**: 40 requests per second (configurable)
- **Middleware**: Applied globally to all endpoints
- **Response**: 429 Too Many Requests with error message

### ✅ Validation
- **Custom Rules**: hexlower, optional_url, date formats
- **Standard Rules**: email, required, min, max, etc.
- **Error Messages**: User-friendly validation messages
- **Framework**: Go Playground Validator with custom tags

---

## 🗃️ Data Models

### User
- `Username` (primary key, varchar(60))
- `FirstName`, `LastName` (varchar(20), not null)
- `Email` (varchar(100), unique, not null)
- `HashedPassword` (varchar(255), bcrypt hashed)
- `Role` (varchar(10), USER/ADMIN)
- `PasswordChangedAt`, `CreatedAt` (timestamptz)
- `DeletedAt` (soft delete with GORM)

### Session
- `ID` (UUID, primary key)
- `Username` (varchar(60), foreign key to User)
- `RefreshToken` (text, indexed)
- `UserAgent` (varchar(255), not null)
- `ClientIp` (varchar(60), not null)
- `IsBlocked` (boolean, default false)
- `ExpiresAt`, `CreatedAt` (timestamptz)

---

## 🔐 Authentication & Tokens
- **Token Types:** Supports PASETO and JWT (configurable via `TOKEN_TYPE`)
- **Access Token:** Short-lived (15m default), used for API authentication
- **Refresh Token:** Long-lived (720h default), used to obtain new access tokens
- **Token Payload:** Includes UUID, username, role, issued/expiry times
- **Session Management:** Each login creates a session record with Redis caching
- **Security Features:** Session blocking, IP tracking, user agent logging
- **Cache Strategy:** Redis caching for session data with TTL

---

## ⚙️ Configuration

Configuration is loaded from `app.env` using Viper:

```env
# Database Configuration
DB_CONNECTION=postgresql
DB_HOST=localhost
DB_PORT=5432
DB_USER=root
DB_PASSWORD=secret
DB_NAME=auth_db

# HTTP Server
HTTP_PORT=8080

# Authentication
TOKEN_TYPE=jwt                    # jwt or paseto
SECRET_KEY=12345678910111213141516171819202
TOKEN_DURATION=15m               # Access token lifetime
REFRESH_TOKEN_DURATION=720h      # Refresh token lifetime

# Redis Configuration
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=                  # Optional
REDIS_TTL=15m                   # Cache TTL
```

---

## 🐘 Database & Caching
- **PostgreSQL**: Primary database with GORM ORM (see `docker-compose.yaml`)
- **Redis**: Caching layer for session data and performance optimization
- **Auto-migration**: `User` and `Session` tables created on startup
- **Connection Pool**: Optimized database connections with pgx driver
- **RedisInsight**: Web UI for Redis monitoring (port 5540)

---

## ⚡ Setup & Usage

### Prerequisites
- [Go 1.20+](https://golang.org/dl/)
- [Docker](https://www.docker.com/)

### 🚀 Quick Start

1. **Clone the repository:**
   ```sh
   git clone <repo-url>
   cd user-auth-service
   ```
2. **Start all services with Docker:**
   ```sh
   make compose-up
   ```
3. **Run the application:**
   ```sh
   make run-app
   ```
4. **API available at:** `http://localhost:8080/api/`

### 🛠️ Makefile Commands
| Command           | Description                        |
|-------------------|------------------------------------|
| `make compose-up` | Start all services (PostgreSQL, Redis, RedisInsight) |
| `make compose-down`| Stop all services                 |
| `make run-app`    | Run the Go application             |
| `make createdb`   | Create the database                |
| `make dropdb`     | Drop the database                  |

---

## 🧩 Extending & Customizing
- **New Endpoints**: Add in `internal/adapter/controller/` and wire in `router.go`
- **Business Logic**: Add in `internal/core/service/` following hexagonal patterns
- **Domain Models**: Add in `internal/core/domain/` with GORM tags
- **Database**: Update migrations in `internal/adapter/storage/database/db.go`
- **Middleware**: Add in `internal/adapter/middleware/` (rate limiting, auth, etc.)
- **Validation**: Add custom rules in `internal/adapter/validator/`
- **Caching**: Extend Redis caching in `internal/adapter/storage/redis/`
- **Token Types**: Switch between PASETO/JWT via `app.env` `TOKEN_TYPE` setting
- **Configuration**: Add new config sections in `pkg/config/config.go`

---

## 📁 Project Structure

```
user-auth-service/
├── main.go                          # Application entry point
├── internal/
│   ├── adapter/                     # External adapters
│   │   ├── controller/              # HTTP controllers (Gin)
│   │   │   ├── auth.go             # Authentication endpoints
│   │   │   ├── router.go           # Route definitions
│   │   │   └── user.go             # User management endpoints
│   │   ├── storage/                 # Data storage layer
│   │   │   ├── database/           # PostgreSQL with GORM
│   │   │   │   ├── db.go           # Database connection & migration
│   │   │   │   └── repository/     # Data access layer
│   │   │   │       ├── session.go  # Session repository
│   │   │   │       └── user.go     # User repository
│   │   │   └── redis/              # Redis caching layer
│   │   ├── middleware/              # HTTP middleware
│   │   │   ├── auth.go             # Authentication middleware
│   │   │   └── ratelimit.go        # Rate limiting middleware
│   │   ├── validator/               # Input validation
│   │   │   ├── validator.go        # Custom validation rules
│   │   │   ├── register_validation.go
│   │   │   └── message.go          # Validation messages
│   │   └── auth/                   # Token services
│   │       ├── payload.go          # Token payload structure
│   │       ├── jwt/                # JWT implementation
│   │       │   └── jwt.go
│   │       └── paseto/             # PASETO implementation
│   │           └── paseto.go
│   └── core/                       # Business logic
│       ├── domain/                 # Domain models
│       │   ├── session.go          # Session entity
│       │   └── user.go             # User entity
│       ├── service/                # Business services
│       │   ├── auth.go             # Authentication service
│       │   └── user.go             # User management service
│       ├── dto/                    # Data transfer objects
│       │   ├── common/             # Shared DTOs
│       │   │   └── auth.go
│       │   ├── request/            # Request DTOs
│       │   │   ├── session_request.go
│       │   │   └── user_request.go
│       │   └── response/           # Response DTOs
│       │       └── error.go
│       └── port/                   # Interface contracts
│           ├── auth.go             # Authentication interfaces
│           ├── session.go          # Session repository interface
│           └── user.go             # User repository interface
├── pkg/                            # Reusable packages
│   ├── config/                     # Configuration management
│   │   └── config.go              # Viper-based config
│   └── util/                       # Utility functions
│       ├── password.go            # Password hashing utilities
│       └── random.go              # Random string generation
├── app.env                         # Environment configuration
├── docker-compose.yaml             # Multi-service Docker setup
├── Makefile                        # Build and run commands
├── go.mod                          # Go module definition
├── go.sum                          # Dependency checksums
└── README.md                       # Project documentation
```

---

## 📝 License

[MIT](LICENSE)

---

## 🤝 Contributing

Contributions are welcome! Please open issues or pull requests for improvements or bug fixes.
