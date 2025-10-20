// Package authkratossimple: Simple token authentication middleware with custom validation
// Provides flexible auth middleware with custom token check functions and context injection
// Supports route scope filtering, APM tracing, and configurable request field names
// Injects authenticated data into context on success
//
// authkratossimple: 简单的令牌认证中间件，支持自定义验证
// 提供灵活的认证中间件，支持用户定义的令牌检查函数和上下文注入
// 支持路由范围过滤、APM 追踪和可配置的请求头字段名
// 认证成功时可以将用户信息注入到上下文中
package authkratossimple

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/orzkratos/authkratos"
	"github.com/orzkratos/authkratos/authkratosroutes"
	"github.com/orzkratos/authkratos/internal/utils"
	"github.com/yyle88/must"
	"github.com/yyle88/neatjson/neatjsons"
	"go.elastic.co/apm/v2"
)

// CheckTokenAndSetCtxFunc validates auth token and injects account data into context
// Parameters: ctx - current request context, token - authentication token
// Returns: new context (with account data if present) and validation status
// On success, account data gets injected into context accessible to downstream handlers
//
// CheckTokenAndSetCtxFunc 验证认证令牌并将用户信息注入上下文
// 参数：ctx - 当前请求上下文，token - 认证令牌
// 返回：新的 context（可能包含用户信息）和错误
// 认证成功时可以将用户信息注入到返回的 context 中，供后续处理程序使用
type CheckTokenAndSetCtxFunc func(ctx context.Context, token string) (context.Context, *errors.Error)

// Config holds the simple auth middleware configuration
// Combines route scope, token validation function, and APM settings
// Note: Avoid non-standard names in production (Nginx drops request fields with underscores unless configured)
//
// Config 保存简单认证中间件的配置
// 组合路由范围、令牌验证函数和 APM 设置
// 注意：生产环境避免非标准字段名（Nginx 默认丢弃带下划线的请求头，除非配置）
type Config struct {
	routeScope     *authkratosroutes.RouteScope // Route scope which auth applies to // 认证应用的路由范围
	checkToken     CheckTokenAndSetCtxFunc      // Custom token validation function // 自定义令牌验证函数
	fieldName      string                       // Request field name extracting auth token // 提取认证令牌的请求头字段名
	apmSpanName    string                       // APM span name, blank disables tracing // APM span 名称，为空时禁用追踪
	apmMatchSuffix string                       // APM match span suffix, default -match // APM match span 后缀，默认 -match
	debugMode      bool                         // Debug mode switch // 调试模式开关
}

// NewConfig creates a new simple auth config with route scope and token check function
// Defaults to Authorization field and current debug mode setting
//
// NewConfig 创建新的简单认证配置，需要路由范围和令牌检查函数
// 默认使用 Authorization 请求头和当前调试模式设置
func NewConfig(routeScope *authkratosroutes.RouteScope, checkToken CheckTokenAndSetCtxFunc) *Config {
	return &Config{
		routeScope:     routeScope,
		checkToken:     checkToken,
		fieldName:      "Authorization",
		apmSpanName:    "",
		apmMatchSuffix: "-match", // Default suffix // 默认后缀
		debugMode:      authkratos.GetDebugMode(),
	}
}

// WithFieldName sets request field name used in authentication
// Avoid non-standard names in configuration
// Nginx ignores names with underscores unless underscores_in_headers is on
// Recommend not using names with extra punctuation in development
//
// WithFieldName 设置请求头中用于认证的字段名
// 注意配置时不要配置非标准的字段名
// Nginx 默认忽略带有下划线的 headers 信息，除非配置 underscores_in_headers on
// 因此在开发中建议不要配置含特殊字符的字段名
func (c *Config) WithFieldName(fieldName string) *Config {
	c.fieldName = fieldName
	return c
}

// GetFieldName gets request field name used in authentication
//
// GetFieldName 获取请求头中用于认证的字段名
func (c *Config) GetFieldName() string {
	return c.fieldName
}

func (c *Config) WithDebugMode(debugMode bool) *Config {
	c.debugMode = debugMode
	return c
}

// WithDefaultApmSpanName sets default APM span name
// Default name: auth-kratos-simple
//
// WithDefaultApmSpanName 使用默认的 APM span 名称
// 默认名称: auth-kratos-simple
func (c *Config) WithDefaultApmSpanName() *Config {
	return c.WithApmSpanName("auth-kratos-simple")
}

// WithApmSpanName sets APM span name
// Empty value disables APM tracing
//
// WithApmSpanName 设置 APM span 名称
// 为空时不启动 APM 追踪
func (c *Config) WithApmSpanName(apmSpanName string) *Config {
	c.apmSpanName = must.Nice(apmSpanName)
	return c
}

// WithApmMatchSuffix sets APM match span suffix
// Default value is -match
//
// WithApmMatchSuffix 设置 APM match span 后缀
// 默认为 -match
func (c *Config) WithApmMatchSuffix(apmMatchSuffix string) *Config {
	c.apmMatchSuffix = must.Nice(apmMatchSuffix)
	return c
}

func NewMiddleware(cfg *Config, logger log.Logger) middleware.Middleware {
	slog := log.NewHelper(logger)
	slog.Infof(
		"auth-kratos-simple: new middleware field-name=%v side=%v operations=%d debug-mode=%v",
		cfg.fieldName,
		cfg.routeScope.Side,
		len(cfg.routeScope.OperationSet),
		utils.BooleanToNum(cfg.debugMode),
	)
	if cfg.debugMode {
		slog.Debugf("auth-kratos-simple: new middleware field-name=%v route-scope: %s", cfg.fieldName, neatjsons.S(cfg.routeScope))
	}
	return selector.Server(middlewareFunc(cfg, logger)).Match(matchFunc(cfg, logger)).Build()
}

func matchFunc(cfg *Config, logger log.Logger) selector.MatchFunc {
	slog := log.NewHelper(logger)

	return func(ctx context.Context, operation string) bool {
		// 如果配置了 APM span 名称，则启动 APM 追踪
		if cfg.apmSpanName != "" {
			apmTx := apm.TransactionFromContext(ctx)
			span := apmTx.StartSpan(cfg.apmSpanName+cfg.apmMatchSuffix, "app", nil)
			defer span.End()
		}

		match := cfg.routeScope.Match(operation)
		if cfg.debugMode {
			if match {
				slog.Debugf("auth-kratos-simple: operation=%s side=%v match=%d next -> check auth", operation, cfg.routeScope.Side, utils.BooleanToNum(match))
			} else {
				slog.Debugf("auth-kratos-simple: operation=%s side=%v match=%d skip -- check auth", operation, cfg.routeScope.Side, utils.BooleanToNum(match))
			}
		}
		return match
	}
}

func middlewareFunc(cfg *Config, logger log.Logger) middleware.Middleware {
	slog := log.NewHelper(logger)

	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if tsp, ok := transport.FromServerContext(ctx); ok {
				// 如果配置了 APM span 名称，则启动 APM 追踪
				if cfg.apmSpanName != "" {
					apmTx := apm.TransactionFromContext(ctx)
					span := apmTx.StartSpan(cfg.apmSpanName, "app", nil)
					defer span.End()
				}

				authToken := tsp.RequestHeader().Get(cfg.fieldName)
				if authToken == "" {
					if cfg.debugMode {
						slog.Debugf("auth-kratos-simple: auth-token is missing")
					}
					return nil, errors.Unauthorized("UNAUTHORIZED", "auth-kratos-simple: auth-token is missing")
				}
				// 调用用户自定义的认证函数
				// 认证成功时返回的 ctx 可能包含用户信息（如用户ID、角色等）
				ctx, erk := cfg.checkToken(ctx, authToken)
				if erk != nil {
					if cfg.debugMode {
						slog.Debugf("auth-kratos-simple: auth-token mismatch: %s", erk.Error())
					}
					return nil, erk
				}
				return handleFunc(ctx, req)
			}
			return nil, errors.Unauthorized("UNAUTHORIZED", "auth-kratos-simple: wrong context")
		}
	}
}
