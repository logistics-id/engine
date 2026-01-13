# Validation Library

A flexible, tag-based struct validator and assertion library.

## Features

- **Tag-based Validation**: Validate struct fields using `valid:"tag"` syntax.
- **Rich Validator Set**: Includes checks for `required`, `email`, `numeric`, `len`, `uuid`, etc.
- **Custom Response**: Returns structured error messages useful for API responses.
- **Standalone Assertions**: Use validator functions directly without structs.

## Installation

```bash
go get github.com/logistics-id/engine/validate
```

## Usage

### Struct Validation

```go
type UserRequest struct {
    Name     string `json:"name" valid:"required|alpha_space"`
    Email    string `json:"email" valid:"required|email"`
    Age      int    `json:"age" valid:"numeric|gte:18"`
    Password string `json:"password" valid:"required|password"`
}

func ValidateUser(req UserRequest) {
    v := validate.New()

    // Validate struct
    response := v.Struct(req)

    if !response.Valid {
        // Get generic error strings
        fmt.Println(response.GetFailures())
        // Output: map[email:email is not valid age:age must be greater than or equal 18]
    }
}
```

### Standalone Assertions

You can use the `validate` (or alias `assert` depending on import) functions directly.

```go
import "github.com/logistics-id/engine/validate"

if !validate.IsEmail("test@example.com") {
    // handle error
}

if validate.IsNumeric("12345") {
    // it is numeric
}
```

### Available Tags

| Tag | Description | Example |
|---|---|---|
| `required` | Field cannot be zero-value | `valid:"required"` |
| `email` | Must be a valid email | `valid:"email"` |
| `numeric` | Must contain only numbers | `valid:"numeric"` |
| `alpha` | Must contain only letters | `valid:"alpha"` |
| `uuid` | Must be a valid UUID | `valid:"uuid"` |
| `min:x` | Min length/value | `valid:"min:10"` |
| `max:x` | Max length/value | `valid:"max:20"` |
| `gte:x` | Greater than or equal | `valid:"gte:18"` |
| `lte:x` | Less than or equal | `valid:"lte:100"` |
| `oneof` | Must be one of values | `valid:"oneof:A,B,C"` |
| `password` | Complex password check | `valid:"password"` |

### Customizing Messages

Implement the `Request` interface to provide custom error messages.

```go
func (r UserRequest) Messages() map[string]string {
    return map[string]string{
        "required": "The %s field is mandatory.",
        "email":    "Please provide a valid email address.",
    }
}

func (r UserRequest) Validate() *validate.Response {
    // Custom logic if needed
    return nil
}

// Then use v.Request(req) instead of v.Struct(req)
res := v.Request(req)
```
