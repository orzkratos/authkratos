// Package ratekratoslimits: Redis-backed distributed rate limiting middleware
// Provides production-grade rate limiting with Redis persistence and context-based ID extraction
// Supports flexible rate limit configurations with distinct throttling options
// Integrates with route scope filtering and APM tracing
//
// ratekratoslimits: 基于 Redis 的分布式速率限制中间件
// 提供生产级别的速率限制，支持 Redis 持久化和基于上下文的键提取
// 支持灵活的速率限制配置，可实现按用户/按 IP 的限流能力
// 集成路由范围过滤和 APM 追踪
package ratekratoslimits

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-redis/redis_rate/v10"
	"github.com/orzkratos/authkratos"
	"github.com/orzkratos/authkratos/authkratosroutes"
	"github.com/orzkratos/authkratos/internal/utils"
	"github.com/yyle88/must"
	"github.com/yyle88/neatjson/neatjsons"
	"go.elastic.co/apm/v2"
)

type Config struct {
	routeScope     *authkratosroutes.RouteScope
	redisCache     *redis_rate.Limiter
	redisLimit     *redis_rate.Limit
	keyFromCtx     func(ctx context.Context) (string, bool)
	apmSpanName    string // APM span 名称，为空时不启动 APM 追踪
	apmMatchSuffix string // APM match span 后缀，默认为 -match
	debugMode      bool
}

func NewConfig(
	routeScope *authkratosroutes.RouteScope,
	redisCache *redis_rate.Limiter,
	redisLimit *redis_rate.Limit,
	keyFromCtx func(ctx context.Context) (string, bool),
) *Config {
	return &Config{
		routeScope:     routeScope,
		redisCache:     redisCache,
		redisLimit:     redisLimit,
		keyFromCtx:     keyFromCtx,
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
// Default name: rate-kratos-limits
//
// WithDefaultApmSpanName 使用默认的 APM span 名称
// 默认名称: rate-kratos-limits
func (c *Config) WithDefaultApmSpanName() *Config {
	return c.WithApmSpanName("rate-kratos-limits")
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

func NewMiddleware(cfg *Config, logger log.Logger) middleware.Middleware {
	slog := log.NewHelper(logger)
	slog.Infof(
		"rate-kratos-limits: new middleware side=%v operations=%d rate=%v debug-mode=%v",
		cfg.routeScope.Side,
		len(cfg.routeScope.OperationSet),
		cfg.redisLimit.String(),
		utils.BooleanToNum(cfg.debugMode),
	)
	if cfg.debugMode {
		slog.Debugf("rate-kratos-limits: new middleware route-scope: %s", neatjsons.S(cfg.routeScope))
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
				slog.Debugf("rate-kratos-limits: operation=%s side=%v match=%d next -> check-rate-limit", operation, cfg.routeScope.Side, utils.BooleanToNum(match))
			} else {
				slog.Debugf("rate-kratos-limits: operation=%s side=%v match=%d skip -- check-rate-limit", operation, cfg.routeScope.Side, utils.BooleanToNum(match))
			}
		}
		return match
	}
}

func middlewareFunc(cfg *Config, logger log.Logger) middleware.Middleware {
	slog := log.NewHelper(logger)

	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (resp interface{}, err error) {
			// 如果配置了 APM span 名称，则启动 APM 追踪
			if cfg.apmSpanName != "" {
				apmTx := apm.TransactionFromContext(ctx)
				span := apmTx.StartSpan(cfg.apmSpanName, "app", nil)
				defer span.End()
			}

			// 这里就是从上下文中获取唯一键
			// 通常是用户的 PK UK ID 或者 IP 地址等信息
			uniqueKey, ok := cfg.keyFromCtx(ctx)
			if !ok {
				if cfg.debugMode {
					slog.Debugf("rate-kratos-limits: reject requests key=unknown missing unique key from context")
				}
				return nil, ratelimit.ErrLimitExceed
			}

			if uniqueKey == "" {
				if cfg.debugMode {
					slog.Debugf("rate-kratos-limits: reject requests key=nothing missing unique key from context")
				}
				return nil, ratelimit.ErrLimitExceed
			}

			// 这块底层包在设计时有 AllowN 的设计
			// 这使得该函数的返回值，还得转换转换 res.Allowed > 0 时才算是通过
			res, err := cfg.redisCache.Allow(ctx, uniqueKey, *cfg.redisLimit)
			if err != nil {
				if cfg.debugMode {
					slog.Debugf("rate-kratos-limits: redis is unavailable key=%s err=%v reject requests", uniqueKey, err)
				}
				return nil, errors.ServiceUnavailable("unavailable", "rate-kratos-limits: redis is unavailable").WithCause(err)
			}
			// 当然在这种场景里 res.Allowed 的返回值只能是0或1两个值
			// 但在写逻辑时把范围放宽些，避免底层不按预期返回
			if res.Allowed <= 0 {
				if cfg.debugMode {
					slog.Debugf("rate-kratos-limits: reject requests key=%s allowed=%v remaining=%v", uniqueKey, res.Allowed, res.Remaining)
				}
				return nil, ratelimit.ErrLimitExceed
			}
			if cfg.debugMode {
				slog.Debugf("rate-kratos-limits: accept requests key=%s allowed=%v remaining=%v", uniqueKey, res.Allowed, res.Remaining)
			}
			return handleFunc(ctx, req)
		}
	}
}
