# PostgreSQL Data Source Library

A PostgreSQL database client library built on top of [Bun ORM](https://bun.uptrace.dev/) with integrated logging, generic repository pattern, and helper utilities.

## Features

- Connection management with automatic health checks
- Singleton pattern support for global database access
- Generic base repository with CRUD operations
- Soft delete support
- Search and filtering utilities
- Query logging with Zap logger
- Transaction support
- JSON field sorting support
- Environment-based configuration

## Dependencies

- [uptrace/bun](https://github.com/uptrace/bun) - SQL-first Golang ORM
- [go.uber.org/zap](https://github.com/uber-go/zap) - Blazing fast, structured logging
- [logistics-id/engine/common](../common) - Common interfaces and utilities

## Installation

```bash
go get github.com/logistics-id/engine/ds/postgres
```

## Quick Start

### 1. Basic Client Usage

```go
package main

import (
    "github.com/logistics-id/engine/ds/postgres"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewProduction()

    // Configure connection
    config := &postgres.Config{
        Server:   "localhost:5432",
        Username: "postgres",
        Password: "secret",
        Database: "mydb",
    }

    // Or use DSN directly
    config.Datasource = "postgres://user:pass@localhost:5432/mydb?sslmode=disable"

    // Create client
    client, err := postgres.NewClient(config, logger)
    if err != nil {
        logger.Fatal("Failed to connect", zap.Error(err))
    }
    defer client.Close()

    // Get Bun DB instance
    db := client.GetDB()
    // Use db for queries...
}
```

### 2. Singleton Pattern (Recommended)

```go
package main

import (
    "github.com/logistics-id/engine/ds/postgres"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewProduction()

    // Initialize connection once
    config := postgres.ConfigDefault("mydb")
    if err := postgres.NewConnection(config, logger); err != nil {
        logger.Fatal("Failed to connect", zap.Error(err))
    }
    defer postgres.CloseConnection()

    // Access DB anywhere in your application
    db := postgres.GetDB()
    // Use db for queries...
}
```

### 3. Environment-Based Configuration

The `ConfigDefault` function reads configuration from environment variables:

```bash
# .env file
POSTGRES_SERVER=localhost:5432
POSTGRES_AUTH_USERNAME=postgres
POSTGRES_AUTH_PASSWORD=secret
```

```go
// Automatically reads from environment
config := postgres.ConfigDefault("mydb")
```

## API Reference

### Config

Configuration struct for database connection.

```go
type Config struct {
    Server     string // Host or IP of the Postgres server (e.g., "localhost:5432")
    Username   string // Database username
    Password   string // Database password
    Database   string // Database name
    Datasource string // Full DSN string (overrides Server/Username/Password/Database)
}
```

### Client

Main database client with connection management.

#### NewClient

```go
func NewClient(cfg *Config, l *zap.Logger) (*Client, error)
```

Creates a new PostgreSQL client with the given configuration and logger. Automatically:
- Tests connection with Ping
- Adds query logging hook
- Logs connection status

#### Methods

```go
func (c *Client) GetDB() *bun.DB       // Returns Bun DB instance
func (c *Client) Close() error          // Closes the database connection
```

### Wrapper Functions (Singleton Pattern)

#### NewConnection

```go
func NewConnection(c *Config, l *zap.Logger) error
```

Initializes a global database connection.

#### ConfigDefault

```go
func ConfigDefault(db string) *Config
```

Creates a config from environment variables (`POSTGRES_SERVER`, `POSTGRES_AUTH_USERNAME`, `POSTGRES_AUTH_PASSWORD`).

#### GetDB

```go
func GetDB() *bun.DB
```

Returns the globally initialized database instance.

#### CloseConnection

```go
func CloseConnection() error
```

Closes the global database connection.

## Base Repository

Generic repository pattern for common CRUD operations.

### Creating a Repository

```go
type User struct {
    ID        int64  `bun:"id,pk,autoincrement"`
    Name      string `bun:"name"`
    Email     string `bun:"email"`
    IsDeleted bool   `bun:"is_deleted,default:false"`
}

// Create repository directly (simple usage)
repo := postgres.NewBaseRepository[User](
    db,
    "users",                    // table name
    []string{"users.name", "users.email"}, // searchable fields
    []string{"Profile"},        // default relations to load
    true,                       // enable soft delete
)
```

### Extending the Repository (Recommended Pattern)

For better code organization and to add custom methods, it's recommended to create a dedicated repository struct that embeds the BaseRepository:

```go
package repository

import (
    "github.com/logistics-id/engine/ds/postgres"
    "github.com/uptrace/bun"
)

// Define your model
type User struct {
    ID        int64  `bun:"id,pk,autoincrement"`
    Name      string `bun:"name"`
    Email     string `bun:"email"`
    IsDeleted bool   `bun:"is_deleted,default:false"`
}

// Create custom repository struct
type UserRepository struct {
    *postgres.BaseRepository[User]
}

// Constructor function
func NewUserRepository(db *bun.DB) *UserRepository {
    return &UserRepository{
        BaseRepository: postgres.NewBaseRepository[User](
            db,
            "users",                           // table name
            []string{"users.name", "users.email"}, // searchable fields
            []string{"Profile"},               // default relations
            true,                              // enable soft delete
        ),
    }
}

// Override WithContext to return concrete type for method chaining
func (r *UserRepository) WithContext(ctx context.Context) *UserRepository {
    return &UserRepository{
        BaseRepository: r.BaseRepository.WithCtx(ctx),
    }
}

// Override WithTx to return concrete type for transaction chaining
func (r *UserRepository) WithTx(ctx context.Context, tx bun.Tx) *UserRepository {
    return &UserRepository{
        BaseRepository: r.BaseRepository.WithTx(ctx, tx),
    }
}

// Override RunInTxWithRepo for convenient single-repository transactions
func (r *UserRepository) RunInTxWithRepo(ctx context.Context, fn func(*UserRepository) error) error {
    return r.BaseRepository.RunInTx(ctx, func(ctx context.Context, tx bun.Tx) error {
        repoWithTx := r.WithTx(ctx, tx)
        return fn(repoWithTx)
    })
}

// Add custom methods to your repository
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
    return r.WithContext(ctx).FindOne(func(q *bun.SelectQuery) *bun.SelectQuery {
        return q.Where("email = ?", email)
    })
}

func (r *UserRepository) FindActiveUsers(ctx context.Context, opts *common.QueryOption) ([]*User, int64, error) {
    return r.WithContext(ctx).FindAll(opts, func(q *bun.SelectQuery) *bun.SelectQuery {
        return q.Where("status = ?", "active")
    })
}
```

Usage:

```go
// Initialize repository
userRepo := repository.NewUserRepository(postgres.GetDB())

// Use base repository methods
user, err := userRepo.WithContext(ctx).FindByID(1)

// Use custom methods
user, err := userRepo.FindByEmail(ctx, "john@example.com")
activeUsers, total, err := userRepo.FindActiveUsers(ctx, opts)
```

This pattern provides:
- Clean separation of concerns
- Type safety with generics
- Ability to add domain-specific methods
- Consistent repository interface across your application

### Repository Methods

#### WithContext

```go
func (r *BaseRepository[T]) WithContext(ctx context.Context) common.BaseRepositoryInterface[T]
```

Creates a new repository instance with the given context. Returns the interface type.

#### WithCtx

```go
func (r *BaseRepository[T]) WithCtx(ctx context.Context) *BaseRepository[T]
```

Creates a new repository instance with the given context, returning the concrete type. This method is useful for:
- Custom repositories that need to override `WithContext` to return their own type
- Method chaining with postgres-specific methods like `FindOne` and `FindAll`

Example:
```go
// In custom repository
func (r *UserRepository) WithContext(ctx context.Context) *UserRepository {
    return &UserRepository{
        BaseRepository: r.BaseRepository.WithCtx(ctx),
    }
}
```

#### WithTx

```go
func (r *BaseRepository[T]) WithTx(ctx context.Context, tx bun.Tx) *BaseRepository[T]
```

Creates a new repository instance with a transaction, returning the concrete type. This allows:
- Method chaining with postgres-specific methods within transactions
- Access to `FindOne`, `FindAll`, and other postgres-specific operations

Example:
```go
err := repo.RunInTx(ctx, func(ctx context.Context, tx bun.Tx) error {
    // Use WithTx and chain postgres-specific methods
    user, err := repo.WithTx(ctx, tx).FindOne(func(q *bun.SelectQuery) *bun.SelectQuery {
        return q.Where("email = ?", email)
    })
    if err != nil {
        return err
    }

    user.Status = "verified"
    return repo.WithTx(ctx, tx).Update(user)
})
```

#### Insert

```go
func (r *BaseRepository[T]) Insert(entity *T) error
```

Inserts a new entity into the database.

```go
user := &User{Name: "John", Email: "john@example.com"}
err := repo.WithContext(ctx).Insert(user)
```

#### FindByID

```go
func (r *BaseRepository[T]) FindByID(id any) (*T, error)
```

Finds an entity by its ID. Automatically:
- Filters soft-deleted records (if enabled)
- Loads default relations

```go
user, err := repo.WithContext(ctx).FindByID(1)
```

#### Update

```go
func (r *BaseRepository[T]) Update(entity *T, fields ...string) error
```

Updates an entity. Optionally specify fields to update.

```go
// Update all fields
user.Name = "Jane"
err := repo.WithContext(ctx).Update(user)

// Update specific fields only
err := repo.WithContext(ctx).Update(user, "name", "email")
```

#### SoftDelete

```go
func (r *BaseRepository[T]) SoftDelete(id any) error
```

Soft deletes an entity by setting `is_deleted = true`.

```go
err := repo.WithContext(ctx).SoftDelete(1)
```

#### FindAll

```go
func (r *BaseRepository[T]) FindAll(
    opts *common.QueryOption,
    customQuery CustomQueryFn,
) ([]*T, int64, error)
```

Finds all entities with pagination, search, and filtering support.

```go
opts := &common.QueryOption{
    Page:   1,
    Limit:  10,
    Search: "john",
    Orders: []string{"-created_at"}, // DESC order
    Conditions: []any{"status = ?", "active"},
}

users, total, err := repo.WithContext(ctx).FindAll(opts, nil)
```

With custom query:

```go
users, total, err := repo.WithContext(ctx).FindAll(opts, func(q *bun.SelectQuery) *bun.SelectQuery {
    return q.Where("age > ?", 18)
})
```

#### FindOne

```go
func (r *BaseRepository[T]) FindOne(customQuery CustomQueryFn) (*T, error)
```

Finds a single entity with custom query conditions.

```go
user, err := repo.WithContext(ctx).FindOne(func(q *bun.SelectQuery) *bun.SelectQuery {
    return q.Where("email = ?", "john@example.com")
})
```

#### RunInTx

```go
func (r *BaseRepository[T]) RunInTx(ctx context.Context, fn func(context.Context, bun.Tx) error) error
```

Executes a function within a transaction with full control. Use this when you need to work with **multiple different repositories** in the same transaction.

```go
err := userRepo.RunInTx(ctx, func(ctx context.Context, tx bun.Tx) error {
    // Create repository instances with the transaction
    userTxRepo := userRepo.WithTx(ctx, tx)
    orderTxRepo := orderRepo.WithTx(ctx, tx)

    // Use multiple repositories
    if err := userTxRepo.Insert(&user); err != nil {
        return err
    }
    if err := orderTxRepo.Insert(&order); err != nil {
        return err
    }

    return nil // Auto-commit on nil, rollback on error
})
```

#### RunInTxWithRepo

```go
func (r *BaseRepository[T]) RunInTxWithRepo(ctx context.Context, fn func(*BaseRepository[T]) error) error
```

Convenience method that automatically passes a repository with transaction context. Use this for **simpler cases with a single repository**.

```go
// Simpler - repository with transaction is passed automatically
err := userRepo.RunInTxWithRepo(ctx, func(txRepo *BaseRepository[User]) error {
    if err := txRepo.Insert(&user1); err != nil {
        return err
    }
    if err := txRepo.Insert(&user2); err != nil {
        return err
    }
    return nil
})
```

For custom repositories, override this method to return your concrete type:

```go
// In UserRepository
func (r *UserRepository) RunInTxWithRepo(ctx context.Context, fn func(*UserRepository) error) error {
    return r.BaseRepository.RunInTx(ctx, func(ctx context.Context, tx bun.Tx) error {
        repoWithTx := r.WithTx(ctx, tx)
        return fn(repoWithTx)
    })
}

// Usage with custom repository
err := userRepo.RunInTxWithRepo(ctx, func(txRepo *UserRepository) error {
    // txRepo is already *UserRepository with transaction
    user, err := txRepo.FindByEmail("john@example.com")
    if err != nil {
        return err
    }
    user.Status = "verified"
    return txRepo.Update(user)
})
```

## Utility Functions

### FilterSearch

```go
func FilterSearch(q *bun.SelectQuery, search string, fields ...string)
```

Adds case-insensitive search filter across multiple fields.

```go
q := db.NewSelect().Model(&users)
postgres.FilterSearch(q, "john", "users.name", "users.email")
// Generates: WHERE (users.name ILIKE '%john%' OR users.email ILIKE '%john%')
```

### RequestSort

```go
func RequestSort(sort []string) string
```

Converts sort parameters into SQL ORDER BY clause with JSON field support.

```go
// Simple sorting
sorts := []string{"-created_at", "name"}
order := postgres.RequestSort(sorts)
// Returns: "created_at DESC, name ASC"

// JSON field sorting
sorts := []string{"metadata__priority"}
order := postgres.RequestSort(sorts)
// Returns: "metadata->>'priority' ASC"

// Relation field sorting
sorts := []string{"profile:name"}
order := postgres.RequestSort(sorts)
// Returns: "profile.name ASC"
```

Sort syntax:
- Prefix with `-` for DESC order
- Use `__` to access JSON fields
- Use `:` to access relation fields

## Query Logging

The library includes automatic query logging via `ZapQueryHook`:

```go
type ZapQueryHook struct {
    Logger *zap.Logger
}
```

Logs include:
- Query operation (SELECT, INSERT, UPDATE, DELETE)
- SQL query string
- Execution duration
- Request ID from context
- Errors (if any)

Sample log output:
```json
{
  "level": "info",
  "msg": "PG/QUERY",
  "event": "SELECT",
  "query": "SELECT * FROM users WHERE id = $1",
  "request_id": "abc-123",
  "duration": "2.5ms"
}
```

## Error Handling

The library provides custom errors:

```go
var ErrClientNotInitialized = errors.New("db client not initialized; call NewConnection first")
```

This error is returned when:
- `GetDB()` is called before `NewConnection()`
- `CloseConnection()` is called before initialization

## Best Practices

### 1. Use Extended Repository Pattern

Create dedicated repository structs for better code organization and reusability:

```go
// repository/user_repository.go
type UserRepository struct {
    *postgres.BaseRepository[User]
}

func NewUserRepository(db *bun.DB) *UserRepository {
    return &UserRepository{
        BaseRepository: postgres.NewBaseRepository[User](
            db, "users",
            []string{"users.name", "users.email"},
            []string{},
            true,
        ),
    }
}

// Add domain-specific methods
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
    return r.WithContext(ctx).FindOne(func(q *bun.SelectQuery) *bun.SelectQuery {
        return q.Where("email = ?", email)
    })
}
```

### 2. Use Singleton Pattern for Application-Wide Access

```go
// Initialize once in main.go
func initDB() {
    logger, _ := zap.NewProduction()
    config := postgres.ConfigDefault("mydb")
    if err := postgres.NewConnection(config, logger); err != nil {
        log.Fatal(err)
    }
}

// Access anywhere
func someHandler() {
    db := postgres.GetDB()
    // Use db...
}
```

### 3. Always Use Context

```go
repo := repo.WithContext(r.Context())
user, err := repo.FindByID(1)
```

### 4. Use Transactions for Multiple Operations

```go
err := repo.RunInTx(ctx, func(ctx context.Context, tx bun.Tx) error {
    repoWithTx := repo.WithTx(ctx, tx)

    // Multiple operations...

    return nil // Auto-commit on nil, rollback on error
})
```

### 5. Leverage Soft Delete for Audit Trail

```go
repo := postgres.NewBaseRepository[User](
    db,
    "users",
    []string{"users.name"},
    []string{},
    true, // Enable soft delete
)

// Soft delete instead of hard delete
repo.WithContext(ctx).SoftDelete(userID)
```

### 6. Use Custom Queries for Complex Filters

```go
users, total, err := repo.WithContext(ctx).FindAll(opts, func(q *bun.SelectQuery) *bun.SelectQuery {
    return q.
        Where("age >= ?", 18).
        Where("status = ?", "active").
        WhereGroup(" OR ", func(sq *bun.SelectQuery) *bun.SelectQuery {
            return sq.
                Where("country = ?", "US").
                Where("country = ?", "CA")
        })
})
```

## Examples

### Complete CRUD Example

```go
package main

import (
    "context"
    "log"

    "github.com/logistics-id/engine/common"
    "github.com/logistics-id/engine/ds/postgres"
    "github.com/uptrace/bun"
    "go.uber.org/zap"
)

type Product struct {
    ID          int64   `bun:"id,pk,autoincrement"`
    Name        string  `bun:"name"`
    Description string  `bun:"description"`
    Price       float64 `bun:"price"`
    IsDeleted   bool    `bun:"is_deleted,default:false"`
}

// ProductRepository extends BaseRepository
type ProductRepository struct {
    *postgres.BaseRepository[Product]
}

func NewProductRepository(db *bun.DB) *ProductRepository {
    return &ProductRepository{
        BaseRepository: postgres.NewBaseRepository[Product](
            db,
            "products",
            []string{"products.name", "products.description"},
            []string{},
            true,
        ),
    }
}

// Add custom method
func (r *ProductRepository) FindByName(ctx context.Context, name string) (*Product, error) {
    return r.WithContext(ctx).FindOne(func(q *bun.SelectQuery) *bun.SelectQuery {
        return q.Where("name = ?", name)
    })
}

func main() {
    logger, _ := zap.NewProduction()
    ctx := context.Background()

    // Initialize connection
    config := postgres.ConfigDefault("shop")
    if err := postgres.NewConnection(config, logger); err != nil {
        log.Fatal(err)
    }
    defer postgres.CloseConnection()

    // Create repository
    repo := NewProductRepository(postgres.GetDB())

    // Create
    product := &Product{
        Name:        "Laptop",
        Description: "High-performance laptop",
        Price:       999.99,
    }
    if err := repo.WithContext(ctx).Insert(product); err != nil {
        log.Fatal(err)
    }

    // Read by ID
    found, err := repo.WithContext(ctx).FindByID(product.ID)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Found: %+v\n", found)

    // Update
    found.Price = 899.99
    if err := repo.WithContext(ctx).Update(found, "price"); err != nil {
        log.Fatal(err)
    }

    // Search and list with pagination
    opts := &common.QueryOption{
        Page:   1,
        Limit:  10,
        Search: "laptop",
        Orders: []string{"-price"},
    }
    products, total, err := repo.WithContext(ctx).FindAll(opts, nil)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Found %d products (total: %d)\n", len(products), total)

    // Use custom repository method
    productByName, err := repo.FindByName(ctx, "Laptop")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Found by name: %+v\n", productByName)

    // Soft delete
    if err := repo.WithContext(ctx).SoftDelete(product.ID); err != nil {
        log.Fatal(err)
    }
}
```

### Transaction Examples

#### Simple Transaction (Single Repository)

Using `RunInTxWithRepo` for cleaner code:

```go
func transferFunds(ctx context.Context, fromID, toID int64, amount float64) error {
    repo := postgres.NewBaseRepository[Account](
        postgres.GetDB(),
        "accounts",
        []string{},
        []string{},
        false,
    )

    // Simpler pattern - repository with transaction passed automatically
    return repo.RunInTxWithRepo(ctx, func(txRepo *postgres.BaseRepository[Account]) error {
        // Debit from account
        fromAccount, err := txRepo.FindByID(fromID)
        if err != nil {
            return err
        }
        fromAccount.Balance -= amount
        if err := txRepo.Update(fromAccount, "balance"); err != nil {
            return err
        }

        // Credit to account
        toAccount, err := txRepo.FindByID(toID)
        if err != nil {
            return err
        }
        toAccount.Balance += amount
        if err := txRepo.Update(toAccount, "balance"); err != nil {
            return err
        }

        return nil // Auto-commit on success
    })
}
```

#### Complex Transaction (Multiple Repositories)

Using `RunInTx` when you need multiple repositories:

```go
func createOrderWithInventory(ctx context.Context, order *Order, items []OrderItem) error {
    orderRepo := NewOrderRepository(postgres.GetDB())
    inventoryRepo := NewInventoryRepository(postgres.GetDB())

    return orderRepo.RunInTx(ctx, func(ctx context.Context, tx bun.Tx) error {
        // Get both repositories with the same transaction
        orderTxRepo := orderRepo.WithTx(ctx, tx)
        inventoryTxRepo := inventoryRepo.WithTx(ctx, tx)

        // Create order
        if err := orderTxRepo.Insert(order); err != nil {
            return err
        }

        // Update inventory for each item
        for _, item := range items {
            inventory, err := inventoryTxRepo.FindByID(item.ProductID)
            if err != nil {
                return err
            }

            if inventory.Quantity < item.Quantity {
                return errors.New("insufficient inventory")
            }

            inventory.Quantity -= item.Quantity
            if err := inventoryTxRepo.Update(inventory, "quantity"); err != nil {
                return err
            }
        }

        return nil // All operations succeed - commit
    })
}
```

## Testing

When testing, you can create isolated client instances:

```go
func TestUserRepository(t *testing.T) {
    logger, _ := zap.NewDevelopment()

    config := &postgres.Config{
        Datasource: "postgres://test:test@localhost:5432/testdb?sslmode=disable",
    }

    client, err := postgres.NewClient(config, logger)
    if err != nil {
        t.Fatal(err)
    }
    defer client.Close()

    repo := postgres.NewBaseRepository[User](
        client.GetDB(),
        "users",
        []string{"users.name"},
        []string{},
        true,
    )

    // Run tests...
}
```

## License

This library is part of the Enigma Engine project.
