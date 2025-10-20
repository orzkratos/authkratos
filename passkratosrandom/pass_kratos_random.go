// Package passkratosrandom: Probabilistic request blocking middleware with chaos testing support
// Provides random pass-through rate with configurable odds
// Good fit in testing, chaos engineering, and staged rollout scenarios
// Enables blocking specific request ratio within designated route scope
//
// passkratosrandom: 概率性请求阻断中间件，支持混沌测试
// 提供可配置概率的随机放行率控制
// 适用于压力测试、混沌工程和灰度发布场景
// 可在指定路由范围内阻断一定百分比的请求
package passkratosrandom

import (
	"context"
	"math/rand"

	"github.com/go-kratos/kratos/v2/errors"
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
	rate           float64
	apmSpanName    string // APM span 名称，为空时不启动 APM 追踪
	apmMatchSuffix string // APM match span 后缀，默认为 -match
	debugMode      bool
}

func NewConfig(routeScope *authkratosroutes.RouteScope, passRate float64) *Config {
	return &Config{
		routeScope:     routeScope,
		rate:           passRate,
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
// Default name: pass-kratos-random
//
// WithDefaultApmSpanName 使用默认的 APM span 名称
// 默认名称: pass-kratos-random
func (c *Config) WithDefaultApmSpanName() *Config {
	return c.WithApmSpanName("pass-kratos-random")
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

// NewMiddleware creates middleware that fails requests with configured rate
//
// NewMiddleware 让接口有一定概率失败
func NewMiddleware(cfg *Config, logger log.Logger) middleware.Middleware {
	slog := log.NewHelper(logger)
	slog.Infof(
		"pass-kratos-random: new middleware side=%v operations=%d rate=%v debug-mode=%v",
		cfg.routeScope.Side,
		len(cfg.routeScope.OperationSet),
		cfg.rate,
		utils.BooleanToNum(cfg.debugMode),
	)
	if cfg.debugMode {
		slog.Debugf("pass-kratos-random: new middleware route-scope: %s", neatjsons.S(cfg.routeScope))
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

		if match := cfg.routeScope.Match(operation); !match {
			if cfg.debugMode {
				slog.Debugf("pass-kratos-random: operation=%s side=%v match=%d next -> skip random", operation, cfg.routeScope.Side, utils.BooleanToNum(match))
			}
			return false
		}
		// 设置rate=0.6就是有60%的概率通过
		// 设置rate=1或者>1就是肯定通过，设置为0或负数就必然不通过
		ratePass := rand.Float64() < cfg.rate

		// 是否进入拦截器，拦截器会拦截请求
		// 因此这里求逆值，通过的不拦截，不通过的拦截
		match := !ratePass
		if cfg.debugMode {
			if match {
				slog.Debugf("pass-kratos-random: operation=%s side=%v match=%d next -> goto unavailable", operation, cfg.routeScope.Side, utils.BooleanToNum(match))
			} else {
				slog.Debugf("pass-kratos-random: operation=%s side=%v match=%d skip -- skip unavailable", operation, cfg.routeScope.Side, utils.BooleanToNum(match))
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

			if cfg.debugMode {
				slog.Debugf("pass-kratos-random: random match unavailable")
			}
			//当已经命中概率的时候，就直接返回错误
			return nil, errors.ServiceUnavailable("RANDOM_RATE_UNAVAILABLE", "pass-kratos-random: random unavailable")
		}
	}
}
