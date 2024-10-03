package slowkratoshandle

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/orzkratos/authkratos/authkratospath"
	"github.com/orzkratos/authkratos/internal/utils"
)

type SlowFastConfig struct {
	fastTimeoutGap time.Duration //快速超时的时间
	fastOperations []authkratospath.Path
	slowOperations []authkratospath.Path
}

func NewSlowFastConfig(
	fastTimeoutGap time.Duration,
	fastOperations authkratospath.Paths,
	slowOperations authkratospath.Paths,
) *SlowFastConfig {
	return &SlowFastConfig{
		fastTimeoutGap: fastTimeoutGap,
		fastOperations: fastOperations,
		slowOperations: slowOperations,
	}
}

// NewMiddleware 有时接口分为快速返回和耗时返回两种，我们可以单独设置它们的timeout时间，否则假如把超时都设置为10分钟，则某些小接口卡住时也不行
func NewMiddleware(cfg *SlowFastConfig, LOGGER log.Logger) middleware.Middleware {
	LOG := log.NewHelper(LOGGER)
	LOG.Infof(
		"new slow_fast middleware slow=%v fast=%v fast_timeout=%v",
		len(cfg.slowOperations),
		len(cfg.fastOperations),
		cfg.fastTimeoutGap,
	)

	return selector.Server(middlewareFunc(cfg)).Match(matchFunc(cfg, LOGGER)).Build()
}

func matchFunc(cfg *SlowFastConfig, LOGGER log.Logger) selector.MatchFunc {
	LOG := log.NewHelper(LOGGER)
	qMap := utils.MpKB(cfg.fastOperations)
	sMap := utils.MpKB(cfg.slowOperations)
	return func(ctx context.Context, operation string) bool {
		path := authkratospath.New(operation)
		if qMap[path] {
			LOG.Debugf("operation=%s slow_fast_middleware [fast]", operation)
			return true
		} else if sMap[path] {
			LOG.Debugf("operation=%s slow_fast_middleware [slow]", operation)
			return false
		} else {
			LOG.Debugf("operation=%s slow_fast_middleware [soon]", operation)
			return true
		}
	}
}

func middlewareFunc(cfg *SlowFastConfig) middleware.Middleware {
	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			//设置新超时时间，因此需要外面的超时时间更长些，选择部分接口设置快速超时
			ctx, can := context.WithTimeout(ctx, cfg.fastTimeoutGap)
			defer can()
			return handleFunc(ctx, req)
		}
	}
}