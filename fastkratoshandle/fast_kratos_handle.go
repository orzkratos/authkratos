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
	"github.com/yyle88/neatjson/neatjsons"
)

type Config struct {
	newTimeout time.Duration //快速超时的时间
	selectPath *authkratosroutes.SelectPath
	debugMode  bool
}

func NewConfig(
	newTimeout time.Duration,
	selectPath *authkratosroutes.SelectPath,
) *Config {
	return &Config{
		newTimeout: newTimeout,
		selectPath: selectPath,
		debugMode:  authkratos.GetDebugMode(),
	}
}

func (c *Config) WithDebugMode(debugMode bool) *Config {
	c.debugMode = debugMode
	return c
}

// NewMiddleware 这个函数得到个middleware让某些接口具有更短的超时时间
// 但现实中我们遇到的问题往往是需要延长某个接口的超时时间
// 这样“设置更长超时时间”的需求更常见，以下是解决的思路
// 由于 ctx 的超时时间只能缩短而不能延长，因此整个设计是用“排除法过滤”，就是先给整个服务的接口配置很长的超时时间，再限制其余接口的超时时间为更短的时间
// 配置时使用 "EXCLUDE" 排除这些接口，其它的都是快速超时的
// 即可满足“设置更长超时时间”的需求
func NewMiddleware(cfg *Config, logger log.Logger) middleware.Middleware {
	LOG := log.NewHelper(logger)
	LOG.Infof(
		"fast-kratos-handle: new middleware include=%s operations=%d new-timeout=%v debug-mode=%v",
		cfg.selectPath.SelectSide,
		len(cfg.selectPath.Operations),
		cfg.newTimeout,
		cfg.debugMode,
	)
	if cfg.debugMode {
		LOG.Debugf("fast-kratos-handle: new middleware select-path: %s", neatjsons.S(cfg.selectPath))
	}
	return selector.Server(middlewareFunc(cfg, logger)).Match(matchFunc(cfg, logger)).Build()
}

func matchFunc(cfg *Config, logger log.Logger) selector.MatchFunc {
	LOG := log.NewHelper(logger)

	return func(ctx context.Context, operation string) bool {
		match := cfg.selectPath.Match(operation)
		if cfg.debugMode {
			if match {
				LOG.Debugf("fast-kratos-handle: operation=%s include=%v match=%d next -> fast-handle", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match))
			} else {
				LOG.Debugf("fast-kratos-handle: operation=%s include=%v match=%d skip -- slow-handle", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match))
			}
		}
		return match
	}
}

func middlewareFunc(cfg *Config, logger log.Logger) middleware.Middleware {
	LOG := log.NewHelper(logger)

	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			//设置新超时时间，由于 ctx 是所有超时时间里取最短的，因此只能缩短而不能延长，因此需要选择快速超时的
			ctx, can := context.WithTimeout(ctx, cfg.newTimeout)
			defer can()
			if cfg.debugMode {
				LOG.Debugf("fast-kratos-handle: context with new-timeout=%v fast-handle", cfg.newTimeout)
			}
			return handleFunc(ctx, req)
		}
	}
}
