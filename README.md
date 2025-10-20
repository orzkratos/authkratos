[![GitHub Workflow Status (branch)](https://img.shields.io/github/actions/workflow/status/orzkratos/authkratos/release.yml?branch=main&label=BUILD)](https://github.com/orzkratos/authkratos/actions/workflows/release.yml?query=branch%3Amain)
[![GoDoc](https://pkg.go.dev/badge/github.com/orzkratos/authkratos)](https://pkg.go.dev/github.com/orzkratos/authkratos)
[![Coverage Status](https://img.shields.io/coveralls/github/orzkratos/authkratos/main.svg)](https://coveralls.io/github/orzkratos/authkratos?branch=main)
[![Supported Go Versions](https://img.shields.io/badge/Go-1.25+-lightgrey.svg)](https://go.dev/)
[![GitHub Release](https://img.shields.io/github/release/orzkratos/authkratos.svg)](https://github.com/orzkratos/authkratos/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/orzkratos/authkratos)](https://goreportcard.com/report/github.com/orzkratos/authkratos)

# authkratos

Kratos authentication middleware collection with route scope management and APM tracing support.

---

<!-- TEMPLATE (EN) BEGIN: LANGUAGE NAVIGATION -->
## CHINESE README

[中文说明](README.zh.md)
<!-- TEMPLATE (EN) END: LANGUAGE NAVIGATION -->

## Main Features

🎯 **Token Authentication**: Simple and pre-configured token-based auth with custom validation
⚡ **Route Scope Control**: Flexible INCLUDE/EXCLUDE mode route filtering
🔄 **Rate Limiting**: Redis-backed distributed rate limiting with context-based ID extraction
🌍 **Random Sampling**: Probabilistic request sampling and blocking with configurable odds
📋 **Timeout Management**: Selective timeout override on specific routes
⏱️ **Periodic Throttling**: Counter-based deterministic request sampling
🔍 **APM Tracing**: Built-in APM span tracking with configurable naming

## Installation

```bash
go get github.com/orzkratos/authkratos
```

## Quick Start

### Token Authentication

```go
import (
    "github.com/orzkratos/authkratos/authkratostokens"
    "github.com/orzkratos/authkratos/authkratosroutes"
)

// Create auth middleware with username-token map
cfg := authkratostokens.NewConfig(
    authkratosroutes.NewInclude(
        "/api.Service/CreateUser",
        "/api.Service/UpdateUser",
    ),
    map[string]string{
        "alice": "secret-token-123",
        "bob":   "another-token-456",
    },
)

middleware := authkratostokens.NewMiddleware(cfg, logger)
```

### Simple Custom Auth

```go
import (
    "github.com/orzkratos/authkratos/authkratossimple"
    "github.com/orzkratos/authkratos/authkratosroutes"
)

// Custom token validation function
checkToken := func(ctx context.Context, token string) (context.Context, *errors.Error) {
    // Validate token and inject account data into context
    if account := validateToken(token); account != nil {
        ctx = context.WithValue(ctx, "account", account)
        return ctx, nil
    }
    return ctx, errors.Unauthorized("INVALID_TOKEN", "token is invalid")
}

cfg := authkratossimple.NewConfig(
    authkratosroutes.NewInclude("/api.Service/ProtectedMethod"),
    checkToken,
)

middleware := authkratossimple.NewMiddleware(cfg, logger)
```

### Rate Limiting

```go
import (
    "github.com/orzkratos/authkratos/ratekratoslimits"
    "github.com/go-redis/redis_rate/v10"
    "github.com/redis/go-redis/v9"
)

// Redis-backed rate limiting
rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
limiter := redis_rate.NewLimiter(rdb)
limit := redis_rate.PerMinute(100) // 100 requests each minute

// Extract unique ID from context (e.g., account ID)
keyFromCtx := func(ctx context.Context) (string, bool) {
    if account, ok := ctx.Value("account").(string); ok {
        return account, true
    }
    return "", false
}

cfg := ratekratoslimits.NewConfig(
    authkratosroutes.NewInclude("/api.Service/ExpensiveOperation"),
    limiter,
    &limit,
    keyFromCtx,
)

middleware := ratekratoslimits.NewMiddleware(cfg, logger)
```

### Random Sampling

```go
import "github.com/orzkratos/authkratos/matchkratosrandom"

// Match 60% of requests at random
cfg := matchkratosrandom.NewConfig(
    authkratosroutes.NewExclude("/api.Service/HealthCheck"),
    0.6, // 60% sampling rate
)

matchFunc := matchkratosrandom.NewMatchFunc(cfg, logger)

// Use with selector middleware
middleware := selector.Server(yourMiddleware).Match(matchFunc).Build()
```

### Chaos Testing

```go
import "github.com/orzkratos/authkratos/passkratosrandom"

// Block 40% of requests at random (pass-through rate: 60%)
cfg := passkratosrandom.NewConfig(
    authkratosroutes.NewInclude("/api.Service/TestMethod"),
    0.6, // 60% pass rate
)

middleware := passkratosrandom.NewMiddleware(cfg, logger)
```

### Timeout Management

```go
import (
    "github.com/orzkratos/authkratos/fastkratoshandle"
    "time"
)

// Set 5-second timeout on specific routes
cfg := fastkratoshandle.NewConfig(
    authkratosroutes.NewInclude("/api.Service/QuickOperation"),
    5*time.Second,
)

middleware := fastkratoshandle.NewMiddleware(cfg, logger)
```

### Periodic Sampling

```go
import "github.com/orzkratos/authkratos/matchkratosperiod"

// Match each 10th request (10% sampling)
cfg := matchkratosperiod.NewConfig(
    authkratosroutes.NewExclude("/api.Service/Monitoring"),
    10, // Period: match each 10th request
)

matchFunc := matchkratosperiod.NewMatchFunc(cfg, logger)
```

## Package Overview

| Package | Purpose |
|---------|---------|
| `authkratostokens` | Pre-configured token auth with username-token map |
| `authkratossimple` | Custom token validation with flexible auth logic |
| `ratekratoslimits` | Redis-backed distributed rate limiting |
| `passkratosrandom` | Probabilistic request blocking (chaos testing) |
| `fastkratoshandle` | Selective timeout override on specific routes |
| `matchkratosrandom` | Random request sampling match function |
| `matchkratosperiod` | Periodic request sampling (each Nth request) |
| `authkratosroutes` | Route scope matching toolkit |

## Advanced Features

### Route Scope Modes

```go
// INCLUDE mode: Just match specified operations
include := authkratosroutes.NewInclude(
    "/api.Service/CreateUser",
    "/api.Service/UpdateUser",
    "/api.Service/DeleteUser",
)

// EXCLUDE mode: Match except specified operations
exclude := authkratosroutes.NewExclude(
    "/api.Service/HealthCheck",
    "/api.Service/Metrics",
)

// Toggle between modes
opposite := include.Opposite() // Converts to EXCLUDE mode
```

### APM Tracing

```go
// Enable APM tracing with default span name
cfg := authkratostokens.NewConfig(routeScope, tokens).
    WithDefaultApmSpanName() // Uses "auth-kratos-tokens"

// Custom APM span name
cfg := authkratostokens.NewConfig(routeScope, tokens).
    WithApmSpanName("custom-auth-span").
    WithApmMatchSuffix("-matching") // Suffix: "custom-auth-span-matching"
```

### Debug Mode

```go
import "github.com/orzkratos/authkratos"

// Turn on debug logging
authkratos.SetDebugMode(true)

// Each middleware outputs extensive debug logging
```

### Request Field Configuration

```go
// Use custom request field name (default: "Authorization")
cfg := authkratostokens.NewConfig(routeScope, tokens).
    WithFieldName("X-API-Token")

// Get configured field name
fieldName := cfg.GetFieldName() // "X-API-Token"
```

### Token Formats

The `authkratostokens` package supports various token formats, each must be explicitly enabled:

```go
tokens := map[string]string{
    "alice": "secret-token",
}

// Enable token types you need (disabled as default, must enable each type)
cfg := authkratostokens.NewConfig(routeScope, tokens).
    WithEnableSimpleType().  // Enable simple format: "secret-token"
    WithEnableBearerType().  // Enable Bearer format: "Bearer secret-token"
    WithEnableBase64Type()   // Enable Basic Auth: "Basic YWxpY2U6c2VjcmV0LXRva2Vu"

// Can enable specific types, e.g., Bearer:
cfg := authkratostokens.NewConfig(routeScope, tokens).
    WithEnableBearerType()  // Accept just "Bearer secret-token" format

// Three token formats:
// 1. Simple: "secret-token"
// 2. Bearer: "Bearer secret-token"
// 3. Basic Auth: "Basic YWxpY2U6c2VjcmV0LXRva2Vu" (base64 of "alice:secret-token")
```

### Extract Username from Context

```go
import "github.com/orzkratos/authkratos/authkratostokens"

// Get authenticated username in the request context
username, ok := authkratostokens.GetUsername(ctx)
if ok {
    // Use username in business logic
}
```

## Testing

```bash
# Run tests
go test -v ./...

# Run tests with coverage
go test -v -cover ./...

# Run specific package tests
go test -v ./authkratostokens/...
```

## Examples

See the [internal/examples](internal/) DIR on detailed usage examples.

<!-- TEMPLATE (EN) BEGIN: STANDARD PROJECT FOOTER -->
<!-- VERSION 2025-09-26 07:39:27.188023 +0000 UTC -->

## 📄 License

MIT License. See [LICENSE](LICENSE).

---

## 🤝 Contributing

Contributions are welcome! Report bugs, suggest features, and contribute code:

- 🐛 **Found a mistake?** Open an issue on GitHub with reproduction steps
- 💡 **Have a feature idea?** Create an issue to discuss the suggestion
- 📖 **Documentation confusing?** Report it so we can improve
- 🚀 **Need new features?** Share the use cases to help us understand requirements
- ⚡ **Performance issue?** Help us optimize through reporting slow operations
- 🔧 **Configuration problem?** Ask questions about complex setups
- 📢 **Follow project progress?** Watch the repo to get new releases and features
- 🌟 **Success stories?** Share how this package improved the workflow
- 💬 **Feedback?** We welcome suggestions and comments

---

## 🔧 Development

New code contributions, follow this process:

1. **Fork**: Fork the repo on GitHub (using the webpage UI).
2. **Clone**: Clone the forked project (`git clone https://github.com/yourname/repo-name.git`).
3. **Navigate**: Navigate to the cloned project (`cd repo-name`)
4. **Branch**: Create a feature branch (`git checkout -b feature/xxx`).
5. **Code**: Implement the changes with comprehensive tests
6. **Testing**: (Golang project) Ensure tests pass (`go test ./...`) and follow Go code style conventions
7. **Documentation**: Update documentation to support client-facing changes and use significant commit messages
8. **Stage**: Stage changes (`git add .`)
9. **Commit**: Commit changes (`git commit -m "Add feature xxx"`) ensuring backward compatible code
10. **Push**: Push to the branch (`git push origin feature/xxx`).
11. **PR**: Open a merge request on GitHub (on the GitHub webpage) with detailed description.

Please ensure tests pass and include relevant documentation updates.

---

## 🌟 Support

Welcome to contribute to this project via submitting merge requests and reporting issues.

**Project Support:**

- ⭐ **Give GitHub stars** if this project helps you
- 🤝 **Share with teammates** and (golang) programming friends
- 📝 **Write tech blogs** about development tools and workflows - we provide content writing support
- 🌟 **Join the ecosystem** - committed to supporting open source and the (golang) development scene

**Have Fun Coding with this package!** 🎉🎉🎉

<!-- TEMPLATE (EN) END: STANDARD PROJECT FOOTER -->

---

## GitHub Stars

[![Stargazers](https://starchart.cc/orzkratos/authkratos.svg?variant=adaptive)](https://starchart.cc/orzkratos/authkratos)
