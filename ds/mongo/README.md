# MongoDB Data Source Library

A robust MongoDB client library wrapper based on the official [mongo-driver](https://github.com/mongodb/mongo-go-driver), integrated with `zap` logging and providing a generic base repository pattern.

## Features

- **Connection Management**: Simplified connection setup with timeouts and health checks (Ping).
- **Integrated Logging**: Automatic logging of connection status and operations using `go.uber.org/zap`.
- **Generic Repository**: `BaseRepository[T]` implementation for standard CRUD operations.
- **Soft Delete**: Built-in support for soft deletes via `is_deleted` field.
- **Custom Queries**: Flexible support for custom BSON filters.

## Dependencies

- [go.mongodb.org/mongo-driver](https://github.com/mongodb/mongo-go-driver)
- [go.uber.org/zap](https://github.com/uber-go/zap)

## Installation

```bash
go get github.com/logistics-id/engine/ds/mongo
```

## Quick Start

### 1. Configuration & Initialization

You can configure the connection manually or use the default environment variable loader.

**Using Environment Variables:**

```bash
# .env
MONGODB_SERVER=localhost:27017
MONGODB_AUTH_USERNAME=admin
MONGODB_AUTH_PASSWORD=password
```

```go
package main

import (
    "os"
    "github.com/logistics-id/engine/ds/mongo"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewProduction()

    // Load config from env
    cfg := mongo.ConfigDefault("my_database")

    // Connect
    if err := mongo.NewConnection(cfg, logger); err != nil {
        logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
    }
    defer mongo.CloseConnection()

    // Application logic...
}
```

**Manual Configuration:**

```go
cfg := &mongo.Config{
    Server:   "localhost:27017",
    Username: "user",
    Password: "password",
    Database: "mydb",
    // Optional: Full connection string overrides other fields
    // Datasource: "mongodb://user:pass@localhost:27017/mydb",
}
```

## Base Repository

The `BaseRepository[T]` provides a standard, reusable interface for interacting with MongoDB collections. It leverages Go generics to provide type-safe CRUD operations.

### Initialization

```go
// NewBaseRepository creates a generic repository for a specific type T.
//
// col: The collection wrapper
// searchFields: List of fields to apply search filters on (for FindAll)
// enableSoftDelete: If true, filters out documents with is_deleted=true
repo := mongo.NewBaseRepository[User](col, []string{"name", "email"}, true)
```

### Core Concepts

#### Context Chaining (`WithContext`)
The repository is designed to be immutable regarding the context. To perform an operation with a specific `context.Context`, use `WithContext`. This follows the pattern:

```go
repo.WithContext(ctx).FindByID("...")
```

#### Custom Queries (`CustomQueryFn`)
For complex filtering, the repository uses a functional pattern `CustomQueryFn` which lets you modify the BSON filter directly.

```go
type CustomQueryFn func(filter bson.M) bson.M
```

### Methods Reference

#### `FindByID(id any) (*T, error)`
Finds a single document by its `_id`.
- Checks `is_deleted` if soft delete is enabled.
- Automatically converts string IDs to `primitive.ObjectID`.

```go
user, err := repo.WithContext(ctx).FindByID("64f8a5c2e1b2c3d4e5f6g7h8")
```

#### `Insert(entity *T) error`
Inserts a new document.
- Returns error if insert fails.

```go
err := repo.WithContext(ctx).Insert(&User{Name: "John"})
```

#### `Update(entity *T, fields ...string) error`
Updates specific fields of an entity.
- Uses reflection to map struct fields to BSON tags.
- Only updates the fields specified in the `fields` argument.

```go
// Update only name and email
err := repo.WithContext(ctx).Update(user, "name", "email")
```

#### `SoftDelete(id any) error`
Marks a document as deleted by setting `is_deleted: true`.
- Requires `enableSoftDelete` to be set to `true` during initialization.

```go
err := repo.WithContext(ctx).SoftDelete(userID)
```

#### `FindOne(query CustomQueryFn) (*T, error)`
Finds a single document matching criteria.

```go
user, err := repo.WithContext(ctx).FindOne(func(f bson.M) bson.M {
    f["email"] = "john@example.com"
    f["status"] = "active"
    return f
})
```

#### `FindAll(opts *common.QueryOption, query CustomQueryFn) ([]*T, int64, error)`
Retrieves a list of documents with pagination, sorting, and filtering.

- **opts**: Contains `Limit`, `Offset`, `Orders` (sort), and `Search`.
- **query**: Optional extra filters.

```go
import "github.com/logistics-id/engine/common"

opts := &common.QueryOption{
    Limit: 20,
    Offset: 0,
    Orders: []string{"-created_at"}, // Descending by created_at
}

users, total, err := repo.WithContext(ctx).FindAll(opts, func(f bson.M) bson.M {
    f["role"] = "admin"
    return f
})
```

### Custom Repository Pattern

For complex logic, extend the base repository:

```go
type UserRepository struct {
    *mongo.BaseRepository[User]
}

func NewUserRepository() *UserRepository {
    return &UserRepository{
        BaseRepository: mongo.NewBaseRepository[User](
            mongo.NewCollection("users"),
            []string{"name", "email"},
            true,
        ),
    }
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
    return r.WithContext(ctx).FindOne(func(f bson.M) bson.M {
        f["email"] = email
        return f
    })
}
```
