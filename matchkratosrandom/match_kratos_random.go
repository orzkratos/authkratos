// Package matchkratosrandom: Probabilistic request sampling match function with configurable rate
// Provides random match control based on configured odds with the selection pattern
// Returns match function that selects requests at random to pass through to wrapped middleware
// Good fit in sampling, staged rollout, and A/B testing scenarios
//
// matchkratosrandom: 概率性请求采样匹配函数，支持可配置的匹配率
// 基于配置的概率提供随机匹配控制，用于 selector 模式
// 返回随机选择请求通过到包裹中间件的匹配函数
// 适用于采样、灰度发布和 A/B 测试场景
package matchkratosrandom

import (
	"context"
	"math/rand"

	"github.com/go-kratos/kratos/v2/log"
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
	matchRate      float64
	apmSpanName    string // APM span 名称，为空时不启动 APM 追踪
	apmMatchSuffix string // APM match span 后缀，默认为 -match
	debugMode      bool
}

func NewConfig(routeScope *authkratosroutes.RouteScope, matchRate float64) *Config {
	return &Config{
		routeScope:     routeScope,
		matchRate:      matchRate,
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
// Default name: match-kratos-random
//
// WithDefaultApmSpanName 使用默认的 APM span 名称
// 默认名称: match-kratos-random
func (c *Config) WithDefaultApmSpanName() *Config {
	return c.WithApmSpanName("match-kratos-random")
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

func NewMatchFunc(cfg *Config, logger log.Logger) selector.MatchFunc {
	slog := log.NewHelper(logger)
	slog.Infof("match-kratos-random: new match func side=%v operations=%d match-rate=%v", cfg.routeScope.Side, len(cfg.routeScope.OperationSet), cfg.matchRate)
	if cfg.debugMode {
		slog.Debugf("match-kratos-random: new match func route-scope: %s", neatjsons.S(cfg.routeScope))
	}

	return func(ctx context.Context, operation string) bool {
		// 如果配置了 APM span 名称，则启动 APM 追踪
		if cfg.apmSpanName != "" {
			apmTx := apm.TransactionFromContext(ctx)
			span := apmTx.StartSpan(cfg.apmSpanName+cfg.apmMatchSuffix, "app", nil)
			defer span.End()
		}

		if match := cfg.routeScope.Match(operation); !match {
			if cfg.debugMode {
				slog.Debugf("match-kratos-random: operation=%s side=%v match=%d next -> skip random", operation, cfg.routeScope.Side, utils.BooleanToNum(match))
			}
			return false
		}

		// matchRate=0.6 means 60% requests will match (return true)
		// matchRate=1 or >1 means always match, matchRate=0 or <0 means never match
		//
		// matchRate=0.6 表示 60% 的请求会匹配（返回 true）
		// matchRate=1 或 >1 表示总是匹配，matchRate=0 或 <0 表示永不匹配
		match := rand.Float64() < cfg.matchRate

		if cfg.debugMode {
			if match {
				slog.Debugf("match-kratos-random: operation=%s side=%v match=%d next -> random matched", operation, cfg.routeScope.Side, utils.BooleanToNum(match))
			} else {
				slog.Debugf("match-kratos-random: operation=%s side=%v match=%d skip -- random skipped", operation, cfg.routeScope.Side, utils.BooleanToNum(match))
			}
		}
		return match
	}
}
