// Package fastkratoshandle: Selective timeout override middleware with route-based control
// Provides fast timeout settings with flexible route scope configuration
// Enables shortening timeout on specific routes while maintaining defaults on others
// Good fit in mixed workload scenarios needing distinct timeout tactics
//
// fastkratoshandle: 选择性超时覆盖中间件，支持基于路由的控制
// 提供快速超时设置和灵活的路由范围配置
// 可在特定路由上缩短超时时间，同时在其他地方保持默认值
// 适用于需要不同超时策略的混合工作负载场景
package fastkratoshandle

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/orzkratos/authkratos"
	"github.com/orzkratos/authkratos/authkratosroutes"
	"github.com/orzkratos/authkratos/internal/utils"
	"github.com/yyle88/must"
	"github.com/yyle88/neatjson/neatjsons"
	"go.elastic.co/apm/v2"
)

type Config struct {
	routeScope     *authkratosroutes.RouteScope
	newTimeout     time.Duration // 快速超时的时间
	apmSpanName    string        // APM span 名称，为空时不启动 APM 追踪
	apmMatchSuffix string        // APM match span 后缀，默认为 -match
	debugMode      bool
}

func NewConfig(routeScope *authkratosroutes.RouteScope, newTimeout time.Duration) *Config {
	return &Config{
		routeScope:     routeScope,
		newTimeout:     newTimeout,
		apmSpanName:    "",
		apmMatchSuffix: "-match", // 默认后缀
		debugMode:      authkratos.GetDebugMode(),
	}
}

func (c *Config) WithDebugMode(debugMode bool) *Config {
	c.debugMode = debugMode
	return c
}

// WithDefaultApmSpanName sets default APM span name
// Default name: fast-kratos-handle
//
// WithDefaultApmSpanName 使用默认的 APM span 名称
// 默认名称: fast-kratos-handle
func (c *Config) WithDefaultApmSpanName() *Config {
	return c.WithApmSpanName("fast-kratos-handle")
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

// NewMiddleware creates middleware with shorter timeout on specific routes
// In practice extending timeout is more common than shortening
// Since ctx timeout can just shorten not extend, use exclusion filtering approach:
// Set long timeout on entire service, then limit other routes with shorter timeouts
// Use EXCLUDE mode to exclude routes needing long timeout, others get fast timeout
// This satisfies the "extend timeout" requirement
//
// NewMiddleware 这个函数得到个middleware让某些接口具有更短的超时时间
// 但现实中我们遇到的问题往往是需要延长某个接口的超时时间
// 这样"设置更长超时时间"的需求更常见，以下是解决的思路
// 由于 ctx 的超时时间只能缩短而不能延长，因此整个设计是用"排除法过滤"
// 就是先给整个服务的接口配置很长的超时时间，再限制其余接口的超时时间为更短的时间
// 配置时使用 "EXCLUDE" 排除这些接口，其它的都是快速超时的
// 即可满足"设置更长超时时间"的需求
func NewMiddleware(cfg *Config, logger log.Logger) middleware.Middleware {
	slog := log.NewHelper(logger)
	slog.Infof(
		"fast-kratos-handle: new middleware side=%v operations=%d new-timeout=%v debug-mode=%v",
		cfg.routeScope.Side,
		len(cfg.routeScope.OperationSet),
		cfg.newTimeout,
		utils.BooleanToNum(cfg.debugMode),
	)
	if cfg.debugMode {
		slog.Debugf("fast-kratos-handle: new middleware route-scope: %s", neatjsons.S(cfg.routeScope))
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
				slog.Debugf("fast-kratos-handle: operation=%s side=%v match=%d next -> fast-handle", operation, cfg.routeScope.Side, utils.BooleanToNum(match))
			} else {
				slog.Debugf("fast-kratos-handle: operation=%s side=%v match=%d skip -- slow-handle", operation, cfg.routeScope.Side, utils.BooleanToNum(match))
			}
		}
		return match
	}
}

func middlewareFunc(cfg *Config, logger log.Logger) middleware.Middleware {
	slog := log.NewHelper(logger)

	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 如果配置了 APM span 名称，则启动 APM 追踪
			if cfg.apmSpanName != "" {
				apmTx := apm.TransactionFromContext(ctx)
				span := apmTx.StartSpan(cfg.apmSpanName, "app", nil)
				defer span.End()
			}

			// 设置新超时时间，由于 ctx 是所有超时时间里取最短的
			// 因此只能缩短而不能延长，因此需要选择快速超时的
			ctx, can := context.WithTimeout(ctx, cfg.newTimeout)
			defer can()
			if cfg.debugMode {
				slog.Debugf("fast-kratos-handle: context with new-timeout=%v fast-handle", cfg.newTimeout)
			}
			return handleFunc(ctx, req)
		}
	}
}
