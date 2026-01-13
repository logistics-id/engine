# Common Utilities Library

A collection of shared types, interfaces, and utility functions used throughout the engine and microservices.

## Features

- **JWT Management**: unified `SessionClaims` structure and token encoding/decoding.
- **Random Generation**: helpers for generating secure random alphanumeric codes.
- **Base Interfaces**: Standard interfaces for Repositories and UseCases.
- **Context Keys**: Standardized keys for storing/retrieving values from `context.Context`.

## API Reference

### JWT & Session

#### `SessionClaims`
Standard claims structure for authenticated users.

```go
type SessionClaims struct {
    UserID      string   `json:"user_id"`
    Username    string   `json:"username"`
    Permissions []string `json:"permission"`
    // ...other fields
}
```

#### `TokenEncode` / `TokenDecode`

```go
// Create a token pair (access + refresh)
claims := &common.SessionClaims{UserID: "123", ...}
tokens, err := common.TokenEncode(claims)

// Decode a token string
claims, err := common.TokenDecode(tokenStr)
```

#### `GetSession`

Retrieve claims from context (usually set by middleware).

```go
claims, err := common.GetSession(ctx)
if err == nil {
    fmt.Println(claims.UserID)
}
```

### Random Generation

#### `RandomCode`

Generates cryptographically secure random strings.

```go
// Generate 6-digit numeric code
code := common.RandomCode(6, common.RandomCodeNumeric) // "123456"

// Generate 12-char alphanumeric code with separators every 4 chars
// Output example: "ABCD-1234-EFGH"
code := common.RandomCode(12, common.RandomCodeAlphaNumeric, "-", 4)
```

### Context Helpers

```go
// Get Request ID from context
reqID := common.GetContextRequestID(ctx)
```
