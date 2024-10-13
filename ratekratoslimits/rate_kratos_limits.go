package utils_kratos_ratelimit

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-redis/redis_rate/v10"
	"github.com/orzkratos/authkratos/authkratosroutes"
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

func (a *Config) IsEnable() bool {
	if a != nil {
		return a.enable
	}
	return false
}

func NewMiddleware(cfg *Config, LOGGER log.Logger) middleware.Middleware {
	LOG := log.NewHelper(LOGGER)
	LOG.Infof(
		"new rate_limit middleware enable=%v rule=%v include=%v operations=%v",
		cfg.IsEnable(),
		cfg.rule.String(),
		cfg.selectPath.SelectSide,
		len(cfg.selectPath.Operations),
	)

	return selector.Server(middlewareFunc(cfg, LOGGER)).Match(matchFunc(cfg, LOGGER)).Build()
}

func matchFunc(cfg *Config, LOGGER log.Logger) selector.MatchFunc {
	LOG := log.NewHelper(LOGGER)

	return func(ctx context.Context, operation string) bool {
		if !cfg.IsEnable() {
			return false
		}
		match := cfg.selectPath.Match(operation)
		if match {
			LOG.Debugf("operation=%s include=%v match=%v must check rate", operation, cfg.selectPath.SelectSide, match)
		} else {
			LOG.Debugf("operation=%s include=%v match=%v skip check rate", operation, cfg.selectPath.SelectSide, match)
		}
		return match
	}
}

func middlewareFunc(cfg *Config, LOGGER log.Logger) middleware.Middleware {
	LOG := log.NewHelper(LOGGER)

	rateLimitRule := *cfg.rule

	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (resp interface{}, err error) {
			if !cfg.IsEnable() {
				LOG.Infof("rate_limit: cfg.enable=false anonymous pass")
				return handleFunc(ctx, req)
			}

			uck := cfg.parseUniqueCode(ctx)

			rls, err := cfg.rateLimitBottle.Allow(ctx, uck, rateLimitRule)
			if err != nil {
				return nil, erero.WithMessage(err, "rate_limit redis exception")
			}

			if rls.Allowed != 0 {
				LOG.Debugf("rate_limit allowed=%v remaining=%v so can pass", rls.Allowed, rls.Remaining)
			} else {
				LOG.Warnf("rate_limit exceeds so reject requests")

				return nil, ratelimit.ErrLimitExceed
			}
			return handleFunc(ctx, req)
		}
	}
}
