package authkratostokens

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/orzkratos/authkratos/authkratospath"
	"github.com/orzkratos/authkratos/internal/utils"
	"github.com/yyle88/must"
	"go.elastic.co/apm/v2"
)

type Config struct {
	field      string
	selectPath *authkratospath.SelectPath
	tokens     map[string]string
	enable     bool
}

func NewConfig(field string, tokens map[string]string, selectPath *authkratospath.SelectPath) *Config {
	return &Config{
		field:      field,
		selectPath: selectPath,
		tokens:     tokens,
		enable:     true,
	}
}

func (a *Config) SetEnable(v bool) {
	a.enable = v
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

func (a *Config) GetAuths() map[string]string {
	if a != nil {
		return a.tokens
	}
	return nil
}

func (a *Config) CreateToken(username string) string {
	password, ok := a.GetAuths()[username]
	must.TRUE(ok)
	must.Nice(password)
	return utils.BasicAuth(username, password)
}

func (a *Config) GetOneToken() string {
	if !a.IsEnable() {
		return utils.BasicAuth(utils.NewUUID(), utils.NewUUID())
	} else {
		return a.CreateToken(utils.Sample(utils.Keys(a.GetAuths())))
	}
}

func (a *Config) GetMapTokens() map[string]string {
	if !a.IsEnable() {
		username := utils.NewUUID()
		password := utils.NewUUID()
		return map[string]string{username: utils.BasicAuth(username, password)}
	} else {
		var res = make(map[string]string, len(a.GetAuths()))
		for username, password := range a.GetAuths() {
			res[username] = utils.BasicAuth(username, password)
		}
		return res
	}
}

func NewMiddleware(cfg *Config, LOGGER log.Logger) middleware.Middleware {
	LOG := log.NewHelper(LOGGER)
	LOG.Infof(
		"new check_auth middleware enable=%v field=%v tokens=%v include=%v operations=%v",
		cfg.IsEnable(),
		cfg.field,
		len(cfg.tokens),
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

	var mapToken = make(map[string]string, len(cfg.tokens))
	for acc, pwd := range cfg.tokens {
		mapToken[pwd] = acc
	}
	var mapBasic = map[string]string{}
	for username, token := range cfg.tokens {
		for _, name := range []string{"None", username} { //有些请求没有用户名因此补个None，兼容老的业务
			s := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", name, token)))
			v := "Basic " + string(s)
			mapBasic[v] = username
		}
	}
	return func(handleFunc middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if !cfg.IsEnable() {
				LOG.Infof("check_auth: cfg.enable=false anonymous pass")
				return handleFunc(ctx, req)
			}
			if tp, ok := transport.FromServerContext(ctx); ok {
				tx := apm.TransactionFromContext(ctx)
				sp := tx.StartSpan("check_auth", "auth", nil)
				defer sp.End()

				var token = tp.RequestHeader().Get(cfg.field)
				if token == "" {
					return nil, errors.Unauthorized("UNAUTHORIZED", "check_auth: auth token is missing")
				}
				if username, ok := mapToken[token]; ok {
					LOG.Infof("check_auth: rawToken request username:%v quick pass", username)
				} else if username, ok := mapBasic[token]; ok {
					LOG.Infof("check_auth: BasicToken request username:%v quick pass", username)
				} else {
					var canPass = false
					if messParts := strings.SplitN(token, " ", 2); len(messParts) == 2 {
						messType := messParts[0]
						switch {
						case strings.EqualFold(messType, "Bearer"):
							//暂不需要
						case strings.EqualFold(messType, "Basic"):
							if erk := checkBasicToken(messParts[1], mapToken, LOG); erk != nil {
								return nil, erk
							}
							canPass = true
						}
					}
					if !canPass {
						return nil, errors.Unauthorized("UNAUTHORIZED", "check_auth: auth token is wrong")
					}
				}
				return handleFunc(ctx, req)
			}
			return nil, errors.Unauthorized("UNAUTHORIZED", "check_auth: wrong context for middleware")
		}
	}
}

func checkBasicToken(messBasic string, mapToken map[string]string, LOG *log.Helper) *errors.Error {
	data, err := base64.StdEncoding.DecodeString(messBasic)
	if err != nil {
		return errors.Unauthorized("UNAUTHORIZED", "check_auth: error:"+err.Error())
	}
	rawParts := strings.Split(string(data), ":")
	rawToken := rawParts[1] //前面不报错的话这边必然就能切出元素，其下标不会超出限制
	username, ok := mapToken[rawToken]
	if !ok {
		return errors.Unauthorized("UNAUTHORIZED", "check_auth: auth token is wrong")
	}
	LOG.Infof("check_auth: basic token request username:%v pass", username)
	return nil
}
