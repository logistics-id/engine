# REST Transport Library

A feature-rich HTTP/REST server wrapper built on top of [gorilla/mux](https://github.com/gorilla/mux). It provides a standardized way to build RESTful APIs with built-in middleware for logging, recovery, CORS, and authentication.

## Features

- **Gorilla Mux Integration**: Full access to `mux` router capabilities.
- **Built-in Middleware**:
    - **Request ID**: Automatically generates and propagates `X-Request-ID`.
    - **Logging**: Structured request/response logging via `zap`.
    - **Recovery**: Gracefully recovers from panics.
    - **CORS**: Pre-configured CORS support.
    - **JWT Auth**: Built-in JWT token validation.
- **Context Management**: Custom `Context` struct for simplified request/response handling.
- **Standardized Errors**: Helper functions for consistent error responses.

## Installation

```bash
go get github.com/logistics-id/engine/transport/rest
```

## Quick Start

### 1. Configuration & Initialization

```go
package main

import (
    "context"
    "os"
    "github.com/logistics-id/engine/transport/rest"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewProduction()

    cfg := &rest.Config{
        Server: ":8080",
        IsDev:  true,
    }

    // Initialize Server
    server := rest.NewServer(cfg, logger, func(srv *rest.RestServer) {
        // Register routes here
        srv.GET("/hello", HelloHandler, nil)
    })

    // Start Server
    ctx := context.Background()
    go server.Start(ctx)

    // Wait for shutdown signal...
    <-ctx.Done()
    server.Shutdown(ctx)
}

func HelloHandler(ctx *rest.Context) error {
    return ctx.JSON(200, map[string]string{"message": "Hello World"})
}
```

## API Reference

### Route Registration

The `RestServer` provides shorthand methods for registering routes:

```go
// GET request
server.GET("/users/{id}", GetUserHandler, middlewareList)

// POST request
server.POST("/users", CreateUserHandler, nil)

// Middleware handling
server.PUT("/users/{id}", UpdateUserHandler, server.WithAuth(true))
```

### Handler Function

### Handler Function

Handlers use the signature `func(*rest.Context) error`.

```go
func MyHandler(ctx *rest.Context) error {
    // ...
}
```

### Request Binding & Validation

The `Bind` method is a powerful helper that handles:
1.  **JSON Decoding**: Decodes the request body into the struct.
2.  **Path Parameters**: Binds URL path variables (from mux) to struct fields with `param:"id"` tag.
3.  **Query Parameters**: Binds URL path variables (from query params) to struct fields; default matches field name (ex `q` matches `q` query param) or use `query:"limit"` tag.
4.  **Validation**: Automatically validates the struct using the `validate` package tags.

```go
import "github.com/logistics-id/engine/validate"

type CreateUserRequest struct {
    Name  string `json:"name" valid:"required|alpha_space"`
    Email string `json:"email" valid:"required|email"`
    Role  string `json:"role" valid:"required|in:admin,user"`
}

// Optional: Implement validate.Request interface for custom messages
func (r *CreateUserRequest) Messages() map[string]string {
    return map[string]string{
        "required": "The %s field is mandatory",
        "email":    "Invalid email format",
    }
}

func (r *CreateUserRequest) Validate() *validate.Response {
    // Optional: Add custom validation logic here
    return nil
}

func CreateUserHandler(ctx *rest.Context) error {
    var req CreateUserRequest

    // Bind decodes JSON, binds params, and validates struct.
    // validation errors return *validate.Response, which ctx.Respond handles automatically (422)
    if err := ctx.Bind(&req); err != nil {
        return ctx.Respond(nil, err)
    }

    return ctx.JSON(201, req)
}
```

### Binding Query Options (GET)

The `Bind` method supports `common.QueryOption` directly for list endpoints, automatically mapping query parameters like `?page=1&limit=10&search=foo&order_by=-created_at`.


The `Bind` method supports `common.QueryOption` directly. It is recommended to embed it in your request struct.

```go
import "github.com/logistics-id/engine/common"

type ListUserRequest struct {
    common.QueryOption
}

func ListUsersHandler(ctx *rest.Context) error {
    var req ListUserRequest

    // Auto-binds embedded QueryOption fields (limit, page, search, order_by)
    if err := ctx.Bind(&req); err != nil {
        return ctx.Respond(nil, err)
    }

    // Pass embedded options to repository
    users, total, err := userRepo.FindAll(&req.QueryOption, nil)
    if err != nil {
        return ctx.Respond(nil, err)
    }

    return ctx.Respond(map[string]any{
        "items": users,
        "total": total,
    }, nil)
}
```

### Middleware

#### Authentication (`WithAuth`)

Validates `Authorization: Bearer <token>`.

```go
// Require authentication
server.GET("/profile", ProfileHandler, server.WithAuth(true))

// Require specific role
server.GET("/admin", AdminHandler, server.WithAuth(true, "admin"))
```

#### Authorization (`Restricted`)

Checks for specific permissions in the JWT claims.

```go
server.POST("/documents", CreateDocHandler, server.Restricted("document:create"))
```
