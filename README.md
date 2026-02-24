# 🚀 User Authentication Service

A production-ready, high-performance user authentication and management service written in Go, featuring secure login/registration, dual token-based authentication (PASETO/JWT), OAuth integration (Google), Redis caching, session management, and comprehensive user operations. Built with clean hexagonal architecture principles for maintainability, testability, and extensibility.

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go" alt="Go Version"/>
  <img src="https://img.shields.io/badge/PostgreSQL-18+-336791?logo=postgresql" alt="PostgreSQL"/>
  <img src="https://img.shields.io/badge/Redis-7+-DC382D?logo=redis" alt="Redis"/>
  <img src="https://img.shields.io/badge/gRPC-1.78-244C5A?logo=grpc" alt="gRPC"/>
  <img src="https://img.shields.io/badge/License-MIT-green" alt="License"/>
  <img src="https://img.shields.io/badge/Status-Active-brightgreen" alt="Status"/>
  <img src="https://img.shields.io/badge/Architecture-Hexagonal-blue" alt="Architecture"/>
</p>

---

## 📚 Table of Contents
- [Features](#-features)
- [Architecture](#-architecture)
- [API Endpoints](#-api-endpoints)
- [gRPC Services](#-grpc-services)
- [Data Models](#-data-models)
- [Authentication & Tokens](#-authentication--tokens)
- [OAuth Integration](#-oauth-integration)
- [Configuration](#-configuration)
- [Database & Caching](#-database--caching)
- [Setup & Usage](#-setup--usage)
- [API Testing with Bruno](#-api-testing-with-bruno)
- [Extending & Customizing](#-extending--customizing)
- [Project Structure](#-project-structure)
- [License](#-license)
- [Contributing](#-contributing)

---

## ✨ Features

- 🔒 **Secure authentication** with bcrypt password hashing
- 🛡️ **Role-based access control** (USER/ADMIN roles)
- 🪪 **Dual token authentication** (PASETO or JWT — configurable via `TOKEN_TYPE`)
- ♻️ **Access & refresh token** system with automatic renewal and rotation
- 🔐 **OAuth authentication** with Google (via [Goth](https://github.com/markbates/goth) library)
- 🗂️ **Session management** with Redis caching and database persistence
- 🏗️ **Clean hexagonal architecture** with clear separation of concerns
- 🧩 **Dependency injection container** (`internal/app/`) for bootstrapping all services
- ✅ **Comprehensive input validation** with custom validation rules (`hexlower`, `optional_url`, `date`)
- 🚦 **Rate limiting** middleware (40 req/sec) for API protection
- 🐘 **PostgreSQL 18** integration with GORM ORM and optimized connection pooling
- 📡 **gRPC API** for high-performance service-to-service communication (with reflection enabled)
- 🔄 **Redis 7** caching with connection pooling, retries, and TTL-based expiration
- 🐳 **Dockerized** with multi-service setup (PostgreSQL, Redis, RedisInsight)
- ⚙️ **Environment-based configuration** with Viper
- 🔍 **User management** (CRUD operations with self-only access, admin-only create)
- 🛡️ **Security features** (token expiration, session blocking, IP tracking, user agent logging, refresh token reuse detection)
- 🔄 **Graceful shutdown** support for both HTTP and gRPC servers
- 📡 **CORS support** via `gin-contrib/cors`

---

## 🏗️ Architecture

This project follows **Clean Hexagonal Architecture** principles with four distinct layers:

```
┌──────────────────────────────────────────────────────────────────┐
│                        Entry Points                              │
│  main.go | cmd/http/main.go | cmd/grpc/main.go                  │
└──────────────────────┬───────────────────────────────────────────┘
                       │
┌──────────────────────▼───────────────────────────────────────────┐
│                   Application Layer                              │
│  internal/app/                                                   │
│  ├── bootstrap.go    Wires all dependencies (DI container)       │
│  └── container.go    Holds all initialized services/repos        │
└──────────────────────┬───────────────────────────────────────────┘
                       │
┌──────────────────────▼───────────────────────────────────────────┐
│                    Adapter Layer (Driving & Driven)               │
│  internal/adapter/                                               │
│  ├── controller/     HTTP handlers (Gin) — auth, user, oauth     │
│  ├── grpc/           gRPC servers — auth, user, metadata         │
│  ├── middleware/      Auth middleware, rate limiting              │
│  ├── validator/       Custom validation rules & messages         │
│  ├── auth/            Token payload, JWT & PASETO implementations│
│  └── storage/         Database (PostgreSQL) & Redis adapters     │
└──────────────────────┬───────────────────────────────────────────┘
                       │
┌──────────────────────▼───────────────────────────────────────────┐
│                      Core Layer                                  │
│  internal/core/                                                  │
│  ├── domain/         Entity models (User, Session, OauthAccount) │
│  ├── service/        Business logic (auth, user services)        │
│  ├── dto/            Data transfer objects (request/response)    │
│  └── port/           Interface contracts (repositories, services)│
└──────────────────────────────────────────────────────────────────┘
                       │
┌──────────────────────▼───────────────────────────────────────────┐
│                     Package Layer                                │
│  pkg/                                                            │
│  ├── config/         Viper-based configuration management        │
│  ├── oauth/          OAuth provider initialization (Goth)        │
│  ├── util/           Utilities (password, token, cache, random)  │
│  ├── proto/          Protobuf service definitions                │
│  └── pb/             Generated gRPC/protobuf Go code             │
└──────────────────────────────────────────────────────────────────┘
```

### Architecture Layers

| Layer | Directory | Responsibility |
|-------|-----------|----------------|
| **Entry Points** | `main.go`, `cmd/` | Application startup, server initialization |
| **Application** | `internal/app/` | Dependency injection, bootstrapping all components into a `Container` |
| **Adapters** | `internal/adapter/` | External concerns — HTTP controllers, gRPC servers, database repos, Redis cache, middleware, validation, token services |
| **Core** | `internal/core/` | Business logic, domain models, service interfaces (ports), DTOs |
| **Packages** | `pkg/` | Reusable utilities, configuration, protobuf definitions, OAuth setup |

---

## 📡 API Endpoints

### 🔑 Auth
| Method | Endpoint                  | Auth Required | Description |
|--------|---------------------------|:---:|------------------------------------------|
| POST   | `/api/auth/login`         | ❌  | User login (returns access/refresh tokens) |
| POST   | `/api/auth/register`      | ❌  | User registration (auto-assigns `USER` role) |
| POST   | `/api/auth/refresh_token` | ❌  | Renew access & refresh tokens (with rotation) |

### 🔐 OAuth
| Method | Endpoint                        | Auth Required | Description |
|--------|---------------------------------|:---:|-------------------------------------------|
| GET    | `/api/oauth/:provider`          | ❌  | Initiate OAuth flow (redirects to provider) |
| GET    | `/api/oauth/:provider/callback` | ❌  | OAuth callback (returns tokens) |

**Supported Providers:**
- ✅ `google` — Google OAuth 2.0

**OAuth Flow:**
1. User visits `/api/oauth/google` → redirected to Google
2. User authenticates on Google
3. Google redirects back to `/api/oauth/google/callback`
4. Service creates/links user account and returns access/refresh tokens

### 👤 Users
| Method | Endpoint               | Auth Required | Role Required | Description |
|--------|------------------------|:---:|:---:|----------------------------------|
| POST   | `/api/users`           | ✅  | `ADMIN` | Create user (admin only) |
| GET    | `/api/users/:username` | ✅  | Any | Get user details (self only) |
| PUT    | `/api/users/:username` | ✅  | Any | Update user details (self only, role change requires ADMIN) |
| DELETE | `/api/users/:username` | ✅  | Any | Delete user (self only, soft delete) |

### 🚦 Rate Limiting
- **Rate Limit**: 40 requests per second (global, token-bucket algorithm)
- **Middleware**: Applied globally to all endpoints via `golang.org/x/time/rate`
- **Response**: `429 Too Many Requests` with error message

### ✅ Validation Rules
| Rule | Description |
|------|-------------|
| `hexlower` | Validates lowercase hexadecimal strings (`^[a-f0-9]+$`) |
| `optional_url` | Allows empty string or valid URL |
| `date` | Supports multiple date formats: `YYYY-MM-DD`, `DD/MM/YYYY`, `DD-MM-YYYY`, `YYYY/MM/DD` |
| Standard | `email`, `required`, `min`, `max`, `oneof`, etc. (Go Playground Validator) |

---

## 📡 gRPC Services

The service exposes gRPC APIs on port `50051` (configurable via `GRPC_PORT`) with **reflection enabled** for tools like `grpcurl`.

### AuthService
| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `LoginUser` | `LoginUserRequest` (email, password, role) | `LoginUserResponse` (user, tokens, session) | Authenticate user and return tokens |

### UserService
| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `CreateUser` | `CreateUserRequest` (first_name, last_name, email, password) | `CreateUserResponse` (user details) | Create a new user account |

### Protobuf Definitions
Located in `pkg/proto/`:
- `user.proto` — `User` message definition
- `rpc_login_user.proto` — Login request/response messages
- `rpc_create_user.proto` — Create user request/response messages
- `service_auth.proto` — AuthService definition
- `service_user.proto` — UserService definition

Generated Go code is in `pkg/pb/`.

---

## 🗃️ Data Models

### User
| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `Username` | `varchar(60)` | Primary Key | Auto-generated random 8-char string |
| `FirstName` | `varchar(20)` | NOT NULL | User's first name |
| `LastName` | `varchar(20)` | NOT NULL | User's last name |
| `Email` | `varchar(100)` | UNIQUE, NOT NULL | User's email address |
| `HashedPassword` | `varchar(255)` | NOT NULL | bcrypt hashed password |
| `Role` | `varchar(10)` | NOT NULL | `USER` or `ADMIN` |
| `PasswordChangedAt` | `timestamptz` | NOT NULL | Last password change timestamp |
| `CreatedAt` | `timestamptz` | NOT NULL, default `CURRENT_TIMESTAMP` | Account creation timestamp |
| `DeletedAt` | `timestamptz` | Indexed | Soft delete (GORM `gorm.DeletedAt`) |

**Relationships:**
- Has one `Session` (foreign key: `Username`)
- Has many `OauthAccounts` (foreign key: `Username`, cascade update/delete)

**Hooks:**
- `BeforeDelete`: Unscoped deletes all associated `OauthAccount` records

### Session
| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `ID` | `uuid` | Primary Key | Matches the refresh token's payload ID |
| `Username` | `varchar(60)` | Indexed, NOT NULL | Foreign key to User |
| `RefreshToken` | `text` | Indexed, NOT NULL | Current valid refresh token |
| `UserAgent` | `varchar(255)` | NOT NULL | Client's user agent string |
| `ClientIP` | `varchar(60)` | NOT NULL | Client's IP address |
| `IsBlocked` | `boolean` | NOT NULL, default `false` | Whether session is blocked |
| `ExpiresAt` | `timestamptz` | NOT NULL | Session expiration time |
| `CreatedAt` | `timestamptz` | NOT NULL, default `CURRENT_TIMESTAMP` | Session creation time |

### OauthAccount
| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `ID` | `uuid` | Primary Key | Unique identifier |
| `Username` | `varchar(60)` | NOT NULL, Composite unique `(username, provider)` | Foreign key to User |
| `Provider` | `varchar(20)` | NOT NULL, Composite unique `(provider, provider_user_id)` | OAuth provider name |
| `ProviderUserID` | `varchar(255)` | NOT NULL, Composite unique `(provider, provider_user_id)` | User ID from provider |
| `Email` | `varchar(100)` | NOT NULL | Email from OAuth provider |
| `CreatedAt` | `timestamptz` | NOT NULL, default `CURRENT_TIMESTAMP` | Record creation time |
| `DeletedAt` | `timestamptz` | Indexed | Soft delete |

**Unique Indexes:**
- `idx_provider_user` — `(provider, provider_user_id)`: Ensures one account per provider
- `idx_user_provider` — `(username, provider)`: Allows multiple providers per user

---

## 🔐 Authentication & Tokens

### Token Types
| Type | Implementation | Key Requirement | Algorithm |
|------|---------------|-----------------|-----------|
| **PASETO** | `internal/adapter/auth/paseto/` | Symmetric key ≥ 32 chars (ChaCha20-Poly1305) | PASETO V2 |
| **JWT** | `internal/adapter/auth/jwt/` | Secret key ≥ 32 chars | HS256 (HMAC-SHA256) |

### Token Flow
```
Login/Register/OAuth
       │
       ├── Generate Access Token  (short-lived: 1h default)
       ├── Generate Refresh Token (long-lived: 720h default)
       └── Create Session Record  (linked to refresh token ID)
              │
              └── Stored in PostgreSQL + cached in Redis
```

### Token Payload Structure
```json
{
  "id":         "uuid",
  "username":   "string",
  "role":       "USER|ADMIN",
  "issued_at":  "timestamp",
  "expires_at": "timestamp"
}
```

### Refresh Token Rotation
When refreshing tokens (`POST /api/auth/refresh_token`):
1. Verify the old refresh token
2. Validate session (not blocked, not expired, user exists)
3. **Reuse detection**: If the submitted token doesn't match the stored token, **all sessions for that user are blocked**
4. **Role validation**: If user's role changed since token was issued, all sessions are invalidated
5. Generate new access + refresh tokens (refresh token keeps same ID)
6. Update session with new refresh token and expiry

### Security Middleware
- **Authorization**: Bearer token validation via `Authorization: Bearer <token>` header
- **User existence check**: Verifies user still exists in database after token verification
- **Role-based access**: Optional role parameter for endpoint-level restrictions

---

## 🔐 OAuth Integration

### Architecture
OAuth is implemented using the [Goth](https://github.com/markbates/goth) library, initialized in `pkg/oauth/oauth.go`.

### Supported Providers
- ✅ **Google** — Scopes: `email`, `profile`

### How It Works
1. **User initiates OAuth**: Visits `/api/oauth/:provider`
2. **Provider authentication**: Redirected to Google for login
3. **Callback handling**: Provider redirects to `/api/oauth/:provider/callback`
4. **Account linking** (3 scenarios):
   - **Existing OAuth account** → Retrieves linked user, issues tokens
   - **No OAuth account, but email exists** → Links to existing user (restores if soft-deleted), creates OAuth account record, issues tokens
   - **Completely new user** → Creates new user (with random username, `USER` role), creates OAuth account record, issues tokens
5. **Token issuance**: Returns standard access/refresh tokens with session

### OAuth Account Management
- Each OAuth provider account is linked to a user account
- Users can have multiple OAuth providers linked to the same account
- OAuth accounts are stored in the `oauth_accounts` table
- Automatic user creation for first-time OAuth users
- Soft-deleted users are automatically restored on OAuth login

---

## ⚙️ Configuration

Configuration is loaded from `app.env` using [Viper](https://github.com/spf13/viper):

```env
# =========================
# DATABASE
# =========================
DB_CONNECTION=postgresql
DB_HOST=localhost
DB_PORT=5432
DB_USER=root
DB_PASSWORD=secret
DB_NAME=auth_db

# =========================
# HTTP
# =========================
HTTP_PORT=8080

# =========================
# GRPC
# =========================
GRPC_PORT=50051

# =========================
# JWT / PASETO
# =========================
TOKEN_TYPE=paseto                # paseto or jwt
SECRET_KEY=12345678910111213141516171819202  # min 32 chars
TOKEN_DURATION=1h                # Access token lifetime
REFRESH_TOKEN_DURATION=720h      # Refresh token lifetime (30 days)

# =========================
# REDIS
# =========================
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=                  # Optional
REDIS_TTL=30m                    # Cache TTL

# =========================
# OAUTH - GOOGLE
# =========================
OAUTH_GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
OAUTH_GOOGLE_CLIENT_SECRET=your-client-secret
OAUTH_GOOGLE_CALLBACK_URL=http://localhost:8080/api/oauth/google/callback
```

### Configuration Structure (`pkg/config/config.go`)
```go
type Configuration struct {
    DB    *DB     // Database connection settings
    HTTP  *HTTP   // HTTP server port
    Auth  *Auth   // Token type, secret, durations
    Redis *Redis  // Redis address, password, TTL
    OAuth *OAuth  // Google OAuth credentials
    Grpc  *Grpc   // gRPC server port
}
```

### Setting Up Google OAuth
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable Google+ API
4. Create OAuth 2.0 credentials
5. Add authorized redirect URI: `http://localhost:8080/api/oauth/google/callback`
6. Copy Client ID and Client Secret to `app.env`

---

## 🐘 Database & Caching

### PostgreSQL
- **Version**: 18 (Alpine)
- **ORM**: GORM with PostgreSQL driver (`pgx/v5`)
- **Auto-migration**: `User`, `Session`, and `OauthAccount` tables created on startup
- **Connection Pool** (optimized):
  - Max Open Connections: 25
  - Max Idle Connections: 10
  - Connection Max Lifetime: 5 minutes
  - Connection Max Idle Time: 1 minute

### Redis
- **Version**: 7 (Alpine)
- **Client**: `go-redis/v9`
- **Connection Pool** (optimized):
  - Pool Size: 10
  - Min Idle Connections: 5
  - Max Retries: 3
  - Dial Timeout: 5s
  - Read/Write Timeout: 3s
- **Cache Strategy**: Key-value with TTL (default 30m)
- **Key Pattern**: `user:<username>` for individual users, `users:*` prefix for list caches
- **Operations**: Parallel cache set + prefix invalidation using goroutines

### RedisInsight
- **Web UI**: Available at `http://localhost:5540` for Redis monitoring
- **Auto-configured** via Docker Compose

### Docker Compose Services
| Service | Image | Port |
|---------|-------|------|
| `postgres` | `postgres:18-alpine` | `5432` |
| `redis` | `redis:7-alpine` | `6379` |
| `redisinsight` | `redis/redisinsight:latest` | `5540` |

---

## ⚡ Setup & Usage

### Prerequisites
- [Go 1.24+](https://golang.org/dl/)
- [Docker](https://www.docker.com/)
- [protoc](https://grpc.io/docs/protoc-installation/) (only for regenerating protobuf code)

### 🚀 Quick Start

1. **Clone the repository:**
   ```sh
   git clone <repo-url>
   cd user-auth-service
   ```

2. **Configure environment:**
   ```sh
   # Edit app.env with your database, Redis, and OAuth credentials
   nano app.env
   ```

3. **Start all services with Docker:**
   ```sh
   make compose-up
   ```

4. **Create database (if needed):**
   ```sh
   make createdb
   ```

5. **Run the application:**
   ```sh
   # HTTP server only (recommended for REST API development)
   make http

   # gRPC server only (for service-to-service communication)
   make grpc

   # Both HTTP + gRPC in single process (legacy entrypoint)
   make run-app
   ```

6. **APIs available at:**
   - HTTP REST: `http://localhost:8080/api/`
   - gRPC: `localhost:50051` (configurable via `GRPC_PORT`)
   - RedisInsight: `http://localhost:5540`

### 🛠️ Makefile Commands
| Command           | Description                        |
|-------------------|------------------------------------|
| `make compose-up` | Start all Docker services (PostgreSQL, Redis, RedisInsight) |
| `make compose-down`| Stop all Docker services          |
| `make http`       | Run the HTTP server (`cmd/http/main.go`) |
| `make grpc`       | Run the gRPC server (`cmd/grpc/main.go`) |
| `make run-app`    | Run combined HTTP + gRPC server (`main.go`, legacy) |
| `make createdb`   | Create the `auth_db` database     |
| `make dropdb`     | Drop the `auth_db` database       |
| `make proto`      | Regenerate gRPC protobuf Go files  |

### Server Entrypoints

| Entrypoint | File | Servers | Bootstrap |
|-----------|------|---------|-----------|
| HTTP only | `cmd/http/main.go` | HTTP (Gin) | Uses `internal/app/Bootstrap()` container |
| gRPC only | `cmd/grpc/main.go` | gRPC | Uses `internal/app/Bootstrap()` container |
| Combined (legacy) | `main.go` | HTTP + gRPC | Manual wiring (no container) |

### 🧪 Testing the API

**Register:**
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "John",
    "last_name": "Doe",
    "email": "user@example.com",
    "password": "password123"
  }'
```

**Login:**
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "role": "USER"
  }'
```

**Refresh Token:**
```bash
curl -X POST http://localhost:8080/api/auth/refresh_token \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "<your-refresh-token>"
  }'
```

**Get User (authenticated):**
```bash
curl -X GET http://localhost:8080/api/users/<username> \
  -H "Authorization: Bearer <your-access-token>"
```

**OAuth Login:**
1. Open browser: `http://localhost:8080/api/oauth/google`
2. Complete Google authentication
3. Redirected back with tokens in response

---

## 🧪 API Testing with Bruno

This project includes a **[Bruno](https://www.usebruno.com/)** API collection for testing all endpoints.

### Collection Location
```
bruno/user-auth-service/
├── opencollection.yml          # Collection config (base URL: http://localhost:8080)
├── auth/                       # Auth endpoint requests
│   ├── login.yml               # POST /api/auth/login
│   ├── register.yml            # POST /api/auth/register
│   ├── refresh-token.yml       # POST /api/auth/refresh_token
│   └── oauth.yml               # GET /api/oauth/:provider
├── user/                       # User endpoint requests
│   ├── create-user.yml         # POST /api/users
│   ├── get-user.yml            # GET /api/users/:username
│   ├── update-user.yml         # PUT /api/users/:username
│   └── delete-user.yml         # DELETE /api/users/:username
└── grpc/                       # gRPC requests
    ├── CreateUser.yml           # UserService.CreateUser
    └── LoginUser.yml            # AuthService.LoginUser
```

### Usage
1. Install [Bruno](https://www.usebruno.com/)
2. Open the `bruno/user-auth-service/` directory as a collection
3. The base URL is pre-configured to `http://localhost:8080`
4. Bearer token auth is pre-configured at collection level

---

## 🧩 Extending & Customizing

| Task | Where | Details |
|------|-------|---------|
| **New HTTP endpoints** | `internal/adapter/controller/` | Create controller, add routes in `router.go` |
| **New gRPC services** | `pkg/proto/`, `internal/adapter/grpc/` | Define `.proto`, generate code, implement server |
| **Business logic** | `internal/core/service/` | Implement service interface from `port/` |
| **Domain models** | `internal/core/domain/` | Add struct with GORM tags, update `db.Migrate()` |
| **Interface contracts** | `internal/core/port/` | Define repository/service interfaces |
| **Database repositories** | `internal/adapter/storage/database/repository/` | Implement port interfaces |
| **OAuth providers** | `pkg/oauth/oauth.go` | Add new Goth providers (GitHub, Facebook, etc.) |
| **Middleware** | `internal/adapter/middleware/` | Add new Gin middleware handlers |
| **Custom validation** | `internal/adapter/validator/` | Register rules in `register_validation.go`, messages in `message.go` |
| **Caching** | `internal/adapter/storage/redis/` | Extend `CacheRepository` interface in `port/cache.go` |
| **Token types** | `app.env` → `TOKEN_TYPE` | Switch between `paseto` and `jwt` |
| **Configuration** | `pkg/config/config.go` | Add new config sections, update Viper bindings |
| **Dependency wiring** | `internal/app/bootstrap.go` | Add new services/repos to `Container` |

---

## 📁 Project Structure

```
user-auth-service/
├── main.go                                  # Combined HTTP + gRPC entry point (legacy)
├── cmd/                                     # Separate CLI entrypoints
│   ├── http/
│   │   └── main.go                          # HTTP-only server entrypoint
│   └── grpc/
│       └── main.go                          # gRPC-only server entrypoint
├── internal/
│   ├── app/                                 # Application bootstrap & DI
│   │   ├── bootstrap.go                     # Wires all dependencies, returns Container
│   │   └── container.go                     # Container struct holding all services
│   ├── adapter/                             # External adapters
│   │   ├── controller/                      # HTTP controllers (Gin framework)
│   │   │   ├── auth.go                      # Auth endpoints (login, register, refresh)
│   │   │   ├── oauth.go                     # OAuth endpoints (begin, callback)
│   │   │   ├── router.go                    # Route definitions & graceful shutdown
│   │   │   └── user.go                      # User CRUD endpoints
│   │   ├── grpc/                            # gRPC servers
│   │   │   ├── server.go                    # gRPC server setup & graceful shutdown
│   │   │   ├── auth.go                      # gRPC AuthService (LoginUser)
│   │   │   ├── user.go                      # gRPC UserService (CreateUser)
│   │   │   └── metadata.go                  # gRPC metadata extraction (user-agent, IP)
│   │   ├── storage/                         # Data storage layer
│   │   │   ├── database/                    # PostgreSQL with GORM
│   │   │   │   ├── db.go                    # Connection, pooling & auto-migration
│   │   │   │   └── repository/              # Data access layer
│   │   │   │       ├── oauth_account.go     # OauthAccount CRUD
│   │   │   │       ├── session.go           # Session CRUD & blocking
│   │   │   │       └── user.go              # User CRUD & lookup
│   │   │   └── redis/                       # Redis caching layer
│   │   │       └── redis.go                 # Cache operations (Set, Get, Delete, DeleteByPrefix)
│   │   ├── middleware/                      # HTTP middleware
│   │   │   ├── auth.go                      # Bearer token auth & role-based authorization
│   │   │   └── ratelimit.go                 # Token-bucket rate limiting (40 req/sec)
│   │   ├── validator/                       # Input validation
│   │   │   ├── validator.go                 # Validation engine with user-friendly messages
│   │   │   ├── register_validation.go       # Custom rules: hexlower, optional_url, date
│   │   │   └── message.go                   # Validation error message templates
│   │   └── auth/                            # Token services
│   │       ├── payload.go                   # Token payload (implements jwt.Claims)
│   │       ├── util.go                      # Gin context helpers (Set/Get payload)
│   │       ├── jwt/
│   │       │   └── jwt.go                   # JWT implementation (HS256)
│   │       └── paseto/
│   │           └── paseto.go                # PASETO V2 implementation
│   └── core/                                # Business logic
│       ├── domain/                          # Domain models
│       │   ├── user.go                      # User entity with GORM tags & hooks
│       │   ├── session.go                   # Session entity
│       │   └── oauth_account.go             # OauthAccount entity with unique indexes
│       ├── service/                         # Business services
│       │   ├── auth.go                      # Auth service (login, register, OAuth, refresh)
│       │   └── user.go                      # User service (CRUD with caching)
│       ├── dto/                             # Data transfer objects
│       │   ├── common/
│       │   │   └── auth.go                  # Login, Register, RefreshToken DTOs
│       │   ├── request/
│       │   │   ├── user_request.go          # CreateUser, UpdateUser requests
│       │   │   └── session_request.go       # CreateSession request
│       │   └── response/
│       │       └── error.go                 # Standard error response
│       └── port/                            # Interface contracts
│           ├── auth.go                      # AuthenticationService, TokenService interfaces
│           ├── user.go                      # UserRepository, UserService interfaces
│           ├── session.go                   # SessionRepository interface
│           ├── oauth_account.go             # OauthAccountRepository interface
│           └── cache.go                     # CacheRepository interface
├── pkg/                                     # Reusable packages
│   ├── config/
│   │   └── config.go                        # Viper-based configuration struct & loader
│   ├── oauth/
│   │   └── oauth.go                         # Google OAuth provider initialization (Goth)
│   ├── proto/                               # Protobuf service definitions
│   │   ├── user.proto                       # User message
│   │   ├── rpc_login_user.proto             # Login RPC messages
│   │   ├── rpc_create_user.proto            # CreateUser RPC messages
│   │   ├── service_auth.proto               # AuthService definition
│   │   └── service_user.proto               # UserService definition
│   ├── pb/                                  # Generated gRPC/protobuf Go code
│   │   ├── user.pb.go
│   │   ├── rpc_login_user.pb.go
│   │   ├── rpc_create_user.pb.go
│   │   ├── service_auth.pb.go
│   │   ├── service_auth_grpc.pb.go
│   │   ├── service_user.pb.go
│   │   └── service_user_grpc.pb.go
│   └── util/                                # Utility functions
│       ├── cache.go                         # Cache key generation & serialization
│       ├── password.go                      # bcrypt hash & compare
│       ├── random.go                        # Random string/username generation
│       └── token.go                         # Token service factory & session issuer
├── bruno/                                   # Bruno API collection
│   └── user-auth-service/
│       ├── opencollection.yml               # Collection config
│       ├── auth/                            # Auth API requests
│       ├── user/                            # User API requests
│       └── grpc/                            # gRPC requests
├── app.env                                  # Environment configuration
├── docker-compose.yaml                      # Multi-service Docker setup
├── Makefile                                 # Build and run commands
├── go.mod                                   # Go module definition (go 1.24.4)
├── go.sum                                   # Dependency checksums
└── README.md                                # This documentation
```

---

## 🔑 Key Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `gin-gonic/gin` | v1.10.1 | HTTP web framework |
| `gorm.io/gorm` | v1.30.1 | ORM for PostgreSQL |
| `gorm.io/driver/postgres` | v1.6.0 | PostgreSQL driver (pgx) |
| `redis/go-redis/v9` | v9.11.0 | Redis client |
| `golang-jwt/jwt/v5` | v5.2.3 | JWT token implementation |
| `o1egl/paseto` | v1.0.0 | PASETO V2 token implementation |
| `markbates/goth` | v1.82.0 | OAuth multi-provider library |
| `spf13/viper` | v1.20.1 | Configuration management |
| `go-playground/validator/v10` | v10.26.0 | Struct validation |
| `google.golang.org/grpc` | v1.78.0 | gRPC framework |
| `google.golang.org/protobuf` | v1.36.10 | Protocol Buffers |
| `gin-contrib/cors` | v1.7.6 | CORS middleware |
| `golang.org/x/crypto` | v0.44.0 | bcrypt password hashing |
| `golang.org/x/time` | v0.12.0 | Rate limiting |
| `jackc/pgx/v5` | v5.7.5 | PostgreSQL driver |
| `google/uuid` | v1.6.0 | UUID generation |

---

## 📝 License

[MIT](LICENSE)

---

## 🤝 Contributing

Contributions are welcome! Please open issues or pull requests for improvements or bug fixes.
