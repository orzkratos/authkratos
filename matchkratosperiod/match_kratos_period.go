// Package matchkratosperiod: Periodic request sampling middleware with count-based throttling
// Provides predictive request sampling with configurable period count
// Passes each Nth request while blocking others with first-match support if needed
// Good fit in load reduction, sampling, and managed traffic shaping scenarios
//
// matchkratosperiod: 周期性请求采样中间件，基于计数器的限流
// 提供可配置周期数的确定性请求采样
// 每 N 个请求放行一个，其余阻断，支持可选的首次匹配
// 适用于负载降低、采样和受控流量整形场景
package matchkratosperiod

import (
	"context"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/orzkratos/authkratos"
	"github.com/orzkratos/authkratos/authkratosroutes"
	"github.com/orzkratos/authkratos/internal/utils"
	"github.com/yyle88/must"
	"github.com/yyle88/neatjson/neatjsons"
	"github.com/yyle88/syncmap"
	"go.elastic.co/apm/v2"
)

type Config struct {
	routeScope     *authkratosroutes.RouteScope
	n              uint32
	matchFirst     bool
	apmSpanName    string // APM span 名称，为空时不启动 APM 追踪
	apmMatchSuffix string // APM match span 后缀，默认为 -match
	debugMode      bool
}

func NewConfig(routeScope *authkratosroutes.RouteScope, n uint32) *Config {
	return &Config{
		routeScope:     routeScope,
		n:              n,
		matchFirst:     true,
		apmSpanName:    "",
		apmMatchSuffix: "-match", // 默认后缀
		debugMode:      authkratos.GetDebugMode(),
	}
}

func (c *Config) WithMatchFirst(matchFirst bool) *Config {
	c.matchFirst = matchFirst
	return c
}

func (c *Config) WithDebugMode(debugMode bool) *Config {
	c.debugMode = debugMode
	return c
}

// WithDefaultApmSpanName sets default APM span name
// Default name: match-kratos-period
//
// WithDefaultApmSpanName 使用默认的 APM span 名称
// 默认名称: match-kratos-period
func (c *Config) WithDefaultApmSpanName() *Config {
	return c.WithApmSpanName("match-kratos-period")
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

func NewMatchFunc(cfg *Config, logger log.Logger) selector.MatchFunc {
	slog := log.NewHelper(logger)
	slog.Infof("match-kratos-period: new match func side=%v operations=%d match-first=%v period=%v", cfg.routeScope.Side, len(cfg.routeScope.OperationSet), cfg.matchFirst, cfg.n)
	if cfg.debugMode {
		slog.Debugf("match-kratos-period: new match func route-scope: %s", neatjsons.S(cfg.routeScope))
	}

	type countBox struct {
		mutex *sync.Mutex
		count uint64
	}
	mp := syncmap.New[authkratosroutes.Operation, *countBox]()
	return func(ctx context.Context, operation string) bool {
		// 如果配置了 APM span 名称，则启动 APM 追踪
		if cfg.apmSpanName != "" {
			apmTx := apm.TransactionFromContext(ctx)
			span := apmTx.StartSpan(cfg.apmSpanName+cfg.apmMatchSuffix, "app", nil)
			defer span.End()
		}

		if match := cfg.routeScope.Match(operation); !match {
			if cfg.debugMode {
				slog.Debugf("match-kratos-period: operation=%s side=%v match=%d next -> skip period", operation, cfg.routeScope.Side, utils.BooleanToNum(match))
			}
			return false
		}
		value, loaded := mp.LoadOrStore(operation, &countBox{&sync.Mutex{}, 0})
		if !loaded && cfg.matchFirst {
			if cfg.debugMode {
				slog.Debugf("match-kratos-period: operation=%s side=%v match=%d next -> match first (count=0)", operation, cfg.routeScope.Side, utils.BooleanToNum(true))
			}
			return true
		}
		value.mutex.Lock()
		value.count = (value.count + 1) % uint64(max(cfg.n, 1))
		count := value.count
		value.mutex.Unlock()
		match := count == 0
		if cfg.debugMode {
			if match {
				slog.Debugf("match-kratos-period: operation=%s side=%v match=%d next -> period matched (count=%d)", operation, cfg.routeScope.Side, utils.BooleanToNum(match), count)
			} else {
				slog.Debugf("match-kratos-period: operation=%s side=%v match=%d skip -- period skipped (count=%d)", operation, cfg.routeScope.Side, utils.BooleanToNum(match), count)
			}
		}
		return match
	}
}
