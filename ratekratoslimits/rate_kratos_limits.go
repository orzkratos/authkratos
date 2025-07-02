package utils_kratos_ratelimit

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
	"github.com/yyle88/neatjson/neatjsons"
)

type Config struct {
	redisCache *redis_rate.Limiter
	redisLimit *redis_rate.Limit
	selectPath *authkratosroutes.SelectPath
	keyFromCtx func(ctx context.Context) (string, bool)
	debugMode  bool
}

func NewConfig(
	redisCache *redis_rate.Limiter,
	redisLimit *redis_rate.Limit,
	selectPath *authkratosroutes.SelectPath,
	keyFromCtx func(ctx context.Context) (string, bool),
) *Config {
	return &Config{
		redisCache: redisCache,
		redisLimit: redisLimit,
		selectPath: selectPath,
		keyFromCtx: keyFromCtx,
		debugMode:  authkratos.GetDebugMode(),
	}
}

func (c *Config) WithDebugMode(debugMode bool) *Config {
	c.debugMode = debugMode
	return c
}

func NewMiddleware(cfg *Config, logger log.Logger) middleware.Middleware {
	LOG := log.NewHelper(logger)
	LOG.Infof(
		"rate-kratos-limits: new middleware include=%s operations=%d rate=%v debug-mode=%v",
		cfg.selectPath.SelectSide,
		len(cfg.selectPath.Operations),
		cfg.redisLimit.String(),
		cfg.debugMode,
	)
	if cfg.debugMode {
		LOG.Debugf("rate-kratos-limits: new middleware select-path: %s", neatjsons.S(cfg.selectPath))
	}
	return selector.Server(middlewareFunc(cfg, logger)).Match(matchFunc(cfg, logger)).Build()
}

func matchFunc(cfg *Config, logger log.Logger) selector.MatchFunc {
	LOG := log.NewHelper(logger)

	return func(ctx context.Context, operation string) bool {
		match := cfg.selectPath.Match(operation)
		if cfg.debugMode {
			if match {
				LOG.Debugf("rate-kratos-limits: operation=%s include=%v match=%d next -> check-rate-limit", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match))
			} else {
				LOG.Debugf("rate-kratos-limits: operation=%s include=%v match=%d skip -- check-rate-limit", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match))
			}
		}
		return match
	}
}

func middlewareFunc(cfg *Config, logger log.Logger) middleware.Middleware {
	LOG := log.NewHelper(logger)

	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (resp interface{}, err error) {
			// 这里就是从上下文中获取唯一键，通常是用户的 PK UK ID 或者 IP 地址等信息
			uniqueKey, ok := cfg.keyFromCtx(ctx)
			if !ok {
				if cfg.debugMode {
					LOG.Debugf("rate-kratos-limits: reject requests key=unknown missing unique key from context")
				}
				return nil, ratelimit.ErrLimitExceed
			}

			if uniqueKey == "" {
				if cfg.debugMode {
					LOG.Debugf("rate-kratos-limits: reject requests key=nothing missing unique key from context")
				}
				return nil, ratelimit.ErrLimitExceed
			}

			// 这块底层包在设计时有 AllowN 的设计，这使得该函数的返回值，还得转换转换 res.Allowed > 0 时才算是通过
			res, err := cfg.redisCache.Allow(ctx, uniqueKey, *cfg.redisLimit)
			if err != nil {
				if cfg.debugMode {
					LOG.Debugf("rate-kratos-limits: redis is unavailable key=%s err=%v reject requests", uniqueKey, err)
				}
				return nil, errors.ServiceUnavailable("unavailable", "rate-kratos-limits: redis is unavailable").WithCause(err)
			}
			// 当然在这种场景里 res.Allowed 的返回值只能是0或1两个值，但在写逻辑时把范围放宽些，避免底层不按预期返回
			if res.Allowed <= 0 {
				if cfg.debugMode {
					LOG.Debugf("rate-kratos-limits: reject requests key=%s allowed=%v remaining=%v", uniqueKey, res.Allowed, res.Remaining)
				}
				return nil, ratelimit.ErrLimitExceed
			}
			if cfg.debugMode {
				LOG.Debugf("rate-kratos-limits: accept requests key=%s allowed=%v remaining=%v", uniqueKey, res.Allowed, res.Remaining)
			}
			return handleFunc(ctx, req)
		}
	}
}
