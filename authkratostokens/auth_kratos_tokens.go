// Package authkratostokens: Pre-configured token-based authentication middleware
// Provides out-of-box auth with username-token map and various token format support
// Supports simple tokens, authorization tokens, and Base64-encoded Basic Auth
// Auto-injects authenticated username into request context
//
// authkratostokens: 预配置的基于令牌的认证中间件
// 提供开箱即用的认证功能，支持用户名-令牌映射和多种令牌格式
// 支持简单令牌、Bearer 令牌和 Base64 编码的 Basic Auth
// 自动将已认证的用户名注入请求上下文
package authkratostokens

import (
	"context"
	"encoding/base64"
	"fmt"

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
	"golang.org/x/exp/maps"
)

// Config holds authentication middleware configuration
// Includes route scope, token map, and enabled token types
//
// Config 认证中间件配置
// 包含路由范围、令牌映射和启用的令牌类型
type Config struct {
	routeScope     *authkratosroutes.RouteScope // Route scope config // 路由范围配置
	authTokens     map[string]string            // Username to token map // 用户名到令牌的映射
	fieldName      string                       // Request field name // 请求头字段名
	apmSpanName    string                       // APM span name, blank disables APM // APM span 名称，空值禁用 APM
	apmMatchSuffix string                       // APM match span suffix // APM match span 后缀
	debugMode      bool                         // Debug mode flag // 调试模式标志
	simpleEnable   bool                         // Enable simple token type // 启用简单令牌类型
	bearerEnable   bool                         // Enable Bearer token type // 启用 Bearer 令牌类型
	base64Enable   bool                         // Enable Base64 Basic Auth type // 启用 Base64 Basic Auth 类型
}

// NewConfig creates a new Config with route scope and token map
// Default: field name "Authorization", APM disabled, debug mode from package setting
//
// NewConfig 创建新的配置，包含路由范围和令牌映射
// 默认字段名 "Authorization"，APM 禁用，调试模式从全局设置获取
func NewConfig(
	routeScope *authkratosroutes.RouteScope,
	authTokens map[string]string,
) *Config {
	return &Config{
		// Avoid non-standard field names
		// Nginx ignores names with underscores unless underscores_in_headers is on
		// Recommend avoiding names with non-standard chars
		//
		// 避免非标准字段名
		// Nginx 默认忽略带下划线的 headers，除非配置 underscores_in_headers on
		// 建议不用含特殊字符的字段名
		routeScope:     routeScope,
		authTokens:     authTokens,
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

// WithDebugMode sets debug mode flag
// When enabled, outputs detailed auth logs
//
// WithDebugMode 设置调试模式标志
// 启用时输出详细的认证日志
func (c *Config) WithDebugMode(debugMode bool) *Config {
	c.debugMode = debugMode
	return c
}

// WithDefaultApmSpanName sets default APM span name
// Default name: auth-kratos-tokens
//
// WithDefaultApmSpanName 使用默认的 APM span 名称
// 默认名称: auth-kratos-tokens
func (c *Config) WithDefaultApmSpanName() *Config {
	return c.WithApmSpanName("auth-kratos-tokens")
}

// WithApmSpanName sets APM span name
// Blank value disables APM tracing
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

// WithSimpleEnable enables simple token type authentication
// Token format: "secret-token-123"
//
// WithSimpleEnable 启用简单令牌类型认证
// 令牌格式: "secret-token-123"
func (c *Config) WithSimpleEnable() *Config {
	c.simpleEnable = true
	return c
}

// WithBearerEnable enables Bearer token type authentication
// Token format: "Bearer secret-token-123"
//
// WithBearerEnable 启用 Bearer 令牌类型认证
// 令牌格式: "Bearer secret-token-123"
func (c *Config) WithBearerEnable() *Config {
	c.bearerEnable = true
	return c
}

// WithBase64Enable enables Base64 Basic Auth type authentication
// Token format: "Basic base64(username:password)"
//
// WithBase64Enable 启用 Base64 Basic Auth 类型认证
// 令牌格式: "Basic base64(username:password)"
func (c *Config) WithBase64Enable() *Config {
	c.base64Enable = true
	return c
}

// GetSimpleTokens returns username to token map
// Returns nil if Config is nil
//
// GetSimpleTokens 返回用户名到令牌的映射
// Config 为 nil 时返回 nil
func (c *Config) GetSimpleTokens() map[string]string {
	if c != nil {
		return c.authTokens
	}
	return nil
}

// GetBase64Tokens returns username to Basic Auth token map
//
// GetBase64Tokens 返回用户名到 Basic Auth 令牌的映射
func (c *Config) GetBase64Tokens() map[string]string {
	var res = make(map[string]string, len(c.GetSimpleTokens()))
	for username, password := range c.GetSimpleTokens() {
		res[username] = utils.BasicAuth(username, password)
	}
	return res
}

// LookupBase64Token picks Basic Auth token by username
// Panics if username not found in token map
//
// LookupBase64Token 根据用户名挑选 Basic Auth 令牌
// 用户名不存在时 panic
func (c *Config) LookupBase64Token(username string) string {
	password, ok := c.GetSimpleTokens()[username]
	must.TRUE(ok)
	must.Nice(password)
	return utils.BasicAuth(username, password)
}

// RandomBase64Token returns a random Basic Auth token from token map
//
// RandomBase64Token 从令牌映射中随机返回一个 Basic Auth 令牌
func (c *Config) RandomBase64Token() string {
	return c.LookupBase64Token(utils.Sample(maps.Keys(c.GetSimpleTokens())))
}

// NewMiddleware creates authentication middleware with config and logger
// Uses selector.Server to match routes and check auth tokens
//
// NewMiddleware 使用配置和日志创建认证中间件
// 使用 selector.Server 匹配路由并检查认证令牌
func NewMiddleware(cfg *Config, logger log.Logger) middleware.Middleware {
	slog := log.NewHelper(logger)
	slog.Infof(
		"auth-kratos-tokens: new middleware field-name=%v auth-tokens=%d side=%v operations=%d simple-enable=%v bearer-enable=%v base64-enable=%v",
		cfg.fieldName,
		len(cfg.authTokens),
		cfg.routeScope.Side,
		len(cfg.routeScope.OperationSet),
		utils.BooleanToNum(cfg.simpleEnable),
		utils.BooleanToNum(cfg.bearerEnable),
		utils.BooleanToNum(cfg.base64Enable),
	)
	if cfg.debugMode {
		slog.Debugf("auth-kratos-tokens: new middleware field-name=%v route-scope: %s", cfg.fieldName, neatjsons.S(cfg.routeScope))
	}
	return selector.Server(middlewareFunc(cfg, logger)).Match(matchFunc(cfg, logger)).Build()
}

// matchFunc creates route matching function
// Returns true if operation should be authenticated
//
// matchFunc 创建路由匹配函数
// 操作需要认证时返回 true
func matchFunc(cfg *Config, logger log.Logger) selector.MatchFunc {
	slog := log.NewHelper(logger)

	return func(ctx context.Context, operation string) bool {
		// Start APM span if APM span name is configured
		// 配置了 APM span 名称时启动 APM 追踪
		if cfg.apmSpanName != "" {
			apmTx := apm.TransactionFromContext(ctx)
			span := apmTx.StartSpan(cfg.apmSpanName+cfg.apmMatchSuffix, "app", nil)
			defer span.End()
		}

		match := cfg.routeScope.Match(operation)
		if cfg.debugMode {
			if match {
				slog.Debugf("auth-kratos-tokens: operation=%s side=%v match=%d next -> check auth", operation, cfg.routeScope.Side, utils.BooleanToNum(match))
			} else {
				slog.Debugf("auth-kratos-tokens: operation=%s side=%v match=%d skip -- check auth", operation, cfg.routeScope.Side, utils.BooleanToNum(match))
			}
		}
		return match
	}
}

// middlewareFunc creates authentication middleware implementation
// Validates tokens and injects username into context
//
// middlewareFunc 创建实际的认证中间件
// 验证令牌并将用户名注入 context
func middlewareFunc(cfg *Config, logger log.Logger) middleware.Middleware {
	slog := log.NewHelper(logger)

	// Build token maps based on enabled types, init blank maps as default
	// 根据启用的类型构建令牌映射，默认初始化空 map
	mapBox := &authTokenMapBox{
		simpleTokenMap: make(map[string]string),
		bearerTokenMap: make(map[string]string),
		base64TokenMap: make(map[string]string),
	}
	if cfg.simpleEnable {
		mapBox.simpleTokenMap = newSimpleTokenMap(cfg.authTokens)
	}
	if cfg.bearerEnable {
		mapBox.bearerTokenMap = newBearerTokenMap(cfg.authTokens)
	}
	if cfg.base64Enable {
		mapBox.base64TokenMap = newBase64TokenMap(cfg.authTokens)
	}

	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if tsp, ok := transport.FromServerContext(ctx); ok {
				// Start APM span if APM span name is configured
				// 配置了 APM span 名称时启动 APM 追踪
				if cfg.apmSpanName != "" {
					apmTx := apm.TransactionFromContext(ctx)
					span := apmTx.StartSpan(cfg.apmSpanName, "app", nil)
					defer span.End()
				}

				var authToken = tsp.RequestHeader().Get(cfg.fieldName)
				if authToken == "" {
					if cfg.debugMode {
						slog.Debugf("auth-kratos-tokens: auth-token is missing")
					}
					return nil, errors.Unauthorized("UNAUTHORIZED", "auth-kratos-tokens: auth-token is missing")
				}
				username, erk := checkAuthToken(cfg, mapBox, authToken, slog)
				if erk != nil {
					if cfg.debugMode {
						slog.Debugf("auth-kratos-tokens: auth-token mismatch: %s", erk.Error())
					}
					return nil, erk
				}
				// Auth success, inject username into context
				// Business code can get username via GetUsername(ctx)
				//
				// 认证成功，将用户名注入 context
				// 业务代码可通过 GetUsername(ctx) 获取用户名
				ctx = SetUsername(ctx, username)
				return handleFunc(ctx, req)
			}
			return nil, errors.Unauthorized("UNAUTHORIZED", "auth-kratos-tokens: wrong context")
		}
	}
}

// checkAuthToken validates token against enabled token maps
// Returns username on success, error on failure
//
// checkAuthToken 根据启用的令牌映射验证令牌
// 成功返回用户名，失败返回错误
func checkAuthToken(cfg *Config, mapBox *authTokenMapBox, token string, slog *log.Helper) (string, *errors.Error) {
	if !cfg.simpleEnable && !cfg.bearerEnable && !cfg.base64Enable {
		if cfg.debugMode {
			slog.Debugf("auth-kratos-tokens: check token (no token types enabled, must enable at least one)")
		}
		return "", errors.Unauthorized("UNAUTHORIZED", "auth-kratos-tokens: no token type enabled")
	}

	if username, ok := mapBox.simpleTokenMap[token]; ok {
		if cfg.debugMode {
			slog.Debugf("auth-kratos-tokens: simple-type request username:%v quick pass", username)
		}
		return username, nil
	}
	if username, ok := mapBox.bearerTokenMap[token]; ok {
		if cfg.debugMode {
			slog.Debugf("auth-kratos-tokens: bearer-type request username:%v quick pass", username)
		}
		return username, nil
	}
	if username, ok := mapBox.base64TokenMap[token]; ok {
		if cfg.debugMode {
			slog.Debugf("auth-kratos-tokens: base64-type request username:%v quick pass", username)
		}
		return username, nil
	}
	return "", errors.Unauthorized("UNAUTHORIZED", "auth-kratos-tokens: auth-token mismatch")
}

// authTokenMapBox holds pre-built token to username maps
// Each map type is built based on enabled token types
//
// authTokenMapBox 保存预构建的令牌到用户名映射
// 每种映射根据启用的令牌类型构建
type authTokenMapBox struct {
	simpleTokenMap map[string]string // Token -> username // 令牌 -> 用户名
	bearerTokenMap map[string]string // "Bearer token" -> username // "Bearer 令牌" -> 用户名
	base64TokenMap map[string]string // "Basic base64" -> username // "Basic base64" -> 用户名
}

// newSimpleTokenMap builds simple token to username map
// Maps raw token to username
//
// newSimpleTokenMap 构建简单令牌到用户名的映射
// 将原始令牌映射到用户名
func newSimpleTokenMap(usernameToTokenMap map[string]string) map[string]string {
	simpleTypeToUsername := make(map[string]string, len(usernameToTokenMap))
	for username, token := range usernameToTokenMap {
		simpleTypeToUsername[token] = username
	}
	return simpleTypeToUsername
}

// newBearerTokenMap builds Bearer token to username map
// Maps "Bearer {token}" to username
//
// newBearerTokenMap 构建 Bearer 令牌到用户名的映射
// 将 "Bearer {令牌}" 映射到用户名
func newBearerTokenMap(usernameToTokenMap map[string]string) map[string]string {
	bearerTypeToUsername := make(map[string]string, len(usernameToTokenMap))
	for username, token := range usernameToTokenMap {
		bearerTypeToUsername["Bearer "+token] = username
	}
	return bearerTypeToUsername
}

// newBase64TokenMap builds Base64 Basic Auth token to username map
// Maps "Basic {base64(username:password)}" to username
//
// newBase64TokenMap 构建 Base64 Basic Auth 令牌到用户名的映射
// 将 "Basic {base64(用户名:密码)}" 映射到用户名
func newBase64TokenMap(usernameToTokenMap map[string]string) map[string]string {
	base64TypeToUsername := make(map[string]string, len(usernameToTokenMap))
	for username, token := range usernameToTokenMap {
		encoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, token)))
		base64TypeToUsername["Basic "+encoded] = username
	}
	return base64TypeToUsername
}

// usernameKey is context key type used to store username
//
// usernameKey 是用于存储用户名的 context key 类型
type usernameKey struct{}

// SetUsernameIntoContext injects username into context
// Use on auth success to pass username in request context
//
// SetUsernameIntoContext 将用户名注入 context
// 认证成功后调用，在请求上下文中传递用户信息
func SetUsernameIntoContext(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey{}, username)
}

// GetUsernameFromContext gets username from context
// Returns username and existence flag
//
// GetUsernameFromContext 从 context 获取用户名
// 返回用户名和是否存在的标志
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(usernameKey{}).(string)
	return username, ok
}

// SetUsername is alias to SetUsernameIntoContext
//
// SetUsername 是 SetUsernameIntoContext 的别名
func SetUsername(ctx context.Context, username string) context.Context {
	return SetUsernameIntoContext(ctx, username)
}

// GetUsername is alias to GetUsernameFromContext
//
// GetUsername 是 GetUsernameFromContext 的别名
func GetUsername(ctx context.Context) (string, bool) {
	return GetUsernameFromContext(ctx)
}
