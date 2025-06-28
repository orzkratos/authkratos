package utils_kratos_ratelimit

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-redis/redis_rate/v10"
	"github.com/orzkratos/authkratos/authkratosroutes"
	"github.com/orzkratos/authkratos/internal/utils"
	"github.com/yyle88/erero"
)

type Config struct {
	rateLimitBottle *redis_rate.Limiter
	rule            *redis_rate.Limit
	parseUniqueCode func(ctx context.Context) string
	selectPath      *authkratosroutes.SelectPath
	enable          bool
}

func NewConfig(
	rateLimitBottle *redis_rate.Limiter,
	rule *redis_rate.Limit,
	parseUniqueCode func(ctx context.Context) string,
	selectPath *authkratosroutes.SelectPath,
) *Config {
	return &Config{
		rateLimitBottle: rateLimitBottle,
		rule:            rule,
		parseUniqueCode: parseUniqueCode,
		selectPath:      selectPath,
		enable:          true,
	}
}

func (a *Config) SetEnable(enable bool) {
	a.enable = enable
}

func (a *Config) GetEnable() bool {
	if a != nil {
		return a.enable
	}
	return false
}

func NewMiddleware(cfg *Config, LOGGER log.Logger) middleware.Middleware {
	LOG := log.NewHelper(LOGGER)
	LOG.Infof(
		"new rate_limit middleware enable=%v rule=%v include=%v operations=%v",
		cfg.GetEnable(),
		cfg.rule.String(),
		cfg.selectPath.SelectSide,
		len(cfg.selectPath.Operations),
	)

	return selector.Server(middlewareFunc(cfg, LOGGER)).Match(matchFunc(cfg, LOGGER)).Build()
}

func matchFunc(cfg *Config, LOGGER log.Logger) selector.MatchFunc {
	LOG := log.NewHelper(LOGGER)

	return func(ctx context.Context, operation string) bool {
		if !cfg.GetEnable() {
			return false
		}
		match := cfg.selectPath.Match(operation)
		if match {
			LOG.Debugf("operation=%s include=%v match=%d next -> check rate", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match))
		} else {
			LOG.Debugf("operation=%s include=%v match=%d skip -- check rate", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match))
		}
		return match
	}
}

func middlewareFunc(cfg *Config, LOGGER log.Logger) middleware.Middleware {
	LOG := log.NewHelper(LOGGER)

	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (resp interface{}, err error) {
			if !cfg.GetEnable() {
				LOG.Infof("rate_limit: cfg.enable=false anonymous pass")
				return handleFunc(ctx, req)
			}

			uck := cfg.parseUniqueCode(ctx)

			allowResult, err := cfg.rateLimitBottle.Allow(ctx, uck, *cfg.rule)
			if err != nil {
				return nil, erero.WithMessage(err, "rate_limit redis exception")
			}

			if allowResult.Allowed != 0 {
				LOG.Debugf("rate_limit allowed=%v remaining=%v so can pass", allowResult.Allowed, allowResult.Remaining)
			} else {
				LOG.Warnf("rate_limit exceeds so reject requests")

				return nil, ratelimit.ErrLimitExceed
			}
			return handleFunc(ctx, req)
		}
	}
}
