package authkratossimple

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/orzkratos/authkratos/authkratospath"
	"go.elastic.co/apm/v2"
)

type Config struct {
	field      string
	selectPath *authkratospath.SelectPath
	check      CheckFunc
	enable     bool
}

type CheckFunc func(ctx context.Context, token string) (context.Context, *errors.Error)

func NewConfig(field string, check CheckFunc, selectPath *authkratospath.SelectPath) *Config {
	return &Config{
		field:      field,
		selectPath: selectPath,
		check:      check,
		enable:     true,
	}
}

func (a *Config) SetEnable(enable bool) {
	a.enable = enable
}

func (a *Config) IsEnable() bool {
	if a != nil {
		return a.enable && a.field != ""
	}
	return false
}

func (a *Config) GetField() string {
	if a != nil {
		return a.field
	}
	return ""
}

func NewMiddleware(cfg *Config, LOGGER log.Logger) middleware.Middleware {
	LOG := log.NewHelper(LOGGER)
	LOG.Infof(
		"new check_auth middleware enable=%v field=%v simple=x include=%v operations=%v",
		cfg.IsEnable(),
		cfg.field,
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
			LOG.Debugf("operation=%s include=%v match=%v must check auth", operation, cfg.selectPath.SelectSide, match)
		} else {
			LOG.Debugf("operation=%s include=%v match=%v skip check auth", operation, cfg.selectPath.SelectSide, match)
		}
		return match
	}
}

func middlewareFunc(cfg *Config, LOGGER log.Logger) middleware.Middleware {
	LOG := log.NewHelper(LOGGER)

	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if !cfg.IsEnable() {
				LOG.Infof("auth_kratos_simple: cfg.enable=false anonymous pass")
				return handleFunc(ctx, req)
			}
			if tp, ok := transport.FromServerContext(ctx); ok {
				apmTx := apm.TransactionFromContext(ctx)
				sp := apmTx.StartSpan("auth_kratos_simple", "auth", nil)
				defer sp.End()

				token := tp.RequestHeader().Get(cfg.field)
				if token == "" {
					return nil, errors.Unauthorized("UNAUTHORIZED", "auth_kratos_simple: auth token is missing")
				}
				ctx, erk := cfg.check(ctx, token)
				if erk != nil {
					return nil, erk
				}
				return handleFunc(ctx, req)
			}
			return nil, errors.Unauthorized("UNAUTHORIZED", "auth_kratos_simple: wrong context for middleware")
		}
	}
}
