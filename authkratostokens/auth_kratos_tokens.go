package authkratostokens

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/orzkratos/authkratos"
	"github.com/orzkratos/authkratos/authkratosroutes"
	"github.com/orzkratos/authkratos/internal/utils"
	"github.com/yyle88/must"
	"github.com/yyle88/neatjson/neatjsons"
	"go.elastic.co/apm/v2"
	"golang.org/x/exp/maps"
)

type Config struct {
	fieldName  string
	authTokens map[string]string
	selectPath *authkratosroutes.SelectPath
	debugMode  bool
}

func NewConfig(authTokens map[string]string, selectPath *authkratosroutes.SelectPath) *Config {
	return &Config{
		fieldName:  "Authorization", // 注意配置时不要配置非标准的 (Nginx 默认忽略带有下划线的 headers 信息，除非配置 underscores_in_headers on; 因此在开发中建议不要配置含特殊字符的字段名)
		authTokens: authTokens,
		selectPath: selectPath,
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

func (c *Config) GetAuthTokens() map[string]string {
	if c != nil {
		return c.authTokens
	}
	return nil
}

func (c *Config) CreateToken(username string) string {
	password, ok := c.GetAuthTokens()[username]
	must.TRUE(ok)
	must.Nice(password)
	return utils.BasicAuth(username, password)
}

func (c *Config) GetOneToken() string {
	return c.CreateToken(utils.Sample(maps.Keys(c.GetAuthTokens())))
}

func (c *Config) GetMapTokens() map[string]string {
	var res = make(map[string]string, len(c.GetAuthTokens()))
	for username, password := range c.GetAuthTokens() {
		res[username] = utils.BasicAuth(username, password)
	}
	return res
}

func NewMiddleware(cfg *Config, logger log.Logger) middleware.Middleware {
	LOG := log.NewHelper(logger)
	LOG.Infof(
		"auth-kratos-tokens: new middleware field-name=%v auth-tokens=%d include=%v operations=%d",
		cfg.fieldName,
		len(cfg.authTokens),
		cfg.selectPath.SelectSide,
		len(cfg.selectPath.Operations),
	)
	if cfg.debugMode {
		LOG.Debugf("auth-kratos-tokens: new middleware field-name=%v select-path: %s", cfg.fieldName, neatjsons.S(cfg.selectPath))
	}
	return selector.Server(middlewareFunc(cfg, logger)).Match(matchFunc(cfg, logger)).Build()
}

func matchFunc(cfg *Config, logger log.Logger) selector.MatchFunc {
	LOG := log.NewHelper(logger)

	return func(ctx context.Context, operation string) bool {
		match := cfg.selectPath.Match(operation)
		if cfg.debugMode {
			if match {
				LOG.Debugf("auth-kratos-tokens: operation=%s include=%v match=%d next -> check auth", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match))
			} else {
				LOG.Debugf("auth-kratos-tokens: operation=%s include=%v match=%d skip -- check auth", operation, cfg.selectPath.SelectSide, utils.BooleanToNum(match))
			}
		}
		return match
	}
}

func middlewareFunc(cfg *Config, logger log.Logger) middleware.Middleware {
	LOG := log.NewHelper(logger)

	mapBox := &authTokenMapBox{
		mapToken: newMapToken(cfg.authTokens),
		mapBasic: newMapBasic(cfg.authTokens),
	}

	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if tsp, ok := transport.FromServerContext(ctx); ok {
				apmTx := apm.TransactionFromContext(ctx)
				span := apmTx.StartSpan("auth-kratos-tokens", "auth", nil)
				defer span.End()

				var authToken = tsp.RequestHeader().Get(cfg.fieldName)
				if authToken == "" {
					if cfg.debugMode {
						LOG.Debugf("auth-kratos-tokens: auth-token is missing")
					}
					return nil, errors.Unauthorized("UNAUTHORIZED", "auth-kratos-tokens: auth-token is missing")
				}
				username, erk := checkAuthToken(cfg, mapBox, authToken, LOG)
				if erk != nil {
					if cfg.debugMode {
						LOG.Debugf("auth-kratos-tokens: auth-token mismatch: %s", erk.Error())
					}
					return nil, erk
				}
				ctx = setContextWithUsername(ctx, username)
				return handleFunc(ctx, req)
			}
			return nil, errors.Unauthorized("UNAUTHORIZED", "auth-kratos-tokens: wrong context")
		}
	}
}

func checkAuthToken(cfg *Config, mapBox *authTokenMapBox, token string, LOG *log.Helper) (string, *errors.Error) {
	if username, ok := mapBox.mapToken[token]; ok {
		if cfg.debugMode {
			LOG.Debugf("auth-kratos-tokens: raw-s-token request username:%v quick pass", username)
		}
		return username, nil
	}
	if username, ok := mapBox.mapBasic[token]; ok {
		if cfg.debugMode {
			LOG.Debugf("auth-kratos-tokens: basic-token request username:%v quick pass", username)
		}
		return username, nil
	}
	return "", errors.Unauthorized("UNAUTHORIZED", "auth-kratos-tokens: auth-token mismatch")
}

type authTokenMapBox struct {
	mapToken map[string]string
	mapBasic map[string]string
}

func newMapToken(authTokens map[string]string) map[string]string {
	var mapToken = make(map[string]string, len(authTokens))
	for acc, pwd := range authTokens {
		mapToken[pwd] = acc
	}
	return mapToken
}

func newMapBasic(authTokens map[string]string) map[string]string {
	var mapBasic = map[string]string{}
	for username, token := range authTokens {
		s := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, token)))
		v := "Basic " + string(s)
		mapBasic[v] = username
	}
	return mapBasic
}

type usernameKey struct{}

func setContextWithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey{}, username)
}

func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(usernameKey{}).(string)
	return username, ok
}

func GetUsername(ctx context.Context) (string, bool) {
	return GetUsernameFromContext(ctx)
}
