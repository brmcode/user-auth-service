# 🚀 User Authentication Service

A robust, modular user authentication and management service written in Go, supporting secure login, registration, token-based authentication (PASETO/JWT), and user/session management. Built with clean architecture principles for maintainability and extensibility.

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.20+-00ADD8?logo=go" alt="Go Version"/>
  <img src="https://img.shields.io/badge/PostgreSQL-16+-336791?logo=postgresql" alt="PostgreSQL"/>
  <img src="https://img.shields.io/badge/License-MIT-green" alt="License"/>
  <img src="https://img.shields.io/badge/Status-Active-brightgreen" alt="Status"/>
  <img src="https://img.shields.io/badge/Architecture-Clean-blue" alt="Architecture"/>
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

- 🔒 **Secure authentication** with hashed passwords
- 🛡️ **Role-based access control** (admin/user)
- 🪪 **Token-based authentication** (PASETO or JWT)
- ♻️ **Access & refresh token** issuance and renewal
- 🗂️ **Session management** with database persistence
- 🏗️ **Clean architecture** with clear separation of concerns
- 🐘 **PostgreSQL** integration
- 🐳 **Dockerized** for easy deployment
- ⚙️ **Environment-based configuration**

---

## 🏗️ Architecture

This project follows **Clean Architecture** principles with the following structure:

```
main.go                    # Application entry point
internal/                  # Application-specific code
├── adapter/              # External interfaces (HTTP, DB, Auth)
│   ├── controller/       # HTTP handlers and routing
│   ├── database/         # Database connection and migration
│   └── auth/             # Token service implementations
└── core/                 # Business logic and domain
    ├── domain/           # Core domain models
    ├── service/          # Business logic services
    ├── dto/              # Data transfer objects
    └── port/             # Interface definitions
pkg/                      # Reusable packages
├── config/               # Configuration management
└── util/                 # Utility functions
repository/               # Data access layer
app.env                   # Environment configuration
Makefile                  # Build and run commands
docker-compose.yaml       # Docker services
go.mod, go.sum           # Go modules
```

### Architecture Layers:
- **Adapters**: Handle external concerns (HTTP, database, authentication)
- **Core**: Contains business logic, domain models, and interfaces
- **Repository**: Data access and persistence logic
- **Pkg**: Reusable utilities and configuration

---

## 📡 API Endpoints

### 🔑 Auth
| Method | Endpoint                  | Description                                 |
|--------|---------------------------|---------------------------------------------|
| POST   | `/api/auth/login`         | User login (returns tokens, session info)   |
| POST   | `/api/auth/register`      | User registration                           |
| POST   | `/api/auth/refresh_token` | Renew access token using refresh token      |

### 👤 Users
| Method | Endpoint                | Description                        |
|--------|-------------------------|------------------------------------|
| POST   | `/api/users`            | Create user (admin only)           |
| GET    | `/api/users/:username`  | Get user details (self only)       |
| PUT    | `/api/users/:username`  | Update user details (self only)    |
| DELETE | `/api/users/:username`  | Delete user (self only)            |

---

## 🗃️ Data Models

### User
- `Username` (primary key)
- `FirstName`, `LastName`, `Email` (unique)
- `HashedPassword`
- `Role` (admin/user)
- `CreatedAt`, `PasswordChangedAt`, `DeletedAt`

### Session
- `ID` (UUID, primary key)
- `Username` (foreign key)
- `RefreshToken`
- `UserAgent`, `ClientIp`
- `IsBlocked`
- `ExpiresAt`, `CreatedAt`

---

## 🔐 Authentication & Tokens
- **Token Types:** Supports PASETO and JWT (configurable via `TOKEN_TYPE`)
- **Access Token:** Short-lived, used for API authentication
- **Refresh Token:** Long-lived, used to obtain new access tokens
- **Token Payload:** Includes user ID, role, issued/expiry times
- **Session:** Each login creates a session record with refresh token

---

## ⚙️ Configuration

Configuration is loaded from `app.env`:

```env
DB_CONNECTION=postgresql
DB_HOST=localhost
DB_PORT=5432
DB_USER=root
DB_PASSWORD=secret
DB_NAME=auth_db

HTTP_PORT=8080
TOKEN_TYPE=JWT
SECRET_KEY=12345678910111213141516171819202
TOKEN_DURATION=15m
REFRESH_TOKEN_DURATION=168h
```

---

## 🐘 Database
- Uses PostgreSQL (see `docker-compose.yaml` for service definition)
- Auto-migrates `User` and `Session` tables on startup

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
2. **Start PostgreSQL with Docker:**
   ```sh
   make docker-up
   ```
3. **Run the application:**
   ```sh
   make run-app
   ```
4. **API available at:** `http://localhost:8080/api/`

### 🛠️ Makefile Commands
| Command           | Description                        |
|-------------------|------------------------------------|
| `make docker-up`  | Start PostgreSQL via Docker Compose |
| `make docker-down`| Stop PostgreSQL                    |
| `make run-app`    | Run the Go application             |
| `make createdb`   | Create the database                |
| `make dropdb`     | Drop the database                  |

---

## 🧩 Extending & Customizing
- Add new endpoints in `internal/adapter/controller/` and wire them in `router.go`
- Add business logic in `internal/core/service/`
- Add new models in `internal/core/domain/` and update migrations in `internal/adapter/database/db.go`
- Switch token type (PASETO/JWT) via `app.env` `TOKEN_TYPE` setting

---

## 📁 Project Structure

```
user-auth-service/
├── main.go
├── internal/
│   ├── adapter/
│   │   ├── controller/
│   │   │   ├── auth.go
│   │   │   ├── middleware.go
│   │   │   ├── router.go
│   │   │   └── user.go
│   │   ├── database/
│   │   │   └── db.go
│   │   └── auth/
│   │       ├── payload.go
│   │       ├── service.go
│   │       ├── jwt/
│   │       │   └── jwt.go
│   │       └── paseto/
│   │           └── paseto.go
│   └── core/
│       ├── domain/
│       │   ├── session.go
│       │   └── user.go
│       ├── service/
│       │   ├── auth.go
│       │   └── user.go
│       ├── dto/
│       │   ├── auth.go
│       │   ├── session_request.go
│       │   ├── user_request.go
│       │   └── response/
│       │       └── error.go
│       └── port/
├── pkg/
│   ├── config/
│   │   └── config.go
│   └── util/
│       ├── password.go
│       └── random.go
├── repository/
│   ├── session.go
│   └── user.go
├── app.env
├── docker-compose.yaml
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

---

## 📝 License

[MIT](LICENSE)

---

## 🤝 Contributing

Contributions are welcome! Please open issues or pull requests for improvements or bug fixes.
