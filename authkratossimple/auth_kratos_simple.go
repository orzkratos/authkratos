package authkratossimple

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/orzkratos/authkratos/authkratosroutes"
	"github.com/orzkratos/authkratos/internal/utils"
	"go.elastic.co/apm/v2"
)

type CheckFunc func(ctx context.Context, token string) (context.Context, *errors.Error)

type Config struct {
	tokenField string
	selectPath *authkratosroutes.SelectPath
	checkMatch CheckFunc
	enable     bool
}

func NewConfig(tokenField string, checkMatch CheckFunc, selectPath *authkratosroutes.SelectPath) *Config {
	return &Config{
		tokenField: tokenField,
		selectPath: selectPath,
		checkMatch: checkMatch,
		enable:     true,
	}
}

func (a *Config) GetTokenField() string {
	return a.tokenField
}

func (a *Config) SetEnable(enable bool) {
	a.enable = enable
}

func (a *Config) GetEnable() bool {
	if a != nil {
		return a.enable && a.tokenField != ""
	}
	return false
}

func NewMiddleware(cfg *Config, LOGGER log.Logger) middleware.Middleware {
	LOG := log.NewHelper(LOGGER)
	LOG.Infof(
		"new check_auth middleware enable=%v tokenField=%v simple=x include=%v operations=%v",
		cfg.GetEnable(),
		cfg.tokenField,
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
			LOG.Debugf("operation=%s include=%v match=%d next -> check auth", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match))
		} else {
			LOG.Debugf("operation=%s include=%v match=%d skip -- check auth", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match))
		}
		return match
	}
}

func middlewareFunc(cfg *Config, LOGGER log.Logger) middleware.Middleware {
	LOG := log.NewHelper(LOGGER)

	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if !cfg.GetEnable() {
				LOG.Infof("auth_kratos_simple: cfg.enable=false anyone can pass")
				return handleFunc(ctx, req)
			}
			if tsp, ok := transport.FromServerContext(ctx); ok {
				apmTx := apm.TransactionFromContext(ctx)
				sp := apmTx.StartSpan("auth_kratos_simple", "auth", nil)
				defer sp.End()

				token := tsp.RequestHeader().Get(cfg.tokenField)
				if token == "" {
					return nil, errors.Unauthorized("UNAUTHORIZED", "auth_kratos_simple: auth token is missing")
				}
				ctx, erk := cfg.checkMatch(ctx, token)
				if erk != nil {
					return nil, erk
				}
				return handleFunc(ctx, req)
			}
			return nil, errors.Unauthorized("UNAUTHORIZED", "auth_kratos_simple: wrong context for middleware")
		}
	}
}
