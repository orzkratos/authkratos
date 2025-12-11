[![GitHub Workflow Status (branch)](https://img.shields.io/github/actions/workflow/status/orzkratos/authkratos/release.yml?branch=main&label=BUILD)](https://github.com/orzkratos/authkratos/actions/workflows/release.yml?query=branch%3Amain)
[![GoDoc](https://pkg.go.dev/badge/github.com/orzkratos/authkratos)](https://pkg.go.dev/github.com/orzkratos/authkratos)
[![Coverage Status](https://img.shields.io/coveralls/github/orzkratos/authkratos/main.svg)](https://coveralls.io/github/orzkratos/authkratos?branch=main)
[![Supported Go Versions](https://img.shields.io/badge/Go-1.25+-lightgrey.svg)](https://go.dev/)
[![GitHub Release](https://img.shields.io/github/release/orzkratos/authkratos.svg)](https://github.com/orzkratos/authkratos/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/orzkratos/authkratos)](https://goreportcard.com/report/github.com/orzkratos/authkratos)

# authkratos

Kratos 认证中间件集合，支持路由范围管理和 APM 追踪。

---

<!-- TEMPLATE (ZH) BEGIN: LANGUAGE NAVIGATION -->

## 英文文档

[ENGLISH README](README.md)
<!-- TEMPLATE (ZH) END: LANGUAGE NAVIGATION -->

## 核心特性

🎯 **令牌认证**: 简单和预配置的令牌认证，支持自定义验证
⚡ **路由范围过滤**: 灵活的 INCLUDE/EXCLUDE 模式路由匹配
🔄 **速率限制**: 基于 Redis 的分布式速率限制，支持基于上下文的 ID 提取
🌍 **随机采样**: 概率性请求采样和阻断，支持可配置的概率
📋 **超时管理**: 特定路由的选择性超时覆盖
⏱️ **周期性限流**: 基于计数器的确定性请求采样
🔍 **APM 追踪**: 内置 APM span 追踪，支持可配置命名

## 安装

```bash
go get github.com/orzkratos/authkratos
```

## 快速开始

### 令牌认证

```go
import (
    "github.com/orzkratos/authkratos/authkratostokens"
    "github.com/orzkratos/authkratos/authkratosroutes"
)

// 使用用户名-令牌映射创建认证中间件
cfg := authkratostokens.NewConfig(
    authkratosroutes.NewInclude(
        "/api.Service/CreateUser",
        "/api.Service/UpdateUser",
    ),
    map[string]string{
        "alice": "secret-token-123",
        "bruce": "another-token-456",
    },
)

middleware := authkratostokens.NewMiddleware(cfg, logger)
```

### 简单自定义认证

```go
import (
    "github.com/orzkratos/authkratos/authkratossimple"
    "github.com/orzkratos/authkratos/authkratosroutes"
)

// 自定义令牌验证函数
checkToken := func(ctx context.Context, token string) (context.Context, *errors.Error) {
    // 验证令牌并将账户数据注入上下文
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

### 速率限制

```go
import (
    "github.com/orzkratos/authkratos/ratekratoslimits"
    "github.com/go-redis/redis_rate/v10"
    "github.com/redis/go-redis/v9"
)

// 基于 Redis 的速率限制
rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
limiter := redis_rate.NewLimiter(rdb)
limit := redis_rate.PerMinute(100) // 每分钟 100 个请求

// 从上下文中提取唯一 ID（例如账户 ID）
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

### 随机采样

```go
import "github.com/orzkratos/authkratos/matchkratosrandom"

// 随机匹配 60% 的请求
cfg := matchkratosrandom.NewConfig(
    authkratosroutes.NewExclude("/api.Service/HealthCheck"),
    0.6, // 60% 采样率
)

matchFunc := matchkratosrandom.NewMatchFunc(cfg, logger)

// 与选择器中间件一起使用
middleware := selector.Server(yourMiddleware).Match(matchFunc).Build()
```

### 混沌测试

```go
import "github.com/orzkratos/authkratos/passkratosrandom"

// 随机阻断 40% 的请求（放行率：60%）
cfg := passkratosrandom.NewConfig(
    authkratosroutes.NewInclude("/api.Service/TestMethod"),
    0.6, // 60% 放行率
)

middleware := passkratosrandom.NewMiddleware(cfg, logger)
```

### 超时管理

```go
import (
    "github.com/orzkratos/authkratos/fastkratoshandle"
    "time"
)

// 对特定路由设置 5 秒超时
cfg := fastkratoshandle.NewConfig(
    authkratosroutes.NewInclude("/api.Service/QuickOperation"),
    5*time.Second,
)

middleware := fastkratoshandle.NewMiddleware(cfg, logger)
```

### 周期性采样

```go
import "github.com/orzkratos/authkratos/matchkratosperiod"

// 每 10 个请求匹配一次（10% 采样率）
cfg := matchkratosperiod.NewConfig(
    authkratosroutes.NewExclude("/api.Service/Monitoring"),
    10, // 周期：每 10 个请求匹配一次
)

matchFunc := matchkratosperiod.NewMatchFunc(cfg, logger)
```

## 包概览

| 包名 | 用途 |
|------|------|
| `authkratostokens` | 预配置令牌认证，支持用户名-令牌映射 |
| `authkratossimple` | 自定义令牌验证，灵活的认证逻辑 |
| `ratekratoslimits` | 基于 Redis 的分布式速率限制 |
| `passkratosrandom` | 概率性请求阻断（混沌测试） |
| `fastkratoshandle` | 特定路由的选择性超时覆盖 |
| `matchkratosrandom` | 随机请求采样匹配函数 |
| `matchkratosperiod` | 周期性请求采样（每 N 个请求） |
| `authkratosroutes` | 路由范围匹配工具包 |

## 高级功能

### 路由范围模式

```go
// INCLUDE 模式：仅匹配指定的操作
include := authkratosroutes.NewInclude(
    "/api.Service/CreateUser",
    "/api.Service/UpdateUser",
    "/api.Service/DeleteUser",
)

// EXCLUDE 模式：匹配除指定操作外的所有操作
exclude := authkratosroutes.NewExclude(
    "/api.Service/HealthCheck",
    "/api.Service/Metrics",
)

// 在模式之间切换
opposite := include.Opposite() // 转换成 EXCLUDE 模式
```

### APM 追踪

```go
// 使用默认 span 名称启用 APM 追踪
cfg := authkratostokens.NewConfig(routeScope, tokens).
    WithDefaultApmSpanName() // 使用 "auth-kratos-tokens"

// 自定义 APM span 名称
cfg := authkratostokens.NewConfig(routeScope, tokens).
    WithApmSpanName("custom-auth-span").
    WithApmMatchSuffix("-matching") // 后缀："custom-auth-span-matching"
```

### 调试模式

```go
import "github.com/orzkratos/authkratos"

// 启用调试日志
authkratos.SetDebugMode(true)

// 每个中间件会输出详细的调试日志
```

### 请求字段配置

```go
// 使用自定义请求字段名（默认："Authorization"）
cfg := authkratostokens.NewConfig(routeScope, tokens).
    WithFieldName("X-API-Token")

// 获取配置的字段名
fieldName := cfg.GetFieldName() // "X-API-Token"
```

### 令牌格式

`authkratostokens` 包支持多种令牌格式，每种格式需要显式启用：

```go
tokens := map[string]string{
    "alice": "secret-token",
}

// 启用需要的令牌类型（默认都关闭，需要显式启用）
cfg := authkratostokens.NewConfig(routeScope, tokens).
    WithEnableSimpleType().  // 启用简单格式："secret-token"
    WithEnableBearerType().  // 启用 Bearer 格式："Bearer secret-token"
    WithEnableBase64Type()   // 启用 Basic Auth："Basic YWxpY2U6c2VjcmV0LXRva2Vu"

// 可以只启用部分类型，如只启用 Bearer：
cfg := authkratostokens.NewConfig(routeScope, tokens).
    WithEnableBearerType()  // 仅接受 "Bearer secret-token" 格式

// 三种令牌格式：
// 1. 简单格式："secret-token"
// 2. Bearer 格式："Bearer secret-token"
// 3. Basic Auth："Basic YWxpY2U6c2VjcmV0LXRva2Vu" ("alice:secret-token" 的 base64)
```

### 从上下文中提取用户名

```go
import "github.com/orzkratos/authkratos/authkratostokens"

// 在请求上下文中获取已认证的用户名
username, ok := authkratostokens.GetUsername(ctx)
if ok {
    // 在业务逻辑中使用用户名
}
```

## 测试

```bash
# 运行测试
go test -v ./...

# 运行测试并生成覆盖率报告
go test -v -cover ./...

# 运行特定包的测试
go test -v ./authkratostokens/...
```

## 示例

查看 [internal/examples](internal/) 目录获取详细的使用示例。

<!-- TEMPLATE (ZH) BEGIN: STANDARD PROJECT FOOTER -->
<!-- VERSION 2025-11-25 03:52:28.131064 +0000 UTC -->

## 📄 许可证类型

MIT 许可证 - 详见 [LICENSE](LICENSE)。

---

## 💬 联系与反馈

非常欢迎贡献代码！报告 BUG、建议功能、贡献代码：

- 🐛 **问题报告？** 在 GitHub 上提交问题并附上重现步骤
- 💡 **新颖思路？** 创建 issue 讨论
- 📖 **文档疑惑？** 报告问题，帮助我们完善文档
- 🚀 **需要功能？** 分享使用场景，帮助理解需求
- ⚡ **性能瓶颈？** 报告慢操作，协助解决性能问题
- 🔧 **配置困扰？** 询问复杂设置的相关问题
- 📢 **关注进展？** 关注仓库以获取新版本和功能
- 🌟 **成功案例？** 分享这个包如何改善工作流程
- 💬 **反馈意见？** 欢迎提出建议和意见

---

## 🔧 代码贡献

新代码贡献，请遵循此流程：

1. **Fork**：在 GitHub 上 Fork 仓库（使用网页界面）
2. **克隆**：克隆 Fork 的项目（`git clone https://github.com/yourname/repo-name.git`）
3. **导航**：进入克隆的项目（`cd repo-name`）
4. **分支**：创建功能分支（`git checkout -b feature/xxx`）
5. **编码**：实现您的更改并编写全面的测试
6. **测试**：（Golang 项目）确保测试通过（`go test ./...`）并遵循 Go 代码风格约定
7. **文档**：面向用户的更改需要更新文档
8. **暂存**：暂存更改（`git add .`）
9. **提交**：提交更改（`git commit -m "Add feature xxx"`）确保向后兼容的代码
10. **推送**：推送到分支（`git push origin feature/xxx`）
11. **PR**：在 GitHub 上打开 Merge Request（在 GitHub 网页上）并提供详细描述

请确保测试通过并包含相关的文档更新。

---

## 🌟 项目支持

非常欢迎通过提交 Merge Request 和报告问题来贡献此项目。

**项目支持：**

- ⭐ **给予星标**如果项目对您有帮助
- 🤝 **分享项目**给团队成员和（golang）编程朋友
- 📝 **撰写博客**关于开发工具和工作流程 - 我们提供写作支持
- 🌟 **加入生态** - 致力于支持开源和（golang）开发场景

**祝你用这个包编程愉快！** 🎉🎉🎉

<!-- TEMPLATE (ZH) END: STANDARD PROJECT FOOTER -->

---

## GitHub 标星点赞

[![标星点赞](https://starchart.cc/orzkratos/authkratos.svg?variant=adaptive)](https://starchart.cc/orzkratos/authkratos)

