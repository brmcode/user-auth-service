# 🚀 User Authentication Service

A production-ready, high-performance user authentication and management service written in Go, featuring secure login/registration, dual token-based authentication (PASETO/JWT), OAuth integration (Google), Redis caching, session management, and comprehensive user operations. Built with clean hexagonal architecture principles for maintainability, testability, and extensibility.

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
- [OAuth Integration](#-oauth-integration)
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
- 🔐 **OAuth authentication** with Google
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
│   │   ├── auth.go      # Authentication endpoints
│   │   ├── oauth.go     # OAuth authentication handlers
│   │   ├── router.go    # Route definitions
│   │   └── user.go      # User management endpoints
│   ├── storage/          # Data storage implementations
│   │   ├── database/     # PostgreSQL with GORM
│   │   │   └── repository/ # Data access layer
│   │   │       ├── oauth_account.go
│   │   │       ├── session.go
│   │   │       └── user.go
│   │   └── redis/        # Redis caching layer
│   ├── middleware/       # HTTP middleware (auth, rate limiting)
│   ├── validator/        # Input validation with custom rules
│   └── auth/             # Token service implementations (JWT/PASETO)
└── core/                 # Business logic and domain
    ├── domain/           # Core domain models (User, Session, OauthAccount)
    ├── service/          # Business logic services
    ├── dto/              # Data transfer objects
    │   ├── common/       # Shared DTOs
    │   ├── request/      # Request DTOs
    │   └── response/     # Response DTOs
    └── port/             # Interface definitions (contracts)
pkg/                      # Reusable packages
├── config/               # Configuration management (Viper)
│   ├── config.go         # Main configuration
│   └── oauth.go          # OAuth provider initialization
└── util/                 # Utility functions (password, random, cache, token)
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

### 🔐 OAuth
| Method | Endpoint                        | Description                                    |
|--------|---------------------------------|------------------------------------------------|
| GET    | `/api/oauth/:provider`     | Initiate OAuth flow (redirects to provider)   |
| GET    | `/api/oauth/:provider/callback` | OAuth callback (returns tokens)              |

**Supported Providers:**
- `google` - Google OAuth 2.0

**OAuth Flow:**
1. User visits `/api/oauth/google` to initiate authentication
2. User is redirected to Google for authentication
3. Google redirects back to `/api/oauth/google/callback`
4. Service creates/links user account and returns access/refresh tokens

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
- `HashedPassword` (varchar(255), bcrypt hashed, nullable for OAuth users)
- `Role` (varchar(10), USER/ADMIN)
- `PasswordChangedAt`, `CreatedAt` (timestamptz)
- `DeletedAt` (soft delete with GORM)
- `OauthAccounts` (relationship to OAuth accounts)

### Session
- `ID` (UUID, primary key)
- `Username` (varchar(60), foreign key to User)
- `RefreshToken` (text, indexed)
- `UserAgent` (varchar(255), not null)
- `ClientIp` (varchar(60), not null)
- `IsBlocked` (boolean, default false)
- `ExpiresAt`, `CreatedAt` (timestamptz)

### OauthAccount
- `ID` (UUID, primary key)
- `Username` (varchar(60), foreign key to User, unique index with provider)
- `Provider` (varchar(20), not null, unique index with provider_user_id)
- `ProviderUserID` (varchar(255), not null, unique index with provider)
- `Email` (varchar(100), not null)
- `CreatedAt` (timestamptz, not null)

**Indexes:**
- Unique index on `(provider, provider_user_id)` - ensures one account per provider
- Unique index on `(username, provider)` - allows multiple providers per user

---

## 🔐 Authentication & Tokens
- **Token Types:** Supports PASETO and JWT (configurable via `TOKEN_TYPE`)
- **Access Token:** Short-lived (15m default), used for API authentication
- **Refresh Token:** Long-lived (168h default), used to obtain new access tokens
- **Token Payload:** Includes UUID, username, role, issued/expiry times
- **Session Management:** Each login creates a session record with Redis caching
- **Security Features:** Session blocking, IP tracking, user agent logging, token reuse detection
- **Cache Strategy:** Redis caching for session data with TTL

---

## 🔐 OAuth Integration

The service supports OAuth 2.0 authentication using the [Goth](https://github.com/markbates/goth) library.

### Supported Providers
- ✅ **Google** - Fully implemented and tested

### How It Works
1. **User initiates OAuth:** Visits `/api/oauth/:provider`
2. **Provider authentication:** Redirected to provider (Google) for login
3. **Callback handling:** Provider redirects to `/api/oauth/:provider/callback`
4. **Account linking:**
   - If OAuth account exists → Links to existing user and issues tokens
   - If new OAuth account → Creates new user account and OAuthAccount record
5. **Token issuance:** Returns standard access/refresh tokens

### OAuth Account Management
- Each OAuth provider account is linked to a user account
- Users can have multiple OAuth providers linked to the same account
- OAuth accounts are stored in the `oauth_accounts` table
- Automatic user creation for first-time OAuth users

---

## ⚙️ Configuration

Configuration is loaded from `app.env` using Viper:

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
# JWT / AUTH
# =========================
TOKEN_TYPE=jwt                   # jwt or paseto
SECRET_KEY=12345678910111213141516171819202
TOKEN_DURATION=15m               # Access token lifetime
REFRESH_TOKEN_DURATION=168h      # Refresh token lifetime

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

### Setting Up OAuth Providers

**Google OAuth Setup:**
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing one
3. Enable Google+ API
4. Create OAuth 2.0 credentials
5. Add authorized redirect URI: `http://localhost:8080/api/oauth/google/callback`
6. Copy Client ID and Client Secret to `app.env`

---

## 🐘 Database & Caching
- **PostgreSQL**: Primary database with GORM ORM (see `docker-compose.yaml`)
- **Redis**: Caching layer for session data and performance optimization
- **Auto-migration**: `User`, `Session`, and `OauthAccount` tables created on startup
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

2. **Configure environment:**
   ```sh
   cp app.env app.env.local  # Optional: create local config
   # Edit app.env with your database, Redis, and OAuth credentials
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
   make run-app
   ```

6. **API available at:** `http://localhost:8080/api/`

### 🛠️ Makefile Commands
| Command           | Description                        |
|-------------------|------------------------------------|
| `make compose-up` | Start all services (PostgreSQL, Redis, RedisInsight) |
| `make compose-down`| Stop all services                 |
| `make run-app`    | Run the Go application             |
| `make createdb`   | Create the database                |
| `make dropdb`     | Drop the database                  |

### 🧪 Testing the API

**Traditional Login:**
```bash
# Register
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123","first_name":"John","last_name":"Doe"}'

# Login
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123","role":"USER"}'
```

**OAuth Login:**
1. Open browser: `http://localhost:8080/api/oauth/google`
2. Complete Google authentication
3. Redirected back with tokens in response

---

## 🧩 Extending & Customizing

- **New Endpoints**: Add in `internal/adapter/controller/` and wire in `router.go`
- **Business Logic**: Add in `internal/core/service/` following hexagonal patterns
- **Domain Models**: Add in `internal/core/domain/` with GORM tags
- **Database**: Update migrations in `internal/adapter/storage/database/db.go`
- **OAuth Providers**: Add new providers in `pkg/config/oauth.go` using Goth library
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
│   │   │   ├── auth.go              # Authentication endpoints
│   │   │   ├── oauth.go             # OAuth authentication handlers
│   │   │   ├── router.go            # Route definitions
│   │   │   └── user.go              # User management endpoints
│   │   ├── storage/                 # Data storage layer
│   │   │   ├── database/            # PostgreSQL with GORM
│   │   │   │   ├── db.go            # Database connection & migration
│   │   │   │   └── repository/      # Data access layer
│   │   │   │       ├── oauth_account.go  # OAuth account repository
│   │   │   │       ├── session.go   # Session repository
│   │   │   │       └── user.go      # User repository
│   │   │   └── redis/               # Redis caching layer
│   │   ├── middleware/              # HTTP middleware
│   │   │   ├── auth.go              # Authentication middleware
│   │   │   └── ratelimit.go         # Rate limiting middleware
│   │   ├── validator/               # Input validation
│   │   │   ├── validator.go         # Custom validation rules
│   │   │   ├── register_validation.go
│   │   │   └── message.go           # Validation messages
│   │   └── auth/                    # Token services
│   │       ├── payload.go           # Token payload structure
│   │       ├── jwt/                 # JWT implementation
│   │       │   └── jwt.go
│   │       └── paseto/              # PASETO implementation
│   │           └── paseto.go
│   └── core/                        # Business logic
│       ├── domain/                  # Domain models
│       │   ├── oauth_account.go     # OAuth account entity
│       │   ├── session.go           # Session entity
│       │   └── user.go              # User entity
│       ├── service/                 # Business services
│       │   ├── auth.go              # Authentication service
│       │   └── user.go              # User management service
│       ├── dto/                     # Data transfer objects
│       │   ├── common/              # Shared DTOs
│       │   │   └── auth.go
│       │   ├── request/             # Request DTOs
│       │   │   ├── session_request.go
│       │   │   └── user_request.go
│       │   └── response/            # Response DTOs
│       │       └── error.go
│       └── port/                    # Interface contracts
│           ├── auth.go              # Authentication interfaces
│           ├── cache.go             # Cache repository interface
│           ├── oauth_account.go     # OAuth account repository interface
│           ├── session.go           # Session repository interface
│           └── user.go              # User repository interface
├── pkg/                             # Reusable packages
│   ├── config/                      # Configuration management
│   │   ├── config.go                # Viper-based config
│   │   └── oauth.go                 # OAuth provider initialization
│   └── util/                        # Utility functions
│       ├── cache.go                 # Cache utilities
│       ├── password.go              # Password hashing utilities
│       ├── random.go                # Random string generation
│       └── token.go                 # Token utilities
├── app.env                          # Environment configuration
├── docker-compose.yaml              # Multi-service Docker setup
├── Makefile                         # Build and run commands
├── go.mod                           # Go module definition
├── go.sum                           # Dependency checksums
├── user-auth-service.postman_collection.json  # Postman API collection
└── README.md                        # Project documentation
```

---

## 📝 License

[MIT](LICENSE)

---

## 🤝 Contributing

Contributions are welcome! Please open issues or pull requests for improvements or bug fixes.
