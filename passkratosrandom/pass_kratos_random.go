package passkratosrandom

import (
	"context"
	"math/rand"
	"net/http"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/orzkratos/authkratos/authkratospath"
)

type Config struct {
	rateMap map[authkratospath.Path]float64
	rate    float64
	enable  bool
}

func NewConfig(
	rateMap map[authkratospath.Path]float64,
	rate float64,
) *Config {
	return &Config{
		rateMap: rateMap,
		rate:    rate,
		enable:  true,
	}
}

func (a *Config) SetEnable(v bool) {
	a.enable = v
}

func (a *Config) IsEnable() bool {
	if a != nil {
		return a.enable
	}
	return false
}

// NewMiddleware 让接口有一定概率失败
func NewMiddleware(cfg *Config, LOGGER log.Logger) middleware.Middleware {
	LOG := log.NewHelper(LOGGER)
	LOG.Infof(
		"new rate_pass middleware enable=%v operations=%v rate=%v",
		cfg.IsEnable(),
		len(cfg.rateMap),
		cfg.rate,
	)

	return selector.Server(middlewareFunc()).Match(matchFunc(cfg, LOGGER)).Build()
}

func matchFunc(cfg *Config, LOGGER log.Logger) selector.MatchFunc {
	LOG := log.NewHelper(LOGGER)

	return func(ctx context.Context, operation string) bool {
		if !cfg.enable {
			return false
		}
		if len(cfg.rateMap) > 0 {
			path := authkratospath.New(operation)
			if rate, ok := cfg.rateMap[path]; ok {
				pass := rand.Float64() < rate //比如设置0.6就是有60%的概率通过
				LOG.Debugf("operation=%s in rate_map rate_pass rate=%v pass=%v", operation, rate, pass)
				return !pass
			}
		}
		//这里不是else，而是默认的，就是没配置通过率的，就是用这个默认的通过率
		pass := rand.Float64() < cfg.rate //设置0.6就是有60%的概率通过
		LOG.Debugf("operation=%s rate_pass rate=%v pass=%v", operation, cfg.rate, pass)
		return !pass //当不通过时才执行 middlewareFunc
	}
}

func middlewareFunc() middleware.Middleware {
	erk := errors.New(http.StatusServiceUnavailable, "RANDOM_RATE_NOT_PASS", "random rate not pass")

	//当已经命中概率的时候，就直接返回错误
	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, erk
		}
	}
}
