package authkratossimple

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/orzkratos/authkratos"
	"github.com/orzkratos/authkratos/authkratosroutes"
	"github.com/orzkratos/authkratos/internal/utils"
	"github.com/yyle88/neatjson/neatjsons"
	"go.elastic.co/apm/v2"
)

type CheckFunc func(ctx context.Context, token string) (context.Context, *errors.Error)

type Config struct {
	fieldName  string // 注意配置时不要配置非标准的 (Nginx 默认忽略带有下划线的 headers 信息，除非配置 underscores_in_headers on; 因此在开发中建议不要配置含特殊字符的字段名)
	selectPath *authkratosroutes.SelectPath
	checkMatch CheckFunc
	debugMode  bool
}

func NewConfig(selectPath *authkratosroutes.SelectPath, checkMatch CheckFunc) *Config {
	return &Config{
		fieldName:  "Authorization",
		selectPath: selectPath,
		checkMatch: checkMatch,
		debugMode:  authkratos.GetDebugMode(),
	}
}

// WithFieldName 设置请求头中用于认证的字段名
// 注意配置时不要配置非标准的 (Nginx 默认忽略带有下划线的 headers 信息，除非配置 underscores_in_headers on; 因此在开发中建议不要配置含特殊字符的字段名)
func (c *Config) WithFieldName(fieldName string) *Config {
	c.fieldName = fieldName
	return c
}

func (c *Config) WithDebugMode(debugMode bool) *Config {
	c.debugMode = debugMode
	return c
}

func NewMiddleware(cfg *Config, logger log.Logger) middleware.Middleware {
	LOG := log.NewHelper(logger)
	LOG.Infof(
		"auth-kratos-simple: new middleware field-name=%v select-side=%v operations=%v debug-mode=%v",
		cfg.fieldName,
		cfg.selectPath.SelectSide,
		len(cfg.selectPath.Operations),
		cfg.debugMode,
	)
	if cfg.debugMode {
		LOG.Debugf("auth-kratos-simple: new middleware field-name=%v select-path: %s", cfg.fieldName, neatjsons.S(cfg.selectPath))
	}
	return selector.Server(middlewareFunc(cfg, logger)).Match(matchFunc(cfg, logger)).Build()
}

func matchFunc(cfg *Config, logger log.Logger) selector.MatchFunc {
	LOG := log.NewHelper(logger)

	return func(ctx context.Context, operation string) bool {
		match := cfg.selectPath.Match(operation)
		if cfg.debugMode {
			if match {
				LOG.Debugf("auth-kratos-simple: operation=%s include=%v match=%d next -> check auth", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match))
			} else {
				LOG.Debugf("auth-kratos-simple: operation=%s include=%v match=%d skip -- check auth", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match))
			}
		}
		return match
	}
}

func middlewareFunc(cfg *Config, logger log.Logger) middleware.Middleware {
	LOG := log.NewHelper(logger)

	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if tsp, ok := transport.FromServerContext(ctx); ok {
				apmTx := apm.TransactionFromContext(ctx)
				span := apmTx.StartSpan("auth-kratos-simple", "auth", nil)
				defer span.End()

				authToken := tsp.RequestHeader().Get(cfg.fieldName)
				if authToken == "" {
					if cfg.debugMode {
						LOG.Debugf("auth-kratos-simple: auth-token is missing")
					}
					return nil, errors.Unauthorized("UNAUTHORIZED", "auth-kratos-simple: auth-token is missing")
				}
				ctx, erk := cfg.checkMatch(ctx, authToken)
				if erk != nil {
					if cfg.debugMode {
						LOG.Debugf("auth-kratos-simple: auth-token mismatch: %s", erk.Error())
					}
					return nil, erk
				}
				return handleFunc(ctx, req)
			}
			return nil, errors.Unauthorized("UNAUTHORIZED", "auth-kratos-simple: wrong context")
		}
	}
}
