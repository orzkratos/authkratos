package authkratosroutes

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/orzkratos/authkratos"
	"github.com/orzkratos/authkratos/internal/utils"
	"github.com/yyle88/must"
	"github.com/yyle88/neatjson/neatjsons"
	"go.elastic.co/apm/v2"
)

// Config holds the match function config
// Config 保存匹配函数的配置
type Config struct {
	actionName     string      // Action name for logging // 用于日志的动作名称
	routeScope     *RouteScope // Route scope to match // 要匹配的路由范围
	apmSpanName    string      // APM span name, empty to disable APM tracing // APM span 名称，为空时不启动 APM 追踪
	apmMatchSuffix string      // APM match span suffix, default is -match // APM match span 后缀，默认为 -match
	debugMode      bool        // Debug mode flag // 调试模式标志
}

// NewConfig creates a new match config
// NewConfig 创建新的匹配配置
func NewConfig(actionName string, routeScope *RouteScope) *Config {
	return &Config{
		actionName:     actionName,
		routeScope:     routeScope,
		apmSpanName:    "",
		apmMatchSuffix: "-match", // Default suffix // 默认后缀
		debugMode:      authkratos.GetDebugMode(),
	}
}

// WithDebugMode sets debug mode
// WithDebugMode 设置调试模式
func (c *Config) WithDebugMode(debugMode bool) *Config {
	c.debugMode = debugMode
	return c
}

// WithDefaultApmSpanName sets default APM span name
// Default name: auth-kratos-routes
//
// WithDefaultApmSpanName 使用默认的 APM span 名称
// 默认名称: auth-kratos-routes
func (c *Config) WithDefaultApmSpanName() *Config {
	return c.WithApmSpanName("auth-kratos-routes")
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

// NewMatchFunc creates a selector match function with route scope
// NewMatchFunc 创建带路由范围的选择器匹配函数
func NewMatchFunc(cfg *Config, logger log.Logger) selector.MatchFunc {
	slog := log.NewHelper(logger)
	slog.Infof(
		"auth-kratos-routes: new middleware action-name=%v side=%v operations=%d debug-mode=%v",
		cfg.actionName,
		cfg.routeScope.Side,
		len(cfg.routeScope.OperationSet),
		cfg.debugMode,
	)
	if cfg.debugMode {
		slog.Debugf("auth-kratos-routes: new middleware action-name=%v route-scope: %s", cfg.actionName, neatjsons.S(cfg.routeScope))
	}
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
				slog.Debugf("auth-kratos-routes: operation=%s side=%v match=%d next -> %s", operation, cfg.routeScope.Side, utils.BooleanToNum(match), cfg.actionName)
			} else {
				slog.Debugf("auth-kratos-routes: operation=%s side=%v match=%d skip -- %s", operation, cfg.routeScope.Side, utils.BooleanToNum(match), cfg.actionName)
			}
		}
		return match
	}
}
