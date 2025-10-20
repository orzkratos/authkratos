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

type Config struct {
	routeScope       *authkratosroutes.RouteScope
	authTokens       map[string]string
	fieldName        string
	apmSpanName      string // APM span 名称，为空时不启动 APM 追踪
	apmMatchSuffix   string // APM match span 后缀，默认为 -match
	debugMode        bool
	enableSimpleType bool // Enable simple token type // 启用简单令牌类型
	enableBearerType bool // Enable Bearer token type // 启用 Bearer 令牌类型
	enableBase64Type bool // Enable Base64 Basic Auth type // 启用 Base64 Basic Auth 类型
}

func NewConfig(
	routeScope *authkratosroutes.RouteScope,
	authTokens map[string]string,
) *Config {
	return &Config{
		// 注意配置时不要配置非标准的字段名
		// Nginx 默认忽略带有下划线的 headers 信息，除非配置 underscores_in_headers on
		// 因此在开发中建议不要配置含特殊字符的字段名
		routeScope:     routeScope,
		authTokens:     authTokens,
		fieldName:      "Authorization",
		apmSpanName:    "",
		apmMatchSuffix: "-match", // 默认后缀
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
// Default name: auth-kratos-tokens
//
// WithDefaultApmSpanName 使用默认的 APM span 名称
// 默认名称: auth-kratos-tokens
func (c *Config) WithDefaultApmSpanName() *Config {
	return c.WithApmSpanName("auth-kratos-tokens")
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

// WithEnableSimpleType enables simple token type authentication
// Token format: "secret-token-123"
//
// WithEnableSimpleType 启用简单令牌类型认证
// 令牌格式: "secret-token-123"
func (c *Config) WithEnableSimpleType() *Config {
	c.enableSimpleType = true
	return c
}

// WithEnableBearerType enables Bearer token type authentication
// Token format: "Bearer secret-token-123"
//
// WithEnableBearerType 启用 Bearer 令牌类型认证
// 令牌格式: "Bearer secret-token-123"
func (c *Config) WithEnableBearerType() *Config {
	c.enableBearerType = true
	return c
}

// WithEnableBase64Type enables Base64 Basic Auth type authentication
// Token format: "Basic base64(username:password)"
//
// WithEnableBase64Type 启用 Base64 Basic Auth 类型认证
// 令牌格式: "Basic base64(username:password)"
func (c *Config) WithEnableBase64Type() *Config {
	c.enableBase64Type = true
	return c
}

func (c *Config) GetAuthTokens() map[string]string {
	if c != nil {
		return c.authTokens
	}
	return nil
}

func (c *Config) CreateToken(username string) string {
	password, ok := c.GetAuthTokens()[username]
	must.TRUE(ok)
	must.Nice(password)
	return utils.BasicAuth(username, password)
}

func (c *Config) GetOneToken() string {
	return c.CreateToken(utils.Sample(maps.Keys(c.GetAuthTokens())))
}

func (c *Config) GetMapTokens() map[string]string {
	var res = make(map[string]string, len(c.GetAuthTokens()))
	for username, password := range c.GetAuthTokens() {
		res[username] = utils.BasicAuth(username, password)
	}
	return res
}

func NewMiddleware(cfg *Config, logger log.Logger) middleware.Middleware {
	slog := log.NewHelper(logger)
	slog.Infof(
		"auth-kratos-tokens: new middleware field-name=%v auth-tokens=%d side=%v operations=%d enable-simple=%v enable-bearer=%v enable-base64=%v",
		cfg.fieldName,
		len(cfg.authTokens),
		cfg.routeScope.Side,
		len(cfg.routeScope.OperationSet),
		utils.BooleanToNum(cfg.enableSimpleType),
		utils.BooleanToNum(cfg.enableBearerType),
		utils.BooleanToNum(cfg.enableBase64Type),
	)
	if cfg.debugMode {
		slog.Debugf("auth-kratos-tokens: new middleware field-name=%v route-scope: %s", cfg.fieldName, neatjsons.S(cfg.routeScope))
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
				slog.Debugf("auth-kratos-tokens: operation=%s side=%v match=%d next -> check auth", operation, cfg.routeScope.Side, utils.BooleanToNum(match))
			} else {
				slog.Debugf("auth-kratos-tokens: operation=%s side=%v match=%d skip -- check auth", operation, cfg.routeScope.Side, utils.BooleanToNum(match))
			}
		}
		return match
	}
}

func middlewareFunc(cfg *Config, logger log.Logger) middleware.Middleware {
	slog := log.NewHelper(logger)

	// Build token maps based on enabled types
	// Initialize blank maps as default
	//
	// 根据启用的类型构建令牌映射
	// 默认初始化为空 map 以确保安全
	mapBox := &authTokenMapBox{
		simpleTypeToUsername: make(map[string]string),
		bearerTypeToUsername: make(map[string]string),
		base64TypeToUsername: make(map[string]string),
	}
	if cfg.enableSimpleType {
		mapBox.simpleTypeToUsername = buildSimpleTokenToUsername(cfg.authTokens)
	}
	if cfg.enableBearerType {
		mapBox.bearerTypeToUsername = buildBearerTokenToUsername(cfg.authTokens)
	}
	if cfg.enableBase64Type {
		mapBox.base64TypeToUsername = buildBase64TokenToUsername(cfg.authTokens)
	}

	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if tsp, ok := transport.FromServerContext(ctx); ok {
				// 如果配置了 APM span 名称，则启动 APM 追踪
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
				// 认证成功，将用户名注入到 context 中
				// 后续业务可通过 GetUsername(ctx) 获取当前用户名
				ctx = SetUsernameIntoContext(ctx, username)
				return handleFunc(ctx, req)
			}
			return nil, errors.Unauthorized("UNAUTHORIZED", "auth-kratos-tokens: wrong context")
		}
	}
}

func checkAuthToken(cfg *Config, mapBox *authTokenMapBox, token string, slog *log.Helper) (string, *errors.Error) {
	if !cfg.enableSimpleType && !cfg.enableBearerType && !cfg.enableBase64Type {
		if cfg.debugMode {
			slog.Debugf("auth-kratos-tokens: check token (no token types enabled, must enable at least one)")
		}
		return "", errors.Unauthorized("UNAUTHORIZED", "auth-kratos-tokens: no token type enabled")
	}

	if username, ok := mapBox.simpleTypeToUsername[token]; ok {
		if cfg.debugMode {
			slog.Debugf("auth-kratos-tokens: simple-type request username:%v quick pass", username)
		}
		return username, nil
	}
	if username, ok := mapBox.bearerTypeToUsername[token]; ok {
		if cfg.debugMode {
			slog.Debugf("auth-kratos-tokens: bearer-type request username:%v quick pass", username)
		}
		return username, nil
	}
	if username, ok := mapBox.base64TypeToUsername[token]; ok {
		if cfg.debugMode {
			slog.Debugf("auth-kratos-tokens: base64-type request username:%v quick pass", username)
		}
		return username, nil
	}
	return "", errors.Unauthorized("UNAUTHORIZED", "auth-kratos-tokens: auth-token mismatch")
}

type authTokenMapBox struct {
	simpleTypeToUsername map[string]string
	bearerTypeToUsername map[string]string
	base64TypeToUsername map[string]string
}

func buildSimpleTokenToUsername(usernameToTokenMap map[string]string) map[string]string {
	simpleTypeToUsername := make(map[string]string, len(usernameToTokenMap))
	for username, token := range usernameToTokenMap {
		simpleTypeToUsername[token] = username
	}
	return simpleTypeToUsername
}

func buildBearerTokenToUsername(usernameToTokenMap map[string]string) map[string]string {
	bearerTypeToUsername := make(map[string]string, len(usernameToTokenMap))
	for username, token := range usernameToTokenMap {
		bearerTypeToUsername["Bearer "+token] = username
	}
	return bearerTypeToUsername
}

func buildBase64TokenToUsername(usernameToTokenMap map[string]string) map[string]string {
	base64TypeToUsername := make(map[string]string, len(usernameToTokenMap))
	for username, token := range usernameToTokenMap {
		encoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, token)))
		base64TypeToUsername["Basic "+encoded] = username
	}
	return base64TypeToUsername
}

type usernameKey struct{}

// SetUsernameIntoContext 将用户名注入到 context 中
// 认证成功后调用，用于在请求上下文中传递用户信息
func SetUsernameIntoContext(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey{}, username)
}

// GetUsernameFromContext 从 context 中获取用户名
// 返回：用户名和是否存在的标志
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(usernameKey{}).(string)
	return username, ok
}

// GetUsername 从 context 中获取用户名
// 这是 GetUsernameFromContext 的简化版本
func GetUsername(ctx context.Context) (string, bool) {
	return GetUsernameFromContext(ctx)
}
