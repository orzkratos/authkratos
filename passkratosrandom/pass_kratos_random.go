package passkratosrandom

import (
	"context"
	"math/rand"
	"net/http"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/orzkratos/authkratos"
	"github.com/orzkratos/authkratos/authkratosroutes"
	"github.com/orzkratos/authkratos/internal/utils"
	"github.com/yyle88/neatjson/neatjsons"
)

type Config struct {
	selectPath *authkratosroutes.SelectPath
	rate       float64
	debugMode  bool
}

func NewConfig(selectPath *authkratosroutes.SelectPath, passRate float64) *Config {
	return &Config{
		selectPath: selectPath,
		rate:       passRate,
		debugMode:  authkratos.GetDebugMode(),
	}
}

// NewMiddleware 让接口有一定概率失败
func NewMiddleware(cfg *Config, logger log.Logger) middleware.Middleware {
	LOG := log.NewHelper(logger)
	LOG.Infof(
		"pass-kratos-random: new middleware include=%s operations=%d rate=%v",
		cfg.selectPath.SelectSide,
		len(cfg.selectPath.Operations),
		cfg.rate,
	)
	if cfg.debugMode {
		LOG.Debugf("pass-kratos-random: new middleware select-path: %s", neatjsons.S(cfg.selectPath))
	}
	return selector.Server(middlewareFunc(cfg, logger)).Match(matchFunc(cfg, logger)).Build()
}

func matchFunc(cfg *Config, logger log.Logger) selector.MatchFunc {
	LOG := log.NewHelper(logger)

	return func(ctx context.Context, operation string) bool {
		if match := cfg.selectPath.Match(operation); !match {
			if cfg.debugMode {
				LOG.Debugf("pass-kratos-random: operation=%s include=%v match=%d next -> skip random", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match))
			}
			return false
		}
		skipBlock := rand.Float64() < cfg.rate //设置0.6就是有60%的概率通过

		match := !skipBlock //是否进入拦截器，拦截器会拦截请求，因此这里求逆值
		if cfg.debugMode {
			if match {
				LOG.Debugf("pass-kratos-random: operation=%s include=%v match=%d next -> goto unavailable", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match))
			} else {
				LOG.Debugf("pass-kratos-random: operation=%s include=%v match=%d skip -- skip unavailable", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match))
			}
		}
		return match
	}
}

func middlewareFunc(cfg *Config, logger log.Logger) middleware.Middleware {
	LOG := log.NewHelper(logger)

	erk := errors.New(http.StatusServiceUnavailable, "RANDOM_RATE_MATCH_UNAVAILABLE", "random match unavailable")

	//当已经命中概率的时候，就直接返回错误
	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if cfg.debugMode {
				LOG.Debugf("pass-kratos-random: random match unavailable")
			}
			return nil, erk
		}
	}
}
